package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
	"github.com/DishanRajapaksha/ocpp-cli/internal/ocppclient"
)

func (a *App) dataTransfer(args []string) error {
	fs := a.flagSet("data-transfer")
	var common commonFlags
	addCommonFlags(fs, &common)
	vendorID := fs.String("vendor-id", "", "vendor identifier")
	messageID := fs.String("message-id", "", "optional vendor message identifier")
	data := fs.String("data", "", "optional JSON payload")
	dataFile := fs.String("data-file", "", "file containing an optional JSON payload")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: data-transfer does not accept positional arguments", config.ErrConfig)
	}
	if *vendorID == "" || len(*vendorID) > 255 {
		return fmt.Errorf("%w: vendor-id is required and must not exceed 255 characters", config.ErrConfig)
	}
	if len(*messageID) > 50 {
		return fmt.Errorf("%w: message-id must not exceed 50 characters", config.ErrConfig)
	}
	payload, err := parseJSONPayload(*data, *dataFile)
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

	result, operationErr := station.DataTransfer(ctx, ocppclient.DataTransferRequest{
		VendorID:  *vendorID,
		MessageID: *messageID,
		Data:      payload,
	})
	if operationErr != nil && !errors.Is(operationErr, ocppclient.ErrRejected) {
		return operationErr
	}
	payloadText, err := compactJSON(result.Data)
	if err != nil {
		return err
	}
	renderErr := renderSnapshot(a.out, format, keyValueSnapshot(result,
		[2]string{"STATUS", result.Status},
		[2]string{"DATA", payloadText},
	))
	if renderErr != nil {
		return renderErr
	}
	return operationErr
}

func (a *App) diagnosticsStatus(args []string) error {
	fs := a.flagSet("diagnostics-status")
	var common commonFlags
	addCommonFlags(fs, &common)
	status := fs.String("status", "", "Idle, Uploaded, UploadFailed, or Uploading")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: diagnostics-status does not accept positional arguments", config.ErrConfig)
	}
	if !validDiagnosticsStatus(*status) {
		return fmt.Errorf("%w: invalid diagnostics status %q", config.ErrConfig, *status)
	}
	return a.sendStatusOnly(fs, common, "DiagnosticsStatusNotification", *status, func(station ocppclient.Station, ctx context.Context) error {
		return station.DiagnosticsStatusNotification(ctx, *status)
	})
}

func (a *App) firmwareStatus(args []string) error {
	fs := a.flagSet("firmware-status")
	var common commonFlags
	addCommonFlags(fs, &common)
	status := fs.String("status", "", "firmware update status")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: firmware-status does not accept positional arguments", config.ErrConfig)
	}
	if !validFirmwareStatus(*status) {
		return fmt.Errorf("%w: invalid firmware status %q", config.ErrConfig, *status)
	}
	return a.sendStatusOnly(fs, common, "FirmwareStatusNotification", *status, func(station ocppclient.Station, ctx context.Context) error {
		return station.FirmwareStatusNotification(ctx, *status)
	})
}

func (a *App) securityEvent(args []string) error {
	fs := a.flagSet("security-event")
	var common commonFlags
	addCommonFlags(fs, &common)
	typ := fs.String("type", "", "security event type")
	techInfo := fs.String("tech-info", "", "additional technical information")
	timestamp := fs.String("timestamp", "now", "RFC3339 timestamp or now")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: security-event does not accept positional arguments", config.ErrConfig)
	}
	if *typ == "" || len(*typ) > 50 {
		return fmt.Errorf("%w: type is required and must not exceed 50 characters", config.ErrConfig)
	}
	if len(*techInfo) > 255 {
		return fmt.Errorf("%w: tech-info must not exceed 255 characters", config.ErrConfig)
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
	if err := station.SecurityEventNotification(ctx, ocppclient.SecurityEventRequest{Type: *typ, TechInfo: *techInfo, Timestamp: ts}); err != nil {
		return err
	}
	result := struct {
		Sent      bool   `json:"sent"`
		Action    string `json:"action"`
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
	}{true, "SecurityEventNotification", *typ, formatTime(ts)}
	return renderSnapshot(a.out, format, keyValueSnapshot(result,
		[2]string{"SENT", "true"},
		[2]string{"ACTION", "SecurityEventNotification"},
		[2]string{"TYPE", *typ},
		[2]string{"TIMESTAMP", formatTime(ts)},
	))
}

