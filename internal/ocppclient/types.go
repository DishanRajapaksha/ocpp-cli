package ocppclient

import (
	"context"
	"time"
)

type Station interface {
	Connect(context.Context) error
	Close()
	Boot(context.Context, BootRequest) (BootResult, error)
	Heartbeat(context.Context) (HeartbeatResult, error)
	Authorize(context.Context, string) (AuthorizationResult, error)
	StatusNotification(context.Context, StatusRequest) error
	MeterValues(context.Context, MeterRequest) error
	StartTransaction(context.Context, StartTransactionRequest) (StartTransactionResult, error)
	StopTransaction(context.Context, StopTransactionRequest) (StopTransactionResult, error)
	DataTransfer(context.Context, DataTransferRequest) (DataTransferResult, error)
	DiagnosticsStatusNotification(context.Context, string) error
	FirmwareStatusNotification(context.Context, string) error
	SecurityEventNotification(context.Context, SecurityEventRequest) error
	LogStatusNotification(context.Context, LogStatusRequest) error
	SignedFirmwareStatusNotification(context.Context, SignedFirmwareStatusRequest) error
	SignCertificate(context.Context, SignCertificateRequest) (SignCertificateResult, error)
}

type BootRequest struct {
	Model             string
	Vendor            string
	FirmwareVersion   string
	SerialNumber      string
	MeterSerialNumber string
	MeterType         string
}

type BootResult struct {
	Status      string    `json:"status"`
	Interval    int       `json:"interval_seconds"`
	CurrentTime time.Time `json:"current_time"`
}

type HeartbeatResult struct {
	CurrentTime time.Time `json:"current_time"`
}

type AuthorizationResult struct {
	Status      string     `json:"status"`
	ExpiryDate  *time.Time `json:"expiry_date,omitempty"`
	ParentIDTag string     `json:"parent_id_tag,omitempty"`
}

type StatusRequest struct {
	ConnectorID     int
	Status          string
	ErrorCode       string
	Info            string
	VendorID        string
	VendorErrorCode string
	Timestamp       time.Time
}

type MeterRequest struct {
	ConnectorID   int
	TransactionID *int
	Value         string
	Measurand     string
	Unit          string
	Context       string
	Location      string
	Phase         string
	Timestamp     time.Time
}

type StartTransactionRequest struct {
	ConnectorID   int
	IDTag         string
	MeterStart    int
	ReservationID *int
	Timestamp     time.Time
}

type StartTransactionResult struct {
	TransactionID int                 `json:"transaction_id"`
	Authorization AuthorizationResult `json:"authorization"`
}

type StopTransactionRequest struct {
	TransactionID int
	MeterStop     int
	IDTag         string
	Reason        string
	Timestamp     time.Time
}

type StopTransactionResult struct {
	Authorization *AuthorizationResult `json:"authorization,omitempty"`
}

type DataTransferRequest struct {
	VendorID  string
	MessageID string
	Data      any
}

type DataTransferResult struct {
	Status string `json:"status"`
	Data   any    `json:"data,omitempty"`
}

type SecurityEventRequest struct {
	Type      string
	TechInfo  string
	Timestamp time.Time
}

type LogStatusRequest struct {
	Status    string
	RequestID int
}

type SignedFirmwareStatusRequest struct {
	Status    string
	RequestID *int
}

type SignCertificateRequest struct {
	CSR             string
	CertificateType string
}

type SignCertificateResult struct {
	Status string `json:"status"`
}
