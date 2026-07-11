package ocppclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/lorenzodonini/ocpp-go/ocpp"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

func (c *client) Boot(ctx context.Context, request BootRequest) (BootResult, error) {
	confirmation, err := await(ctx, func() (*core.BootNotificationConfirmation, error) {
		return c.cp.BootNotification(request.Model, request.Vendor, func(payload *core.BootNotificationRequest) {
			payload.FirmwareVersion = request.FirmwareVersion
			payload.ChargePointSerialNumber = request.SerialNumber
			payload.MeterSerialNumber = request.MeterSerialNumber
			payload.MeterType = request.MeterType
		})
	})
	if err != nil { return BootResult{}, classify(err, ErrProtocol) }
	result := BootResult{Status: string(confirmation.Status), Interval: confirmation.Interval}
	if confirmation.CurrentTime != nil { result.CurrentTime = confirmation.CurrentTime.Time }
	if confirmation.Status == core.RegistrationStatusRejected { return result, fmt.Errorf("%w: BootNotification status is %s", ErrRejected, confirmation.Status) }
	return result, nil
}

func (c *client) Heartbeat(ctx context.Context) (HeartbeatResult, error) {
	confirmation, err := await(ctx, func() (*core.HeartbeatConfirmation, error) { return c.cp.Heartbeat() })
	if err != nil { return HeartbeatResult{}, classify(err, ErrProtocol) }
	result := HeartbeatResult{}
	if confirmation.CurrentTime != nil { result.CurrentTime = confirmation.CurrentTime.Time }
	return result, nil
}

func (c *client) Authorize(ctx context.Context, idTag string) (AuthorizationResult, error) {
	confirmation, err := await(ctx, func() (*core.AuthorizeConfirmation, error) { return c.cp.Authorize(idTag) })
	if err != nil { return AuthorizationResult{}, classify(err, ErrProtocol) }
	result := authorizationResult(confirmation.IdTagInfo)
	if result.Status != string(types.AuthorizationStatusAccepted) { return result, fmt.Errorf("%w: authorization status is %s", ErrRejected, result.Status) }
	return result, nil
}

func (c *client) StatusNotification(ctx context.Context, request StatusRequest) error {
	_, err := await(ctx, func() (*core.StatusNotificationConfirmation, error) {
		return c.cp.StatusNotification(request.ConnectorID, core.ChargePointErrorCode(request.ErrorCode), core.ChargePointStatus(request.Status), func(payload *core.StatusNotificationRequest) {
			payload.Info = request.Info
			payload.VendorId = request.VendorID
			payload.VendorErrorCode = request.VendorErrorCode
			payload.Timestamp = types.NewDateTime(request.Timestamp)
		})
	})
	return classify(err, ErrProtocol)
}

func (c *client) MeterValues(ctx context.Context, request MeterRequest) error {
	sampled := types.SampledValue{Value: request.Value, Measurand: types.Measurand(request.Measurand), Unit: types.UnitOfMeasure(request.Unit), Context: types.ReadingContext(request.Context), Location: types.Location(request.Location), Phase: types.Phase(request.Phase), Format: types.ValueFormatRaw}
	meterValue := types.MeterValue{Timestamp: types.NewDateTime(request.Timestamp), SampledValue: []types.SampledValue{sampled}}
	_, err := await(ctx, func() (*core.MeterValuesConfirmation, error) {
		return c.cp.MeterValues(request.ConnectorID, []types.MeterValue{meterValue}, func(payload *core.MeterValuesRequest) { payload.TransactionId = request.TransactionID })
	})
	return classify(err, ErrProtocol)
}

func (c *client) StartTransaction(ctx context.Context, request StartTransactionRequest) (StartTransactionResult, error) {
	confirmation, err := await(ctx, func() (*core.StartTransactionConfirmation, error) {
		return c.cp.StartTransaction(request.ConnectorID, request.IDTag, request.MeterStart, types.NewDateTime(request.Timestamp), func(payload *core.StartTransactionRequest) { payload.ReservationId = request.ReservationID })
	})
	if err != nil { return StartTransactionResult{}, classify(err, ErrProtocol) }
	result := StartTransactionResult{TransactionID: confirmation.TransactionId, Authorization: authorizationResult(confirmation.IdTagInfo)}
	if result.Authorization.Status != string(types.AuthorizationStatusAccepted) { return result, fmt.Errorf("%w: transaction authorization status is %s", ErrRejected, result.Authorization.Status) }
	return result, nil
}

func (c *client) StopTransaction(ctx context.Context, request StopTransactionRequest) (StopTransactionResult, error) {
	confirmation, err := await(ctx, func() (*core.StopTransactionConfirmation, error) {
		return c.cp.StopTransaction(request.MeterStop, types.NewDateTime(request.Timestamp), request.TransactionID, func(payload *core.StopTransactionRequest) { payload.IdTag = request.IDTag; payload.Reason = core.Reason(request.Reason) })
	})
	if err != nil { return StopTransactionResult{}, classify(err, ErrProtocol) }
	result := StopTransactionResult{}
	if confirmation.IdTagInfo != nil {
		auth := authorizationResult(confirmation.IdTagInfo)
		result.Authorization = &auth
		if auth.Status != string(types.AuthorizationStatusAccepted) { return result, fmt.Errorf("%w: stop transaction authorization status is %s", ErrRejected, auth.Status) }
	}
	return result, nil
}

func authorizationResult(info *types.IdTagInfo) AuthorizationResult {
	if info == nil { return AuthorizationResult{} }
	result := AuthorizationResult{Status: string(info.Status), ParentIDTag: info.ParentIdTag}
	if info.ExpiryDate != nil { expiry := info.ExpiryDate.Time; result.ExpiryDate = &expiry }
	return result
}

func await[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	type response struct { value T; err error }
	ch := make(chan response, 1)
	go func() { value, err := fn(); ch <- response{value: value, err: err} }()
	select {
	case <-ctx.Done(): var zero T; return zero, fmt.Errorf("%w: %v", ErrTimeout, ctx.Err())
	case result := <-ch: return result.value, result.err
	}
}

func classify(err error, fallback error) error {
	if err == nil { return nil }
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrTimeout) || strings.Contains(strings.ToLower(err.Error()), "timeout") { return fmt.Errorf("%w: %v", ErrTimeout, err) }
	var protocolError *ocpp.Error
	if errors.As(err, &protocolError) { return fmt.Errorf("%w: %v", ErrProtocol, err) }
	var netError net.Error
	if errors.As(err, &netError) { return fmt.Errorf("%w: %v", ErrConnection, err) }
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "401") || strings.Contains(message, "403") || strings.Contains(message, "unauthorized") || strings.Contains(message, "forbidden") || strings.Contains(message, "certificate") || strings.Contains(message, "x509") || strings.Contains(message, "tls") { return fmt.Errorf("%w: %v", ErrAuthSecurity, err) }
	return fmt.Errorf("%w: %v", fallback, err)
}
