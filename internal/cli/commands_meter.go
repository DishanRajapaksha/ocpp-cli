package cli

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/DishanRajapaksha/industrial-cli-kit/safety"
	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
	"github.com/DishanRajapaksha/ocpp-cli/internal/ocppclient"
)

func (a *App) meterValues(args []string) error {
	fs := a.flagSet("meter-values")
	var common commonFlags
	addCommonFlags(fs, &common)
	connector := fs.Int("connector", 1, "connector ID")
	transactionID := fs.Int("transaction-id", -1, "associated transaction ID")
	value := fs.String("value", "", "sample value")
	measurand := fs.String("measurand", "Energy.Active.Import.Register", "OCPP measurand")
	unit := fs.String("unit", "Wh", "OCPP unit")
	readingContext := fs.String("context", "Sample.Periodic", "OCPP reading context")
	location := fs.String("location", "Outlet", "measurement location")
	phase := fs.String("phase", "", "optional electrical phase")
	timestamp := fs.String("timestamp", "now", "RFC3339 timestamp or now")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: meter-values does not accept positional arguments", config.ErrConfig)
	}
	if *connector < 0 {
		return fmt.Errorf("%w: connector must be zero or greater", config.ErrConfig)
	}
	if *value == "" {
		return fmt.Errorf("%w: value is required", config.ErrConfig)
	}
	if !validMeasurand(*measurand) || !validUnit(*unit) || !validReadingContext(*readingContext) || !validLocation(*location) || !validPhase(*phase) {
		return fmt.Errorf("%w: invalid meter-value metadata", config.ErrConfig)
	}
	ts, err := parseTimestamp(*timestamp)
	if err != nil {
		return err
	}
	cfg, format, err := resolveCommon(fs, common)
	if err != nil {
		return err
	}
	var txID *int
	if *transactionID >= 0 {
		txID = transactionID
	}
	station, ctx, cancel, err := a.connect(cfg)
	if err != nil {
		return err
	}
	defer cancel()
	defer station.Close()
	err = station.MeterValues(ctx, ocppclient.MeterRequest{ConnectorID: *connector, TransactionID: txID, Value: *value, Measurand: *measurand, Unit: *unit, Context: *readingContext, Location: *location, Phase: *phase, Timestamp: ts})
	if err != nil {
		return err
	}
	result := struct {
		Sent        bool   `json:"sent"`
		Action      string `json:"action"`
		ConnectorID int    `json:"connector_id"`
		Value       string `json:"value"`
		Measurand   string `json:"measurand"`
		Unit        string `json:"unit"`
	}{true, "MeterValues", *connector, *value, *measurand, *unit}
	return renderSnapshot(a.out, format, keyValueSnapshot(result,
		[2]string{"SENT", "true"}, [2]string{"ACTION", "MeterValues"}, [2]string{"CONNECTOR_ID", strconv.Itoa(*connector)}, [2]string{"VALUE", *value}, [2]string{"MEASURAND", *measurand}, [2]string{"UNIT", *unit},
	))
}

func (a *App) startTransaction(args []string) error {
	fs := a.flagSet("start-transaction")
	var common commonFlags
	addCommonFlags(fs, &common)
	connector := fs.Int("connector", 1, "connector ID")
	idTag := fs.String("id-tag", "", "authorized identifier")
	meterStart := fs.Int("meter-start", -1, "starting meter value in Wh")
	reservationID := fs.Int("reservation-id", -1, "optional reservation ID")
	timestamp := fs.String("timestamp", "now", "RFC3339 timestamp or now")
	yes := fs.Bool("yes", false, "send the transaction request")
	dryRun := fs.Bool("dry-run", false, "validate and show the request without sending it")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: start-transaction does not accept positional arguments", config.ErrConfig)
	}
	if *connector <= 0 {
		return fmt.Errorf("%w: connector must be greater than zero", config.ErrConfig)
	}
	if *idTag == "" || len(*idTag) > 20 {
		return fmt.Errorf("%w: id-tag is required and must not exceed 20 characters", config.ErrConfig)
	}
	if *meterStart < 0 {
		return fmt.Errorf("%w: meter-start is required and must be zero or greater", config.ErrConfig)
	}
	ts, err := parseTimestamp(*timestamp)
	if err != nil {
		return err
	}
	cfg, format, err := resolveCommon(fs, common)
	if err != nil {
		return err
	}
	mode, err := safety.Resolve(*yes, *dryRun)
	if err != nil {
		return fmt.Errorf("%w: %v", config.ErrConfig, err)
	}
	var reservation *int
	if *reservationID >= 0 {
		reservation = reservationID
	}
	request := ocppclient.StartTransactionRequest{ConnectorID: *connector, IDTag: *idTag, MeterStart: *meterStart, ReservationID: reservation, Timestamp: ts}
	if mode == safety.DryRun {
		result := struct {
			Mode          string `json:"mode"`
			Action        string `json:"action"`
			ConnectorID   int    `json:"connector_id"`
			IDTag         string `json:"id_tag"`
			MeterStart    int    `json:"meter_start"`
			ReservationID *int   `json:"reservation_id,omitempty"`
		}{mode.String(), "StartTransaction", *connector, *idTag, *meterStart, reservation}
		return renderSnapshot(a.out, format, keyValueSnapshot(result,
			[2]string{"MODE", mode.String()}, [2]string{"ACTION", "StartTransaction"}, [2]string{"CONNECTOR_ID", strconv.Itoa(*connector)}, [2]string{"ID_TAG", *idTag}, [2]string{"METER_START", strconv.Itoa(*meterStart)},
		))
	}
	station, ctx, cancel, err := a.connect(cfg)
	if err != nil {
		return err
	}
	defer cancel()
	defer station.Close()
	result, operationErr := station.StartTransaction(ctx, request)
	if operationErr != nil && !errors.Is(operationErr, ocppclient.ErrRejected) {
		return operationErr
	}
	expiry := ""
	if result.Authorization.ExpiryDate != nil {
		expiry = formatTime(*result.Authorization.ExpiryDate)
	}
	renderErr := renderSnapshot(a.out, format, keyValueSnapshot(result,
		[2]string{"TRANSACTION_ID", strconv.Itoa(result.TransactionID)}, [2]string{"STATUS", result.Authorization.Status}, [2]string{"EXPIRY_DATE", expiry},
	))
	if renderErr != nil {
		return renderErr
	}
	return operationErr
}

