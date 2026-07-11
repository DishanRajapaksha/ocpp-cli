package ocppclient

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

func (s *simulator) OnChangeAvailability(request *core.ChangeAvailabilityRequest) (*core.ChangeAvailabilityConfirmation, error) {
	s.mu.Lock()
	ids := make([]int, 0, len(s.connectors))
	if request.ConnectorId == 0 {
		for id := range s.connectors {
			ids = append(ids, id)
		}
	} else if _, ok := s.connectors[request.ConnectorId]; ok {
		ids = append(ids, request.ConnectorId)
	} else {
		s.mu.Unlock()
		s.emit("inbound", core.ChangeAvailabilityFeatureName, string(core.AvailabilityStatusRejected), &request.ConnectorId, string(request.Type))
		return core.NewChangeAvailabilityConfirmation(core.AvailabilityStatusRejected), nil
	}
	scheduled := false
	updates := make(map[int]core.ChargePointStatus, len(ids))
	for _, id := range ids {
		connector := s.connectors[id]
		connector.availability = request.Type
		if connector.transactionID > 0 {
			scheduled = true
			continue
		}
		if request.Type == core.AvailabilityTypeInoperative {
			connector.status = core.ChargePointStatusUnavailable
		} else {
			connector.status = core.ChargePointStatusAvailable
		}
		updates[id] = connector.status
	}
	s.mu.Unlock()
	status := core.AvailabilityStatusAccepted
	if scheduled {
		status = core.AvailabilityStatusScheduled
	}
	s.emit("inbound", core.ChangeAvailabilityFeatureName, string(status), &request.ConnectorId, string(request.Type))
	for id, connectorStatus := range updates {
		idCopy := id
		statusCopy := connectorStatus
		go func() { _ = s.notifyStatus(s.runContext(), idCopy, statusCopy) }()
	}
	return core.NewChangeAvailabilityConfirmation(status), nil
}

func (s *simulator) OnChangeConfiguration(request *core.ChangeConfigurationRequest) (*core.ChangeConfigurationConfirmation, error) {
	s.mu.Lock()
	key, ok := s.configuration[request.Key]
	if !ok {
		s.mu.Unlock()
		s.emit("inbound", core.ChangeConfigurationFeatureName, string(core.ConfigurationStatusNotSupported), nil, request.Key)
		return core.NewChangeConfigurationConfirmation(core.ConfigurationStatusNotSupported), nil
	}
	if key.Readonly {
		s.mu.Unlock()
		s.emit("inbound", core.ChangeConfigurationFeatureName, string(core.ConfigurationStatusRejected), nil, request.Key)
		return core.NewChangeConfigurationConfirmation(core.ConfigurationStatusRejected), nil
	}
	value := request.Value
	key.Value = &value
	s.configuration[request.Key] = key
	s.mu.Unlock()
	s.emit("inbound", core.ChangeConfigurationFeatureName, string(core.ConfigurationStatusAccepted), nil, request.Key+"="+request.Value)
	return core.NewChangeConfigurationConfirmation(core.ConfigurationStatusAccepted), nil
}

func (s *simulator) OnClearCache(_ *core.ClearCacheRequest) (*core.ClearCacheConfirmation, error) {
	s.emit("inbound", core.ClearCacheFeatureName, string(core.ClearCacheStatusAccepted), nil, "")
	return core.NewClearCacheConfirmation(core.ClearCacheStatusAccepted), nil
}

func (s *simulator) OnDataTransfer(request *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	s.emit("inbound", core.DataTransferFeatureName, string(core.DataTransferStatusAccepted), nil, request.VendorId+":"+request.MessageId)
	confirmation := core.NewDataTransferConfirmation(core.DataTransferStatusAccepted)
	confirmation.Data = request.Data
	return confirmation, nil
}

