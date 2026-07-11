package ocppclient

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/localauth"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/reservation"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

type reservationState struct {
	connectorID int
	idTag       string
	parentIDTag string
	expiry      time.Time
}

type storedChargingProfile struct {
	connectorID int
	profile     *types.ChargingProfile
}

// Reservation profile.
func (s *simulator) OnReserveNow(request *reservation.ReserveNowRequest) (*reservation.ReserveNowConfirmation, error) {
	status := reservation.ReservationStatusAccepted
	updates := make(map[int]core.ChargePointStatus)

	if request.ExpiryDate == nil || !request.ExpiryDate.Time.After(time.Now()) {
		status = reservation.ReservationStatusRejected
	} else {
		s.mu.Lock()
		if existing, ok := s.reservations[request.ReservationId]; ok {
			delete(s.reservations, request.ReservationId)
			if existing.connectorID > 0 {
				updates[existing.connectorID] = s.idleStatusLocked(existing.connectorID)
				s.connectors[existing.connectorID].status = updates[existing.connectorID]
			}
		}

		if request.ConnectorId == 0 {
			available := false
			operative := false
			for id, connector := range s.connectors {
				if connector.availability == core.AvailabilityTypeOperative {
					operative = true
				}
				_, allowed := s.reservationAccessLocked(id, request.IdTag)
				if connector.availability == core.AvailabilityTypeOperative && connector.transactionID == 0 && allowed {
					available = true
					break
				}
			}
			if !available {
				if operative {
					status = reservation.ReservationStatusOccupied
				} else {
					status = reservation.ReservationStatusUnavailable
				}
			}
		} else {
			connector, ok := s.connectors[request.ConnectorId]
			switch {
			case !ok:
				status = reservation.ReservationStatusRejected
			case connector.status == core.ChargePointStatusFaulted:
				status = reservation.ReservationStatusFaulted
			case connector.availability == core.AvailabilityTypeInoperative || connector.status == core.ChargePointStatusUnavailable:
				status = reservation.ReservationStatusUnavailable
			case connector.transactionID > 0 || connector.status == core.ChargePointStatusPreparing || connector.status == core.ChargePointStatusCharging || connector.status == core.ChargePointStatusFinishing:
				status = reservation.ReservationStatusOccupied
			default:
				for _, existing := range s.reservations {
					if existing.connectorID == request.ConnectorId {
						status = reservation.ReservationStatusOccupied
						break
					}
				}
			}
		}

		if status == reservation.ReservationStatusAccepted {
			s.reservations[request.ReservationId] = &reservationState{
				connectorID: request.ConnectorId,
				idTag:       request.IdTag,
				parentIDTag: request.ParentIdTag,
				expiry:      request.ExpiryDate.Time,
			}
			if request.ConnectorId > 0 {
				s.connectors[request.ConnectorId].status = core.ChargePointStatusReserved
				updates[request.ConnectorId] = core.ChargePointStatusReserved
			}
		}
		s.mu.Unlock()
	}

	detail := request.IdTag + " reservation=" + strconv.Itoa(request.ReservationId)
	s.emit("inbound", reservation.ReserveNowFeatureName, string(status), &request.ConnectorId, detail)
	for connectorID, connectorStatus := range updates {
		s.notifyStatusAsync(connectorID, connectorStatus)
	}
	return reservation.NewReserveNowConfirmation(status), nil
}

func (s *simulator) OnCancelReservation(request *reservation.CancelReservationRequest) (*reservation.CancelReservationConfirmation, error) {
	status := reservation.CancelReservationStatusRejected
	connectorID := 0
	connectorStatus := core.ChargePointStatusAvailable

	s.mu.Lock()
	if existing, ok := s.reservations[request.ReservationId]; ok {
		delete(s.reservations, request.ReservationId)
		status = reservation.CancelReservationStatusAccepted
		connectorID = existing.connectorID
		if connectorID > 0 {
			connectorStatus = s.idleStatusLocked(connectorID)
			s.connectors[connectorID].status = connectorStatus
		}
	}
	s.mu.Unlock()

	s.emit("inbound", reservation.CancelReservationFeatureName, string(status), nil, strconv.Itoa(request.ReservationId))
	if connectorID > 0 {
		s.notifyStatusAsync(connectorID, connectorStatus)
	}
	return reservation.NewCancelReservationConfirmation(status), nil
}

