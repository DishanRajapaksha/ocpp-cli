package cli

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
	"github.com/DishanRajapaksha/ocpp-cli/internal/ocppclient"
)

func (a *App) bootNotification(args []string) error {
	fs := a.flagSet("boot-notification")
	var common commonFlags
	addCommonFlags(fs, &common)
	model := fs.String("model", "", "charge point model")
	vendor := fs.String("vendor", "", "charge point vendor")
	firmware := fs.String("firmware-version", "", "firmware version")
	serial := fs.String("serial-number", "", "charge point serial number")
	meterSerial := fs.String("meter-serial-number", "", "meter serial number")
	meterType := fs.String("meter-type", "", "meter type")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: boot-notification does not accept positional arguments", config.ErrConfig)
	}
	cfg, format, err := resolveCommon(fs, common)
	if err != nil {
		return err
	}
	if !wasSet(fs, "model") {
		*model = cfg.ChargePointModel
	}
	if !wasSet(fs, "vendor") {
		*vendor = cfg.ChargePointVendor
	}
	if !wasSet(fs, "firmware-version") {
		*firmware = cfg.FirmwareVersion
	}
	if !wasSet(fs, "serial-number") {
		*serial = cfg.SerialNumber
	}
	if !wasSet(fs, "meter-serial-number") {
		*meterSerial = cfg.MeterSerialNumber
	}
	if !wasSet(fs, "meter-type") {
		*meterType = cfg.MeterType
	}
	if *model == "" || len(*model) > 20 {
		return fmt.Errorf("%w: model is required and must not exceed 20 characters", config.ErrConfig)
	}
	if *vendor == "" || len(*vendor) > 20 {
		return fmt.Errorf("%w: vendor is required and must not exceed 20 characters", config.ErrConfig)
	}
	if len(*firmware) > 50 || len(*serial) > 25 || len(*meterSerial) > 25 || len(*meterType) > 25 {
		return fmt.Errorf("%w: boot metadata exceeds OCPP 1.6 field limits", config.ErrConfig)
	}
	station, ctx, cancel, err := a.connect(cfg)
	if err != nil {
		return err
	}
	defer cancel()
	defer station.Close()
	result, operationErr := station.Boot(ctx, ocppclient.BootRequest{Model: *model, Vendor: *vendor, FirmwareVersion: *firmware, SerialNumber: *serial, MeterSerialNumber: *meterSerial, MeterType: *meterType})
	if operationErr != nil && !errors.Is(operationErr, ocppclient.ErrRejected) {
		return operationErr
	}
	renderErr := renderSnapshot(a.out, format, keyValueSnapshot(result,
		[2]string{"STATUS", result.Status}, [2]string{"INTERVAL_SECONDS", strconv.Itoa(result.Interval)}, [2]string{"CURRENT_TIME", formatTime(result.CurrentTime)},
	))
	if renderErr != nil {
		return renderErr
	}
	return operationErr
}

func (a *App) heartbeat(args []string) error {
	fs := a.flagSet("heartbeat")
	var common commonFlags
	addCommonFlags(fs, &common)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: heartbeat does not accept positional arguments", config.ErrConfig)
	}
	cfg, format, err := resolveCommon(fs, common)
	if err != nil {
		return err
	}
	station, ctx, cancel, err := a.connect(cfg)
	if err != nil {
		return err
	}
	defer cancel()
	defer station.Close()
	result, err := station.Heartbeat(ctx)
	if err != nil {
		return err
	}
	return renderSnapshot(a.out, format, keyValueSnapshot(result, [2]string{"CURRENT_TIME", formatTime(result.CurrentTime)}))
}

func (a *App) authorize(args []string) error {
	fs := a.flagSet("authorize")
	var common commonFlags
	addCommonFlags(fs, &common)
	idTag := fs.String("id-tag", "", "identifier to authorize")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: authorize does not accept positional arguments", config.ErrConfig)
	}
	if *idTag == "" || len(*idTag) > 20 {
		return fmt.Errorf("%w: id-tag is required and must not exceed 20 characters", config.ErrConfig)
	}
	cfg, format, err := resolveCommon(fs, common)
	if err != nil {
		return err
	}
	station, ctx, cancel, err := a.connect(cfg)
	if err != nil {
		return err
	}
	defer cancel()
	defer station.Close()
	result, operationErr := station.Authorize(ctx, *idTag)
	if operationErr != nil && !errors.Is(operationErr, ocppclient.ErrRejected) {
		return operationErr
	}
	expiry := ""
	if result.ExpiryDate != nil {
		expiry = formatTime(*result.ExpiryDate)
	}
	renderErr := renderSnapshot(a.out, format, keyValueSnapshot(result,
		[2]string{"STATUS", result.Status}, [2]string{"EXPIRY_DATE", expiry}, [2]string{"PARENT_ID_TAG", result.ParentIDTag},
	))
	if renderErr != nil {
		return renderErr
	}
	return operationErr
}

func (a *App) statusNotification(args []string) error {
	fs := a.flagSet("status-notification")
	var common commonFlags
	addCommonFlags(fs, &common)
	connector := fs.Int("connector", 0, "connector ID; 0 represents the whole charge point")
	status := fs.String("status", "Available", "charge point status")
	errorCode := fs.String("error-code", "NoError", "charge point error code")
	info := fs.String("info", "", "additional status information")
	vendorID := fs.String("vendor-id", "", "vendor identifier")
	vendorError := fs.String("vendor-error-code", "", "vendor-specific error code")
	timestamp := fs.String("timestamp", "now", "RFC3339 timestamp or now")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: status-notification does not accept positional arguments", config.ErrConfig)
	}
	if *connector < 0 {
		return fmt.Errorf("%w: connector must be zero or greater", config.ErrConfig)
	}
	if !validStatus(*status) {
		return fmt.Errorf("%w: invalid status %q", config.ErrConfig, *status)
	}
	if !validErrorCode(*errorCode) {
		return fmt.Errorf("%w: invalid error-code %q", config.ErrConfig, *errorCode)
	}
	if len(*info) > 50 || len(*vendorID) > 255 || len(*vendorError) > 50 {
		return fmt.Errorf("%w: status metadata exceeds OCPP 1.6 field limits", config.ErrConfig)
	}
	ts, err := parseTimestamp(*timestamp)
	if err != nil {
		return err
	}
	cfg, format, err := resolveCommon(fs, common)
	if err != nil {
		return err
	}
	station, ctx, cancel, err := a.connect(cfg)
	if err != nil {
		return err
	}
	defer cancel()
	defer station.Close()
	err = station.StatusNotification(ctx, ocppclient.StatusRequest{ConnectorID: *connector, Status: *status, ErrorCode: *errorCode, Info: *info, VendorID: *vendorID, VendorErrorCode: *vendorError, Timestamp: ts})
	if err != nil {
		return err
	}
	result := struct {
		Sent        bool   `json:"sent"`
		Action      string `json:"action"`
		ConnectorID int    `json:"connector_id"`
		Status      string `json:"status"`
	}{true, "StatusNotification", *connector, *status}
	return renderSnapshot(a.out, format, keyValueSnapshot(result,
		[2]string{"SENT", "true"}, [2]string{"ACTION", "StatusNotification"}, [2]string{"CONNECTOR_ID", strconv.Itoa(*connector)}, [2]string{"STATUS", *status},
	))
}