func (s *simulator) OnGetConfiguration(request *core.GetConfigurationRequest) (*core.GetConfigurationConfirmation, error) {
	s.mu.Lock()
	keys := append([]string(nil), request.Key...)
	if len(keys) == 0 {
		for key := range s.configuration {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	result := make([]core.ConfigurationKey, 0, len(keys))
	unknown := make([]string, 0)
	for _, key := range keys {
		if value, ok := s.configuration[key]; ok {
			result = append(result, value)
		} else {
			unknown = append(unknown, key)
		}
	}
	s.mu.Unlock()
	confirmation := core.NewGetConfigurationConfirmation(result)
	confirmation.UnknownKey = unknown
	s.emit("inbound", core.GetConfigurationFeatureName, "Accepted", nil, strings.Join(keys, ","))
	return confirmation, nil
}

func (s *simulator) OnRemoteStartTransaction(request *core.RemoteStartTransactionRequest) (*core.RemoteStartTransactionConfirmation, error) {
	s.mu.Lock()
	connectorID := 0
	if request.ConnectorId != nil {
		connectorID = *request.ConnectorId
	} else {
		for id := 1; id <= s.opts.Connectors; id++ {
			connector := s.connectors[id]
			if connector.availability == core.AvailabilityTypeOperative && connector.status == core.ChargePointStatusAvailable && connector.transactionID == 0 {
				connectorID = id
				break
			}
		}
	}
	connector, ok := s.connectors[connectorID]
	if !ok || connector.availability != core.AvailabilityTypeOperative || connector.status != core.ChargePointStatusAvailable || connector.transactionID != 0 {
		s.mu.Unlock()
		s.emit("inbound", core.RemoteStartTransactionFeatureName, string(types.RemoteStartStopStatusRejected), request.ConnectorId, request.IdTag)
		return core.NewRemoteStartTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
	}
	connector.status = core.ChargePointStatusPreparing
	connector.idTag = request.IdTag
	s.mu.Unlock()
	s.emit("inbound", core.RemoteStartTransactionFeatureName, string(types.RemoteStartStopStatusAccepted), &connectorID, request.IdTag)
	go s.beginRemoteTransaction(connectorID, request.IdTag)
	return core.NewRemoteStartTransactionConfirmation(types.RemoteStartStopStatusAccepted), nil
}

func (s *simulator) beginRemoteTransaction(connectorID int, idTag string) {
	ctx := s.runContext()
	_ = s.notifyStatus(ctx, connectorID, core.ChargePointStatusPreparing)
	s.mu.Lock()
	meterStart := s.connectors[connectorID].meter
	s.mu.Unlock()
	confirmation, err := await(ctx, func() (*core.StartTransactionConfirmation, error) {
		return s.cp.StartTransaction(connectorID, idTag, meterStart, types.Now())
	})
	if err != nil {
		s.mu.Lock()
		s.connectors[connectorID].status = core.ChargePointStatusAvailable
		s.connectors[connectorID].idTag = ""
		s.mu.Unlock()
		s.emit("outbound", core.StartTransactionFeatureName, "Error", &connectorID, err.Error())
		_ = s.notifyStatus(ctx, connectorID, core.ChargePointStatusAvailable)
		return
	}
	status := ""
	if confirmation.IdTagInfo != nil {
		status = string(confirmation.IdTagInfo.Status)
	}
	s.mu.Lock()
	connector := s.connectors[connectorID]
	if status == string(types.AuthorizationStatusAccepted) {
		connector.transactionID = confirmation.TransactionId
		connector.status = core.ChargePointStatusCharging
	} else {
		connector.status = core.ChargePointStatusAvailable
		connector.idTag = ""
	}
	finalStatus := connector.status
	s.mu.Unlock()
	s.emit("outbound", core.StartTransactionFeatureName, status, &connectorID, strconv.Itoa(confirmation.TransactionId))
	_ = s.notifyStatus(ctx, connectorID, finalStatus)
}

func (s *simulator) OnRemoteStopTransaction(request *core.RemoteStopTransactionRequest) (*core.RemoteStopTransactionConfirmation, error) {
	s.mu.Lock()
	connectorID := 0
	for id, connector := range s.connectors {
		if connector.transactionID == request.TransactionId {
			connectorID = id
			connector.status = core.ChargePointStatusFinishing
			break
		}
	}
	s.mu.Unlock()
	if connectorID == 0 {
		s.emit("inbound", core.RemoteStopTransactionFeatureName, string(types.RemoteStartStopStatusRejected), nil, strconv.Itoa(request.TransactionId))
		return core.NewRemoteStopTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
	}
	s.emit("inbound", core.RemoteStopTransactionFeatureName, string(types.RemoteStartStopStatusAccepted), &connectorID, strconv.Itoa(request.TransactionId))
	go s.finishRemoteTransaction(connectorID, request.TransactionId)
	return core.NewRemoteStopTransactionConfirmation(types.RemoteStartStopStatusAccepted), nil
}

func (s *simulator) finishRemoteTransaction(connectorID, transactionID int) {
	ctx := s.runContext()
	_ = s.notifyStatus(ctx, connectorID, core.ChargePointStatusFinishing)
	s.mu.Lock()
	connector := s.connectors[connectorID]
	meterStop := connector.meter
	idTag := connector.idTag
	s.mu.Unlock()
	confirmation, err := await(ctx, func() (*core.StopTransactionConfirmation, error) {
		return s.cp.StopTransaction(meterStop, types.Now(), transactionID, func(request *core.StopTransactionRequest) {
			request.IdTag = idTag
			request.Reason = core.ReasonRemote
		})
	})
	if err != nil {
		s.emit("outbound", core.StopTransactionFeatureName, "Error", &connectorID, err.Error())
		return
	}
	status := "Accepted"
	if confirmation.IdTagInfo != nil {
		status = string(confirmation.IdTagInfo.Status)
	}
	s.mu.Lock()
	connector.transactionID = 0
	connector.idTag = ""
	if connector.availability == core.AvailabilityTypeInoperative {
		connector.status = core.ChargePointStatusUnavailable
	} else {
		connector.status = core.ChargePointStatusAvailable
	}
	finalStatus := connector.status
	s.mu.Unlock()
	s.emit("outbound", core.StopTransactionFeatureName, status, &connectorID, strconv.Itoa(transactionID))
	_ = s.notifyStatus(ctx, connectorID, finalStatus)
}

func (s *simulator) OnReset(request *core.ResetRequest) (*core.ResetConfirmation, error) {
	s.mu.Lock()
	updates := make(map[int]core.ChargePointStatus, len(s.connectors))
	for id, connector := range s.connectors {
		connector.transactionID = 0
		connector.idTag = ""
		if connector.availability == core.AvailabilityTypeInoperative {
			connector.status = core.ChargePointStatusUnavailable
		} else {
			connector.status = core.ChargePointStatusAvailable
		}
		updates[id] = connector.status
	}
	s.mu.Unlock()
	s.emit("inbound", core.ResetFeatureName, string(core.ResetStatusAccepted), nil, string(request.Type))
	for id, status := range updates {
		idCopy := id
		statusCopy := status
		go func() { _ = s.notifyStatus(s.runContext(), idCopy, statusCopy) }()
	}
	return core.NewResetConfirmation(core.ResetStatusAccepted), nil
}

func (s *simulator) OnUnlockConnector(request *core.UnlockConnectorRequest) (*core.UnlockConnectorConfirmation, error) {
	s.mu.Lock()
	_, ok := s.connectors[request.ConnectorId]
	s.mu.Unlock()
	if !ok {
		s.emit("inbound", core.UnlockConnectorFeatureName, string(core.UnlockStatusNotSupported), &request.ConnectorId, "")
		return core.NewUnlockConnectorConfirmation(core.UnlockStatusNotSupported), nil
	}
	s.emit("inbound", core.UnlockConnectorFeatureName, string(core.UnlockStatusUnlocked), &request.ConnectorId, "")
	return core.NewUnlockConnectorConfirmation(core.UnlockStatusUnlocked), nil
}

func (s *simulator) runContext() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}

var _ core.ChargePointHandler = (*simulator)(nil)