func (s *simulator) expireReservations(ctx context.Context) {
	now := time.Now()
	updates := make(map[int]core.ChargePointStatus)
	expired := make([]int, 0)

	s.mu.Lock()
	for reservationID, current := range s.reservations {
		if current.expiry.After(now) {
			continue
		}
		delete(s.reservations, reservationID)
		expired = append(expired, reservationID)
		if current.connectorID > 0 {
			status := s.idleStatusLocked(current.connectorID)
			s.connectors[current.connectorID].status = status
			updates[current.connectorID] = status
		}
	}
	s.mu.Unlock()

	sort.Ints(expired)
	for _, reservationID := range expired {
		s.emit("system", "ReservationExpired", "Expired", nil, strconv.Itoa(reservationID))
	}
	for connectorID, status := range updates {
		if s.cp.IsConnected() {
			_ = s.notifyStatus(ctx, connectorID, status)
		}
	}
}

func (s *simulator) removeReservationsForConnectorLocked(connectorID int) {
	for reservationID, current := range s.reservations {
		if current.connectorID == connectorID {
			delete(s.reservations, reservationID)
		}
	}
}

func (s *simulator) consumeReservationLocked(reservationID int) {
	delete(s.reservations, reservationID)
}

func (s *simulator) idleStatusLocked(connectorID int) core.ChargePointStatus {
	connector := s.connectors[connectorID]
	if connector.availability == core.AvailabilityTypeInoperative {
		return core.ChargePointStatusUnavailable
	}
	for _, current := range s.reservations {
		if current.connectorID == connectorID && current.expiry.After(time.Now()) {
			return core.ChargePointStatusReserved
		}
	}
	return core.ChargePointStatusAvailable
}

func (s *simulator) reservationAccessLocked(connectorID int, idTag string) (int, bool) {
	matchedGlobal := 0
	for reservationID, current := range s.reservations {
		if !current.expiry.After(time.Now()) {
			continue
		}
		if current.connectorID != 0 && current.connectorID != connectorID {
			continue
		}
		matches := idTag == current.idTag || (current.parentIDTag != "" && idTag == current.parentIDTag)
		if current.connectorID == connectorID {
			if matches {
				return reservationID, true
			}
			return 0, false
		}
		if matches {
			matchedGlobal = reservationID
		} else {
			return 0, false
		}
	}
	return matchedGlobal, true
}

func (s *simulator) selectRemoteStartConnectorLocked(requested *int, idTag string) (int, int, bool) {
	if requested != nil {
		connector, ok := s.connectors[*requested]
		if !ok || connector.availability != core.AvailabilityTypeOperative || connector.transactionID != 0 {
			return 0, 0, false
		}
		reservationID, allowed := s.reservationAccessLocked(*requested, idTag)
		if !allowed || (connector.status != core.ChargePointStatusAvailable && connector.status != core.ChargePointStatusReserved) {
			return 0, 0, false
		}
		return *requested, reservationID, true
	}

	for id := 1; id <= s.opts.Connectors; id++ {
		connector := s.connectors[id]
		if connector.availability != core.AvailabilityTypeOperative || connector.transactionID != 0 {
			continue
		}
		reservationID, allowed := s.reservationAccessLocked(id, idTag)
		if allowed && reservationID > 0 && (connector.status == core.ChargePointStatusAvailable || connector.status == core.ChargePointStatusReserved) {
			return id, reservationID, true
		}
	}
	for id := 1; id <= s.opts.Connectors; id++ {
		connector := s.connectors[id]
		if connector.availability != core.AvailabilityTypeOperative || connector.transactionID != 0 || connector.status != core.ChargePointStatusAvailable {
			continue
		}
		reservationID, allowed := s.reservationAccessLocked(id, idTag)
		if allowed {
			return id, reservationID, true
		}
	}
	return 0, 0, false
}

// Local authorization list profile.
func (s *simulator) OnGetLocalListVersion(_ *localauth.GetLocalListVersionRequest) (*localauth.GetLocalListVersionConfirmation, error) {
	s.mu.Lock()
	version := s.localAuthListVersion
	s.mu.Unlock()
	s.emit("inbound", localauth.GetLocalListVersionFeatureName, "Accepted", nil, strconv.Itoa(version))
	return localauth.NewGetLocalListVersionConfirmation(version), nil
}