func (a *App) stopTransaction(args []string) error {
	fs := a.flagSet("stop-transaction")
	var common commonFlags
	addCommonFlags(fs, &common)
	transactionID := fs.Int("transaction-id", -1, "transaction ID")
	meterStop := fs.Int("meter-stop", -1, "final meter value in Wh")
	idTag := fs.String("id-tag", "", "optional identifier that stopped the transaction")
	reason := fs.String("reason", "Local", "OCPP stop reason")
	timestamp := fs.String("timestamp", "now", "RFC3339 timestamp or now")
	yes := fs.Bool("yes", false, "send the transaction request")
	dryRun := fs.Bool("dry-run", false, "validate and show the request without sending it")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: stop-transaction does not accept positional arguments", config.ErrConfig)
	}
	if *transactionID < 0 {
		return fmt.Errorf("%w: transaction-id is required and must be zero or greater", config.ErrConfig)
	}
	if *meterStop < 0 {
		return fmt.Errorf("%w: meter-stop is required and must be zero or greater", config.ErrConfig)
	}
	if len(*idTag) > 20 {
		return fmt.Errorf("%w: id-tag must not exceed 20 characters", config.ErrConfig)
	}
	if !validStopReason(*reason) {
		return fmt.Errorf("%w: invalid reason %q", config.ErrConfig, *reason)
	}
	ts, err := parseTimestamp(*timestamp)
	if err != nil {
		return err
	}
	cfg, format, err := resolveCommon(fs, common)
	if err != nil {
		return err
	}
	mode, err := safety.Resolve(*yes, *dryRun)
	if err != nil {
		return fmt.Errorf("%w: %v", config.ErrConfig, err)
	}
	request := ocppclient.StopTransactionRequest{TransactionID: *transactionID, MeterStop: *meterStop, IDTag: *idTag, Reason: *reason, Timestamp: ts}
	if mode == safety.DryRun {
		result := struct {
			Mode          string `json:"mode"`
			Action        string `json:"action"`
			TransactionID int    `json:"transaction_id"`
			MeterStop     int    `json:"meter_stop"`
			Reason        string `json:"reason"`
		}{mode.String(), "StopTransaction", *transactionID, *meterStop, *reason}
		return renderSnapshot(a.out, format, keyValueSnapshot(result,
			[2]string{"MODE", mode.String()}, [2]string{"ACTION", "StopTransaction"}, [2]string{"TRANSACTION_ID", strconv.Itoa(*transactionID)}, [2]string{"METER_STOP", strconv.Itoa(*meterStop)}, [2]string{"REASON", *reason},
		))
	}
	station, ctx, cancel, err := a.connect(cfg)
	if err != nil {
		return err
	}
	defer cancel()
	defer station.Close()
	result, operationErr := station.StopTransaction(ctx, request)
	if operationErr != nil && !errors.Is(operationErr, ocppclient.ErrRejected) {
		return operationErr
	}
	status := ""
	if result.Authorization != nil {
		status = result.Authorization.Status
	}
	renderErr := renderSnapshot(a.out, format, keyValueSnapshot(result,
		[2]string{"STOPPED", "true"}, [2]string{"TRANSACTION_ID", strconv.Itoa(*transactionID)}, [2]string{"AUTHORIZATION_STATUS", status},
	))
	if renderErr != nil {
		return renderErr
	}
	return operationErr
}
