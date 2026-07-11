package ocppclient

import (
	"testing"
	"time"

	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/localauth"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/reservation"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

func newTestSimulator(t *testing.T) *simulator {
	t.Helper()
	created, err := NewSimulator(config.DefaultClientConfig(), SimulatorOptions{Connectors: 2, MeterStart: 1000, MeterStep: 10})
	if err != nil {
		t.Fatal(err)
	}
	result, ok := created.(*simulator)
	if !ok {
		t.Fatalf("unexpected simulator type %T", created)
	}
	return result
}

func TestReservationLifecycle(t *testing.T) {
	s := newTestSimulator(t)
	expiry := types.NewDateTime(time.Now().Add(time.Hour))
	confirmation, err := s.OnReserveNow(reservation.NewReserveNowRequest(1, expiry, "TAG-1", 7))
	if err != nil {
		t.Fatal(err)
	}
	if confirmation.Status != reservation.ReservationStatusAccepted {
		t.Fatalf("reserve status = %s", confirmation.Status)
	}
	if s.connectors[1].status != core.ChargePointStatusReserved {
		t.Fatalf("connector status = %s", s.connectors[1].status)
	}

	s.mu.Lock()
	_, _, allowed := s.selectRemoteStartConnectorLocked(intPointer(1), "OTHER")
	connectorID, reservationID, matchingAllowed := s.selectRemoteStartConnectorLocked(intPointer(1), "TAG-1")
	s.mu.Unlock()
	if allowed {
		t.Fatal("reservation allowed a different idTag")
	}
	if !matchingAllowed || connectorID != 1 || reservationID != 7 {
		t.Fatalf("matching reservation selection = connector %d reservation %d allowed %v", connectorID, reservationID, matchingAllowed)
	}

	cancel, err := s.OnCancelReservation(reservation.NewCancelReservationRequest(7))
	if err != nil {
		t.Fatal(err)
	}
	if cancel.Status != reservation.CancelReservationStatusAccepted {
		t.Fatalf("cancel status = %s", cancel.Status)
	}
	if s.connectors[1].status != core.ChargePointStatusAvailable {
		t.Fatalf("connector status after cancel = %s", s.connectors[1].status)
	}
}

func TestExpiredReservationIsRemoved(t *testing.T) {
	s := newTestSimulator(t)
	expiry := types.NewDateTime(time.Now().Add(-time.Second))
	confirmation, err := s.OnReserveNow(reservation.NewReserveNowRequest(1, expiry, "TAG-1", 8))
	if err != nil {
		t.Fatal(err)
	}
	if confirmation.Status != reservation.ReservationStatusRejected {
		t.Fatalf("status = %s", confirmation.Status)
	}
}

func TestLocalAuthorizationListFullAndDifferentialUpdates(t *testing.T) {
	s := newTestSimulator(t)
	full := localauth.NewSendLocalListRequest(1, localauth.UpdateTypeFull)
	full.LocalAuthorizationList = []localauth.AuthorizationData{
		{IdTag: "A", IdTagInfo: types.NewIdTagInfo(types.AuthorizationStatusAccepted)},
		{IdTag: "B", IdTagInfo: types.NewIdTagInfo(types.AuthorizationStatusBlocked)},
	}
	confirmation, err := s.OnSendLocalList(full)
	if err != nil {
		t.Fatal(err)
	}
	if confirmation.Status != localauth.UpdateStatusAccepted || s.localAuthListVersion != 1 || len(s.localAuthList) != 2 {
		t.Fatalf("full update status=%s version=%d entries=%d", confirmation.Status, s.localAuthListVersion, len(s.localAuthList))
	}

	differential := localauth.NewSendLocalListRequest(2, localauth.UpdateTypeDifferential)
	differential.LocalAuthorizationList = []localauth.AuthorizationData{
		{IdTag: "A", IdTagInfo: nil},
		{IdTag: "C", IdTagInfo: types.NewIdTagInfo(types.AuthorizationStatusAccepted)},
	}
	confirmation, err = s.OnSendLocalList(differential)
	if err != nil {
		t.Fatal(err)
	}
	if confirmation.Status != localauth.UpdateStatusAccepted || s.localAuthListVersion != 2 {
		t.Fatalf("differential status=%s version=%d", confirmation.Status, s.localAuthListVersion)
	}
	if _, exists := s.localAuthList["A"]; exists {
		t.Fatal("deleted idTag remained in local list")
	}
	if _, exists := s.localAuthList["C"]; !exists {
		t.Fatal("new idTag was not added")
	}

	version, err := s.OnGetLocalListVersion(localauth.NewGetLocalListVersionRequest())
	if err != nil {
		t.Fatal(err)
	}
	if version.ListVersion != 2 {
		t.Fatalf("list version = %d", version.ListVersion)
	}
}

func TestLocalAuthorizationVersionMismatch(t *testing.T) {
	s := newTestSimulator(t)
	first := localauth.NewSendLocalListRequest(3, localauth.UpdateTypeFull)
	if _, err := s.OnSendLocalList(first); err != nil {
		t.Fatal(err)
	}
	stale := localauth.NewSendLocalListRequest(3, localauth.UpdateTypeDifferential)
	confirmation, err := s.OnSendLocalList(stale)
	if err != nil {
		t.Fatal(err)
	}
	if confirmation.Status != localauth.UpdateStatusVersionMismatch {
		t.Fatalf("status = %s", confirmation.Status)
	}
}

func TestSmartChargingProfileLifecycle(t *testing.T) {
	s := newTestSimulator(t)
	schedule := types.NewChargingSchedule(types.ChargingRateUnitAmperes, types.NewChargingSchedulePeriod(0, 16))
	profile := types.NewChargingProfile(42, 2, types.ChargingProfilePurposeTxDefaultProfile, types.ChargingProfileKindAbsolute, schedule)
	set, err := s.OnSetChargingProfile(smartcharging.NewSetChargingProfileRequest(1, profile))
	if err != nil {
		t.Fatal(err)
	}
	if set.Status != smartcharging.ChargingProfileStatusAccepted {
		t.Fatalf("set status = %s", set.Status)
	}

	compositeRequest := smartcharging.NewGetCompositeScheduleRequest(1, 900)
	compositeRequest.ChargingRateUnit = types.ChargingRateUnitAmperes
	composite, err := s.OnGetCompositeSchedule(compositeRequest)
	if err != nil {
		t.Fatal(err)
	}
	if composite.Status != smartcharging.GetCompositeScheduleStatusAccepted || composite.ChargingSchedule == nil {
		t.Fatalf("composite status=%s schedule=%v", composite.Status, composite.ChargingSchedule)
	}
	if got := composite.ChargingSchedule.ChargingSchedulePeriod[0].Limit; got != 16 {
		t.Fatalf("composite limit = %v", got)
	}

	clearRequest := smartcharging.NewClearChargingProfileRequest()
	clearRequest.Id = intPointer(42)
	cleared, err := s.OnClearChargingProfile(clearRequest)
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Status != smartcharging.ClearChargingProfileStatusAccepted {
		t.Fatalf("clear status = %s", cleared.Status)
	}
	cleared, err = s.OnClearChargingProfile(clearRequest)
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Status != smartcharging.ClearChargingProfileStatusUnknown {
		t.Fatalf("second clear status = %s", cleared.Status)
	}
}

func TestCompositeScheduleProvidesLocalDefault(t *testing.T) {
	s := newTestSimulator(t)
	request := smartcharging.NewGetCompositeScheduleRequest(1, 300)
	request.ChargingRateUnit = types.ChargingRateUnitWatts
	confirmation, err := s.OnGetCompositeSchedule(request)
	if err != nil {
		t.Fatal(err)
	}
	if confirmation.Status != smartcharging.GetCompositeScheduleStatusAccepted {
		t.Fatalf("status = %s", confirmation.Status)
	}
	if confirmation.ChargingSchedule == nil || confirmation.ChargingSchedule.ChargingRateUnit != types.ChargingRateUnitWatts {
		t.Fatalf("schedule = %#v", confirmation.ChargingSchedule)
	}
	if got := confirmation.ChargingSchedule.ChargingSchedulePeriod[0].Limit; got != 22000 {
		t.Fatalf("default power limit = %v", got)
	}
}

func TestTriggerRejectsUnknownConnectorForStatus(t *testing.T) {
	s := newTestSimulator(t)
	request := remotetrigger.NewTriggerMessageRequest(core.StatusNotificationFeatureName)
	request.ConnectorId = intPointer(99)
	confirmation, err := s.OnTriggerMessage(request)
	if err != nil {
		t.Fatal(err)
	}
	if confirmation.Status != remotetrigger.TriggerMessageStatusRejected {
		t.Fatalf("status = %s", confirmation.Status)
	}
}

func TestFirmwareAndDiagnosticsRequestsAreAcknowledged(t *testing.T) {
	s := newTestSimulator(t)
	diagnostics := firmware.NewGetDiagnosticsRequest("https://example.test/upload")
	diagnosticsConfirmation, err := s.OnGetDiagnostics(diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	if diagnosticsConfirmation.FileName == "" {
		t.Fatal("diagnostics filename is empty")
	}

	firmwareRequest := firmware.NewUpdateFirmwareRequest("https://example.test/firmware.bin", types.Now())
	if _, err := s.OnUpdateFirmware(firmwareRequest); err != nil {
		t.Fatal(err)
	}
}

func TestSupportedProfilesConfiguration(t *testing.T) {
	s := newTestSimulator(t)
	key := s.configuration["SupportedFeatureProfiles"]
	if key.Value == nil {
		t.Fatal("SupportedFeatureProfiles has no value")
	}
	for _, expected := range []string{"Core", "FirmwareManagement", "LocalAuthListManagement", "Reservation", "RemoteTrigger", "SmartCharging"} {
		if !containsCSV(*key.Value, expected) {
			t.Fatalf("supported profiles %q missing %q", *key.Value, expected)
		}
	}
}

func intPointer(value int) *int { return &value }

func containsCSV(value, expected string) bool {
	for _, item := range splitCSV(value) {
		if item == expected {
			return true
		}
	}
	return false
}

func splitCSV(value string) []string {
	var result []string
	start := 0
	for index := 0; index <= len(value); index++ {
		if index == len(value) || value[index] == ',' {
			result = append(result, value[start:index])
			start = index + 1
		}
	}
	return result
}