func (s *simulator) OnSendLocalList(request *localauth.SendLocalListRequest) (*localauth.SendLocalListConfirmation, error) {
	status := localauth.UpdateStatusAccepted
	s.mu.Lock()
	if request.ListVersion <= s.localAuthListVersion {
		status = localauth.UpdateStatusVersionMismatch
	} else {
		seen := make(map[string]struct{}, len(request.LocalAuthorizationList))
		for _, entry := range request.LocalAuthorizationList {
			if _, exists := seen[entry.IdTag]; exists {
				status = localauth.UpdateStatusFailed
				break
			}
			seen[entry.IdTag] = struct{}{}
		}
		if status == localauth.UpdateStatusAccepted {
			if request.UpdateType == localauth.UpdateTypeFull {
				s.localAuthList = make(map[string]*types.IdTagInfo, len(request.LocalAuthorizationList))
			}
			for _, entry := range request.LocalAuthorizationList {
				if entry.IdTagInfo == nil {
					delete(s.localAuthList, entry.IdTag)
					continue
				}
				info := *entry.IdTagInfo
				s.localAuthList[entry.IdTag] = &info
			}
			s.localAuthListVersion = request.ListVersion
		}
	}
	count := len(s.localAuthList)
	s.mu.Unlock()
	detail := fmt.Sprintf("version=%d type=%s entries=%d", request.ListVersion, request.UpdateType, count)
	s.emit("inbound", localauth.SendLocalListFeatureName, string(status), nil, detail)
	return localauth.NewSendLocalListConfirmation(status), nil
}

func (s *simulator) configurationBoolLocked(key string) bool {
	current, ok := s.configuration[key]
	if !ok || current.Value == nil {
		return false
	}
	value, err := strconv.ParseBool(strings.TrimSpace(*current.Value))
	return err == nil && value
}

// Remote trigger profile.
func (s *simulator) OnTriggerMessage(request *remotetrigger.TriggerMessageRequest) (*remotetrigger.TriggerMessageConfirmation, error) {
	status := remotetrigger.TriggerMessageStatusAccepted
	if request.ConnectorId != nil && *request.ConnectorId > 0 {
		s.mu.Lock()
		_, ok := s.connectors[*request.ConnectorId]
		s.mu.Unlock()
		if !ok {
			status = remotetrigger.TriggerMessageStatusRejected
		}
	}

	switch request.RequestedMessage {
	case core.BootNotificationFeatureName,
		core.HeartbeatFeatureName,
		core.MeterValuesFeatureName,
		core.StatusNotificationFeatureName,
		firmware.DiagnosticsStatusNotificationFeatureName,
		firmware.FirmwareStatusNotificationFeatureName:
	default:
		status = remotetrigger.TriggerMessageStatusNotImplemented
	}

	detail := string(request.RequestedMessage)
	s.emit("inbound", remotetrigger.TriggerMessageFeatureName, string(status), request.ConnectorId, detail)
	if status == remotetrigger.TriggerMessageStatusAccepted && s.cp.IsConnected() {
		connectorID := request.ConnectorId
		go func() {
			timer := time.NewTimer(25 * time.Millisecond)
			defer timer.Stop()
			select {
			case <-s.runContext().Done():
				return
			case <-timer.C:
				s.executeTrigger(request.RequestedMessage, connectorID)
			}
		}()
	}
	return remotetrigger.NewTriggerMessageConfirmation(status), nil
}

func (s *simulator) executeTrigger(message remotetrigger.MessageTrigger, connectorID *int) {
	ctx := s.runContext()
	var err error
	switch message {
	case core.BootNotificationFeatureName:
		_, err = s.sendBootNotification(ctx)
	case core.HeartbeatFeatureName:
		err = s.sendHeartbeat(ctx)
	case core.StatusNotificationFeatureName:
		err = s.sendTriggeredStatuses(ctx, connectorID)
	case core.MeterValuesFeatureName:
		err = s.sendTriggeredMeterValues(ctx, connectorID)
	case firmware.DiagnosticsStatusNotificationFeatureName:
		s.mu.Lock()
		status := s.diagnosticsStatus
		s.mu.Unlock()
		err = s.sendDiagnosticsStatus(ctx, status)
	case firmware.FirmwareStatusNotificationFeatureName:
		s.mu.Lock()
		status := s.firmwareStatus
		s.mu.Unlock()
		err = s.sendFirmwareStatus(ctx, status)
	}
	if err != nil {
		s.emit("outbound", string(message), "Error", connectorID, err.Error())
	}
}