func (a *App) logStatus(args []string) error {
	fs := a.flagSet("log-status")
	var common commonFlags
	addCommonFlags(fs, &common)
	status := fs.String("status", "", "log upload status")
	requestID := fs.Int("request-id", -1, "GetLog request ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: log-status does not accept positional arguments", config.ErrConfig)
	}
	if !validLogStatus(*status) {
		return fmt.Errorf("%w: invalid log status %q", config.ErrConfig, *status)
	}
	if *requestID < 0 {
		return fmt.Errorf("%w: request-id is required and must be zero or greater", config.ErrConfig)
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
	if err := station.LogStatusNotification(ctx, ocppclient.LogStatusRequest{Status: *status, RequestID: *requestID}); err != nil {
		return err
	}
	result := struct {
		Sent      bool   `json:"sent"`
		Action    string `json:"action"`
		Status    string `json:"status"`
		RequestID int    `json:"request_id"`
	}{true, "LogStatusNotification", *status, *requestID}
	return renderSnapshot(a.out, format, keyValueSnapshot(result,
		[2]string{"SENT", "true"},
		[2]string{"ACTION", "LogStatusNotification"},
		[2]string{"STATUS", *status},
		[2]string{"REQUEST_ID", strconv.Itoa(*requestID)},
	))
}

func (a *App) signedFirmwareStatus(args []string) error {
	fs := a.flagSet("signed-firmware-status")
	var common commonFlags
	addCommonFlags(fs, &common)
	status := fs.String("status", "", "signed firmware update status")
	requestID := fs.Int("request-id", -1, "optional SignedUpdateFirmware request ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: signed-firmware-status does not accept positional arguments", config.ErrConfig)
	}
	if !validSignedFirmwareStatus(*status) {
		return fmt.Errorf("%w: invalid signed firmware status %q", config.ErrConfig, *status)
	}
	var requestIDValue *int
	if *requestID >= 0 {
		requestIDValue = requestID
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
	if err := station.SignedFirmwareStatusNotification(ctx, ocppclient.SignedFirmwareStatusRequest{Status: *status, RequestID: requestIDValue}); err != nil {
		return err
	}
	requestText := ""
	if requestIDValue != nil {
		requestText = strconv.Itoa(*requestIDValue)
	}
	result := struct {
		Sent      bool   `json:"sent"`
		Action    string `json:"action"`
		Status    string `json:"status"`
		RequestID *int   `json:"request_id,omitempty"`
	}{true, "SignedFirmwareStatusNotification", *status, requestIDValue}
	return renderSnapshot(a.out, format, keyValueSnapshot(result,
		[2]string{"SENT", "true"},
		[2]string{"ACTION", "SignedFirmwareStatusNotification"},
		[2]string{"STATUS", *status},
		[2]string{"REQUEST_ID", requestText},
	))
}

func (a *App) signCertificate(args []string) error {
	fs := a.flagSet("sign-certificate")
	var common commonFlags
	addCommonFlags(fs, &common)
	csrFile := fs.String("csr-file", "", "PEM certificate signing request file")
	certificateType := fs.String("certificate-type", "ChargingStationCertificate", "certificate signing use")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: sign-certificate does not accept positional arguments", config.ErrConfig)
	}
	if *csrFile == "" {
		return fmt.Errorf("%w: csr-file is required", config.ErrConfig)
	}
	if *certificateType != "" && *certificateType != "ChargingStationCertificate" {
		return fmt.Errorf("%w: unsupported certificate-type %q", config.ErrConfig, *certificateType)
	}
	csr, err := os.ReadFile(*csrFile)
	if err != nil {
		return fmt.Errorf("%w: read CSR file %q: %v", config.ErrConfig, *csrFile, err)
	}
	csrText := strings.TrimSpace(string(csr))
	if csrText == "" || len(csrText) > 5500 {
		return fmt.Errorf("%w: CSR must contain 1 to 5500 characters", config.ErrConfig)
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
	result, operationErr := station.SignCertificate(ctx, ocppclient.SignCertificateRequest{CSR: csrText, CertificateType: *certificateType})
	if operationErr != nil && !errors.Is(operationErr, ocppclient.ErrRejected) {
		return operationErr
	}
	renderErr := renderSnapshot(a.out, format, keyValueSnapshot(result, [2]string{"STATUS", result.Status}))
	if renderErr != nil {
		return renderErr
	}
	return operationErr
}

func (a *App) sendStatusOnly(fs *flag.FlagSet, common commonFlags, action, status string, send func(ocppclient.Station, context.Context) error) error {
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
	if err := send(station, ctx); err != nil {
		return err
	}
	result := struct {
		Sent   bool   `json:"sent"`
		Action string `json:"action"`
		Status string `json:"status"`
	}{true, action, status}
	return renderSnapshot(a.out, format, keyValueSnapshot(result,
		[2]string{"SENT", "true"},
		[2]string{"ACTION", action},
		[2]string{"STATUS", status},
	))
}

func parseJSONPayload(inline, file string) (any, error) {
	if inline != "" && file != "" {
		return nil, fmt.Errorf("%w: use either data or data-file, not both", config.ErrConfig)
	}
	var raw []byte
	if file != "" {
		contents, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("%w: read data file %q: %v", config.ErrConfig, file, err)
		}
		raw = contents
	} else if inline != "" {
		raw = []byte(inline)
	} else {
		return nil, nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%w: data payload must be valid JSON: %v", config.ErrConfig, err)
	}
	return value, nil
}

func compactJSON(value any) (string, error) {
	if value == nil {
		return "", nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal data transfer response: %w", err)
	}
	return string(encoded), nil
}