func (s *simulator) sendTriggeredStatuses(ctx context.Context, connectorID *int) error {
	if connectorID == nil {
		if err := s.notifyStatus(ctx, 0, core.ChargePointStatusAvailable); err != nil {
			return err
		}
		for id := 1; id <= s.opts.Connectors; id++ {
			s.mu.Lock()
			status := s.connectors[id].status
			s.mu.Unlock()
			if err := s.notifyStatus(ctx, id, status); err != nil {
				return err
			}
		}
		return nil
	}
	if *connectorID == 0 {
		return s.notifyStatus(ctx, 0, core.ChargePointStatusAvailable)
	}
	s.mu.Lock()
	connector, ok := s.connectors[*connectorID]
	status := core.ChargePointStatusAvailable
	if ok {
		status = connector.status
	}
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("unknown connector %d", *connectorID)
	}
	return s.notifyStatus(ctx, *connectorID, status)
}

func (s *simulator) sendTriggeredMeterValues(ctx context.Context, connectorID *int) error {
	ids := make([]int, 0, s.opts.Connectors)
	if connectorID == nil || *connectorID == 0 {
		for id := 1; id <= s.opts.Connectors; id++ {
			ids = append(ids, id)
		}
	} else {
		ids = append(ids, *connectorID)
	}
	for _, id := range ids {
		s.mu.Lock()
		connector, ok := s.connectors[id]
		meter, transactionID := 0, 0
		if ok {
			meter = connector.meter
			transactionID = connector.transactionID
		}
		s.mu.Unlock()
		if !ok {
			return fmt.Errorf("unknown connector %d", id)
		}
		if err := s.sendMeterSample(ctx, id, meter, transactionID, types.ReadingContextTrigger); err != nil {
			return err
		}
	}
	return nil
}

// Firmware management profile.
func (s *simulator) OnGetDiagnostics(request *firmware.GetDiagnosticsRequest) (*firmware.GetDiagnosticsConfirmation, error) {
	fileName := "ocpp-cli-diagnostics-" + time.Now().UTC().Format("20060102T150405Z") + ".log"
	confirmation := firmware.NewGetDiagnosticsConfirmation()
	confirmation.FileName = fileName
	s.emit("inbound", firmware.GetDiagnosticsFeatureName, "Accepted", nil, request.Location+"/"+fileName)
	if s.cp.IsConnected() {
		go s.simulateDiagnosticsUpload()
	}
	return confirmation, nil
}

func (s *simulator) simulateDiagnosticsUpload() {
	ctx := s.runContext()
	if err := s.sendDiagnosticsStatus(ctx, firmware.DiagnosticsStatusUploading); err != nil {
		s.emit("outbound", firmware.DiagnosticsStatusNotificationFeatureName, "Error", nil, err.Error())
		return
	}
	if !waitContext(ctx, 150*time.Millisecond) {
		return
	}
	if err := s.sendDiagnosticsStatus(ctx, firmware.DiagnosticsStatusUploaded); err != nil {
		s.emit("outbound", firmware.DiagnosticsStatusNotificationFeatureName, "Error", nil, err.Error())
	}
}

func (s *simulator) sendDiagnosticsStatus(ctx context.Context, status firmware.DiagnosticsStatus) error {
	s.mu.Lock()
	s.diagnosticsStatus = status
	s.mu.Unlock()
	_, err := await(ctx, func() (*firmware.DiagnosticsStatusNotificationConfirmation, error) {
		return s.cp.DiagnosticsStatusNotification(status)
	})
	if err != nil {
		return classify(err, ErrProtocol)
	}
	s.emit("outbound", firmware.DiagnosticsStatusNotificationFeatureName, string(status), nil, "")
	return nil
}

func (s *simulator) OnUpdateFirmware(request *firmware.UpdateFirmwareRequest) (*firmware.UpdateFirmwareConfirmation, error) {
	retrieve := time.Now()
	if request.RetrieveDate != nil {
		retrieve = request.RetrieveDate.Time
	}
	s.emit("inbound", firmware.UpdateFirmwareFeatureName, "Accepted", nil, request.Location+" retrieve="+retrieve.UTC().Format(time.RFC3339))
	if s.cp.IsConnected() {
		go s.simulateFirmwareUpdate(retrieve)
	}
	return firmware.NewUpdateFirmwareConfirmation(), nil
}

func (s *simulator) simulateFirmwareUpdate(retrieve time.Time) {
	ctx := s.runContext()
	if delay := time.Until(retrieve); delay > 0 && !waitContext(ctx, delay) {
		return
	}
	statuses := []firmware.FirmwareStatus{
		firmware.FirmwareStatusDownloading,
		firmware.FirmwareStatusDownloaded,
		firmware.FirmwareStatusInstalling,
		firmware.FirmwareStatusInstalled,
	}
	for index, status := range statuses {
		if index > 0 && !waitContext(ctx, 150*time.Millisecond) {
			return
		}
		if err := s.sendFirmwareStatus(ctx, status); err != nil {
			s.emit("outbound", firmware.FirmwareStatusNotificationFeatureName, "Error", nil, err.Error())
			return
		}
	}
}

func (s *simulator) sendFirmwareStatus(ctx context.Context, status firmware.FirmwareStatus) error {
	s.mu.Lock()
	s.firmwareStatus = status
	s.mu.Unlock()
	_, err := await(ctx, func() (*firmware.FirmwareStatusNotificationConfirmation, error) {
		return s.cp.FirmwareStatusNotification(status)
	})
	if err != nil {
		return classify(err, ErrProtocol)
	}
	s.emit("outbound", firmware.FirmwareStatusNotificationFeatureName, string(status), nil, "")
	return nil
}

func waitContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// Smart charging profile.
func (s *simulator) OnSetChargingProfile(request *smartcharging.SetChargingProfileRequest) (*smartcharging.SetChargingProfileConfirmation, error) {
	status := smartcharging.ChargingProfileStatusAccepted
	if request.ChargingProfile == nil {
		status = smartcharging.ChargingProfileStatusRejected
	} else if request.ConnectorId > 0 {
		s.mu.Lock()
		_, ok := s.connectors[request.ConnectorId]
		s.mu.Unlock()
		if !ok {
			status = smartcharging.ChargingProfileStatusRejected
		}
	}
	if status == smartcharging.ChargingProfileStatusAccepted {
		s.mu.Lock()
		s.chargingProfiles[request.ChargingProfile.ChargingProfileId] = storedChargingProfile{connectorID: request.ConnectorId, profile: request.ChargingProfile}
		s.mu.Unlock()
	}
	detail := ""
	if request.ChargingProfile != nil {
		detail = fmt.Sprintf("id=%d stack=%d purpose=%s", request.ChargingProfile.ChargingProfileId, request.ChargingProfile.StackLevel, request.ChargingProfile.ChargingProfilePurpose)
	}
	s.emit("inbound", smartcharging.SetChargingProfileFeatureName, string(status), &request.ConnectorId, detail)
	return smartcharging.NewSetChargingProfileConfirmation(status), nil
}

func (s *simulator) OnClearChargingProfile(request *smartcharging.ClearChargingProfileRequest) (*smartcharging.ClearChargingProfileConfirmation, error) {
	removed := 0
	s.mu.Lock()
	for profileID, stored := range s.chargingProfiles {
		if chargingProfileMatchesClear(profileID, stored, request) {
			delete(s.chargingProfiles, profileID)
			removed++
		}
	}
	s.mu.Unlock()
	status := smartcharging.ClearChargingProfileStatusUnknown
	if removed > 0 {
		status = smartcharging.ClearChargingProfileStatusAccepted
	}
	s.emit("inbound", smartcharging.ClearChargingProfileFeatureName, string(status), request.ConnectorId, "removed="+strconv.Itoa(removed))
	return smartcharging.NewClearChargingProfileConfirmation(status), nil
}

func chargingProfileMatchesClear(profileID int, stored storedChargingProfile, request *smartcharging.ClearChargingProfileRequest) bool {
	if request.Id != nil && profileID != *request.Id {
		return false
	}
	if request.ConnectorId != nil && stored.connectorID != *request.ConnectorId {
		return false
	}
	if request.StackLevel != nil && stored.profile.StackLevel != *request.StackLevel {
		return false
	}
	if request.ChargingProfilePurpose != "" && stored.profile.ChargingProfilePurpose != request.ChargingProfilePurpose {
		return false
	}
	return true
}

func (s *simulator) OnGetCompositeSchedule(request *smartcharging.GetCompositeScheduleRequest) (*smartcharging.GetCompositeScheduleConfirmation, error) {
	if request.ConnectorId > 0 {
		s.mu.Lock()
		_, ok := s.connectors[request.ConnectorId]
		s.mu.Unlock()
		if !ok {
			s.emit("inbound", smartcharging.GetCompositeScheduleFeatureName, string(smartcharging.GetCompositeScheduleStatusRejected), &request.ConnectorId, "unknown connector")
			return smartcharging.NewGetCompositeScheduleConfirmation(smartcharging.GetCompositeScheduleStatusRejected), nil
		}
	}

	s.mu.Lock()
	profile := s.activeChargingProfileLocked(request.ConnectorId)
	s.mu.Unlock()

	unit := request.ChargingRateUnit
	if unit == "" {
		unit = types.ChargingRateUnitAmperes
	}
	var schedule *types.ChargingSchedule
	if profile != nil && profile.ChargingSchedule != nil {
		if request.ChargingRateUnit != "" && profile.ChargingSchedule.ChargingRateUnit != request.ChargingRateUnit {
			s.emit("inbound", smartcharging.GetCompositeScheduleFeatureName, string(smartcharging.GetCompositeScheduleStatusRejected), &request.ConnectorId, "rate unit conversion not supported")
			return smartcharging.NewGetCompositeScheduleConfirmation(smartcharging.GetCompositeScheduleStatusRejected), nil
		}
		schedule = cloneChargingSchedule(profile.ChargingSchedule)
		unit = schedule.ChargingRateUnit
	} else {
		limit := 32.0
		if unit == types.ChargingRateUnitWatts {
			limit = 22000
		}
		schedule = types.NewChargingSchedule(unit, types.NewChargingSchedulePeriod(0, limit))
	}
	duration := request.Duration
	schedule.Duration = &duration
	schedule.StartSchedule = types.Now()
	confirmation := smartcharging.NewGetCompositeScheduleConfirmation(smartcharging.GetCompositeScheduleStatusAccepted)
	connectorID := request.ConnectorId
	confirmation.ConnectorId = &connectorID
	confirmation.ScheduleStart = types.Now()
	confirmation.ChargingSchedule = schedule
	detail := fmt.Sprintf("duration=%d unit=%s periods=%d", request.Duration, unit, len(schedule.ChargingSchedulePeriod))
	s.emit("inbound", smartcharging.GetCompositeScheduleFeatureName, string(smartcharging.GetCompositeScheduleStatusAccepted), &request.ConnectorId, detail)
	return confirmation, nil
}

func (s *simulator) activeChargingProfileLocked(connectorID int) *types.ChargingProfile {
	now := time.Now()
	var selected *types.ChargingProfile
	for _, stored := range s.chargingProfiles {
		profile := stored.profile
		if profile == nil || (stored.connectorID != 0 && stored.connectorID != connectorID) {
			continue
		}
		if profile.ValidFrom != nil && now.Before(profile.ValidFrom.Time) {
			continue
		}
		if profile.ValidTo != nil && !now.Before(profile.ValidTo.Time) {
			continue
		}
		if profile.ChargingProfilePurpose == types.ChargingProfilePurposeTxProfile && connectorID > 0 {
			connector := s.connectors[connectorID]
			if connector == nil || connector.transactionID == 0 || (profile.TransactionId != 0 && profile.TransactionId != connector.transactionID) {
				continue
			}
		}
		if selected == nil || profile.StackLevel > selected.StackLevel {
			selected = profile
		}
	}
	return selected
}

func cloneChargingSchedule(source *types.ChargingSchedule) *types.ChargingSchedule {
	if source == nil {
		return nil
	}
	copySchedule := *source
	copySchedule.ChargingSchedulePeriod = append([]types.ChargingSchedulePeriod(nil), source.ChargingSchedulePeriod...)
	if source.Duration != nil {
		value := *source.Duration
		copySchedule.Duration = &value
	}
	if source.MinChargingRate != nil {
		value := *source.MinChargingRate
		copySchedule.MinChargingRate = &value
	}
	return &copySchedule
}

var (
	_ firmware.ChargePointHandler      = (*simulator)(nil)
	_ localauth.ChargePointHandler     = (*simulator)(nil)
	_ remotetrigger.ChargePointHandler = (*simulator)(nil)
	_ reservation.ChargePointHandler   = (*simulator)(nil)
	_ smartcharging.ChargePointHandler = (*simulator)(nil)
)
