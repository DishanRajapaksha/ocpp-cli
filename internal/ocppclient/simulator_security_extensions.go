package ocppclient

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/certificates"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/extendedtriggermessage"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/logging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/securefirmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/security"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

type installedCertificate struct {
	certificateType types.CertificateUse
	hash            types.CertificateHashData
	pem             string
}

// Security profile.
func (s *simulator) OnCertificateSigned(request *security.CertificateSignedRequest) (*security.CertificateSignedResponse, error) {
	status := security.CertificateSignedStatusAccepted
	if _, err := parseFirstCertificate(request.CertificateChain); err != nil {
		status = security.CertificateSignedStatusRejected
	} else {
		s.mu.Lock()
		s.stationCertificateChain = request.CertificateChain
		s.mu.Unlock()
	}
	s.emit("inbound", security.CertificateSignedFeatureName, string(status), nil, fmt.Sprintf("chain_bytes=%d", len(request.CertificateChain)))
	return security.NewCertificateSignedResponse(status), nil
}

// Certificates profile.
func (s *simulator) OnInstallCertificate(request *certificates.InstallCertificateRequest) (*certificates.InstallCertificateResponse, error) {
	certificate, err := parseFirstCertificate(request.Certificate)
	if err != nil {
		s.emit("inbound", certificates.InstallCertificateFeatureName, string(certificates.CertificateStatusFailed), nil, err.Error())
		return certificates.NewInstallCertificateResponse(certificates.CertificateStatusFailed), nil
	}
	hash := certificateHash(certificate)
	key := certificateHashKey(hash)
	s.mu.Lock()
	s.installedCertificates[key] = installedCertificate{certificateType: request.CertificateType, hash: hash, pem: request.Certificate}
	s.mu.Unlock()
	s.emit("inbound", certificates.InstallCertificateFeatureName, string(certificates.CertificateStatusAccepted), nil, string(request.CertificateType)+":"+hash.SerialNumber)
	return certificates.NewInstallCertificateResponse(certificates.CertificateStatusAccepted), nil
}

func (s *simulator) OnGetInstalledCertificateIds(request *certificates.GetInstalledCertificateIdsRequest) (*certificates.GetInstalledCertificateIdsResponse, error) {
	s.mu.Lock()
	entries := make([]types.CertificateHashData, 0, len(s.installedCertificates))
	for _, certificate := range s.installedCertificates {
		if certificate.certificateType == request.CertificateType {
			entries = append(entries, certificate.hash)
		}
	}
	s.mu.Unlock()
	sort.Slice(entries, func(i, j int) bool { return certificateHashKey(entries[i]) < certificateHashKey(entries[j]) })
	status := certificates.GetInstalledCertificateStatusAccepted
	if len(entries) == 0 {
		status = certificates.GetInstalledCertificateStatusNotFound
	}
	response := certificates.NewGetInstalledCertificateIdsResponse(status)
	response.CertificateHashData = entries
	s.emit("inbound", certificates.GetInstalledCertificateIdsFeatureName, string(status), nil, fmt.Sprintf("type=%s count=%d", request.CertificateType, len(entries)))
	return response, nil
}

func (s *simulator) OnDeleteCertificate(request *certificates.DeleteCertificateRequest) (*certificates.DeleteCertificateResponse, error) {
	key := certificateHashKey(request.CertificateHashData)
	s.mu.Lock()
	_, found := s.installedCertificates[key]
	if found {
		delete(s.installedCertificates, key)
	}
	s.mu.Unlock()
	status := certificates.DeleteCertificateStatusNotFound
	if found {
		status = certificates.DeleteCertificateStatusAccepted
	}
	s.emit("inbound", certificates.DeleteCertificateFeatureName, string(status), nil, request.CertificateHashData.SerialNumber)
	return certificates.NewDeleteCertificateResponse(status), nil
}

func parseFirstCertificate(value string) (*x509.Certificate, error) {
	rest := []byte(value)
	for len(rest) > 0 {
		block, remaining := pem.Decode(rest)
		rest = remaining
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse certificate: %w", err)
		}
		return certificate, nil
	}
	return nil, fmt.Errorf("no PEM encoded certificate found")
}

func certificateHash(certificate *x509.Certificate) types.CertificateHashData {
	issuerName := sha256.Sum256(certificate.RawIssuer)
	issuerKey := sha256.Sum256(certificate.RawSubjectPublicKeyInfo)
	return types.CertificateHashData{
		HashAlgorithm:  types.SHA256,
		IssuerNameHash: hex.EncodeToString(issuerName[:]),
		IssuerKeyHash:  hex.EncodeToString(issuerKey[:]),
		SerialNumber:   strings.ToUpper(certificate.SerialNumber.Text(16)),
	}
}

func certificateHashKey(hash types.CertificateHashData) string {
	return strings.Join([]string{string(hash.HashAlgorithm), strings.ToLower(hash.IssuerNameHash), strings.ToLower(hash.IssuerKeyHash), strings.ToLower(hash.SerialNumber)}, ":")
}

// Log profile.
func (s *simulator) OnGetLog(request *logging.GetLogRequest) (*logging.GetLogResponse, error) {
	s.mu.Lock()
	status := logging.LogStatusAccepted
	if s.logStatus == logging.UploadLogStatusUploading && s.activeLogRequestID != request.RequestID {
		status = logging.LogStatusAcceptedCanceled
	}
	s.activeLogRequestID = request.RequestID
	s.logStatus = logging.UploadLogStatusUploading
	s.mu.Unlock()

	filename := fmt.Sprintf("ocpp-cli-%s-%d.log", strings.ToLower(string(request.LogType)), request.RequestID)
	response := logging.NewGetLogResponse(status)
	response.Filename = filename
	s.emit("inbound", logging.GetLogFeatureName, string(status), nil, request.Log.RemoteLocation+"/"+filename)
	if s.cp.IsConnected() {
		go s.simulateLogUpload(request.RequestID)
	}
	return response, nil
}

func (s *simulator) simulateLogUpload(requestID int) {
	ctx := s.runContext()
	if err := s.sendLogStatus(ctx, requestID, logging.UploadLogStatusUploading); err != nil {
		s.emit("outbound", logging.LogStatusNotificationFeatureName, "Error", nil, err.Error())
		return
	}
	if !waitContext(ctx, 150*time.Millisecond) {
		return
	}
	if err := s.sendLogStatus(ctx, requestID, logging.UploadLogStatusUploaded); err != nil {
		s.emit("outbound", logging.LogStatusNotificationFeatureName, "Error", nil, err.Error())
	}
}

func (s *simulator) sendLogStatus(ctx context.Context, requestID int, status logging.UploadLogStatus) error {
	s.mu.Lock()
	if s.activeLogRequestID != 0 && s.activeLogRequestID != requestID {
		s.mu.Unlock()
		return nil
	}
	s.activeLogRequestID = requestID
	s.logStatus = status
	s.mu.Unlock()
	if s.cp.IsConnected() {
		_, err := await(ctx, func() (*logging.LogStatusNotificationResponse, error) {
			return s.cp.LogStatusNotification(status, requestID)
		})
		if err != nil {
			return classify(err, ErrProtocol)
		}
	}
	s.emit("outbound", logging.LogStatusNotificationFeatureName, string(status), nil, fmt.Sprintf("request_id=%d", requestID))
	return nil
}

// Secure firmware profile.
func (s *simulator) OnSignedUpdateFirmware(request *securefirmware.SignedUpdateFirmwareRequest) (*securefirmware.SignedUpdateFirmwareResponse, error) {
	if request.Firmware.RetrieveDateTime == nil || strings.TrimSpace(request.Firmware.Location) == "" {
		s.emit("inbound", securefirmware.SignedUpdateFirmwareFeatureName, string(securefirmware.UpdateFirmwareStatusRejected), nil, "missing firmware location or retrieve date")
		return securefirmware.NewSignedUpdateFirmwareResponse(securefirmware.UpdateFirmwareStatusRejected), nil
	}
	if strings.TrimSpace(request.Firmware.SigningCertificate) == "" || strings.TrimSpace(request.Firmware.Signature) == "" {
		s.emit("inbound", securefirmware.SignedUpdateFirmwareFeatureName, string(securefirmware.UpdateFirmwareStatusInvalidCertificate), nil, "missing signing certificate or signature")
		return securefirmware.NewSignedUpdateFirmwareResponse(securefirmware.UpdateFirmwareStatusInvalidCertificate), nil
	}

	s.mu.Lock()
	status := securefirmware.UpdateFirmwareStatusAccepted
	if signedFirmwareActive(s.signedFirmwareStatus) && s.signedFirmwareRequestID != request.RequestID {
		status = securefirmware.UpdateFirmwareStatusAcceptedCanceled
	}
	s.signedFirmwareRequestID = request.RequestID
	s.signedFirmwareStatus = securefirmware.FirmwareStatusDownloadScheduled
	s.mu.Unlock()
	s.emit("inbound", securefirmware.SignedUpdateFirmwareFeatureName, string(status), nil, fmt.Sprintf("request_id=%d location=%s", request.RequestID, request.Firmware.Location))
	if s.cp.IsConnected() {
		go s.simulateSignedFirmwareUpdate(request)
	}
	return securefirmware.NewSignedUpdateFirmwareResponse(status), nil
}

func signedFirmwareActive(status securefirmware.FirmwareStatus) bool {
	switch status {
	case securefirmware.FirmwareStatusDownloadScheduled,
		securefirmware.FirmwareStatusDownloading,
		securefirmware.FirmwareStatusDownloaded,
		securefirmware.FirmwareStatusSignatureVerified,
		securefirmware.FirmwareStatusInstallScheduled,
		securefirmware.FirmwareStatusInstalling:
		return true
	default:
		return false
	}
}

func (s *simulator) simulateSignedFirmwareUpdate(request *securefirmware.SignedUpdateFirmwareRequest) {
	ctx := s.runContext()
	retrieveAt := request.Firmware.RetrieveDateTime.Time
	if delay := time.Until(retrieveAt); delay > 0 && !waitContext(ctx, delay) {
		return
	}
	sequence := []securefirmware.FirmwareStatus{
		securefirmware.FirmwareStatusDownloading,
		securefirmware.FirmwareStatusDownloaded,
		securefirmware.FirmwareStatusSignatureVerified,
	}
	for _, status := range sequence {
		if err := s.sendSignedFirmwareStatus(ctx, request.RequestID, status); err != nil {
			s.emit("outbound", securefirmware.SignedFirmwareStatusNotificationFeatureName, "Error", nil, err.Error())
			return
		}
		if !waitContext(ctx, 100*time.Millisecond) {
			return
		}
	}
	if request.Firmware.InstallDateTime != nil {
		if delay := time.Until(request.Firmware.InstallDateTime.Time); delay > 0 && !waitContext(ctx, delay) {
			return
		}
	}
	for _, status := range []securefirmware.FirmwareStatus{securefirmware.FirmwareStatusInstalling, securefirmware.FirmwareStatusInstalled} {
		if err := s.sendSignedFirmwareStatus(ctx, request.RequestID, status); err != nil {
			s.emit("outbound", securefirmware.SignedFirmwareStatusNotificationFeatureName, "Error", nil, err.Error())
			return
		}
		if !waitContext(ctx, 100*time.Millisecond) {
			return
		}
	}
}

func (s *simulator) sendSignedFirmwareStatus(ctx context.Context, requestID int, status securefirmware.FirmwareStatus) error {
	s.mu.Lock()
	if s.signedFirmwareRequestID != 0 && s.signedFirmwareRequestID != requestID {
		s.mu.Unlock()
		return nil
	}
	s.signedFirmwareRequestID = requestID
	s.signedFirmwareStatus = status
	s.mu.Unlock()
	if s.cp.IsConnected() {
		_, err := await(ctx, func() (*securefirmware.SignedFirmwareStatusNotificationResponse, error) {
			return s.cp.SignedUpdateFirmwareStatusNotification(status, func(notification *securefirmware.SignedFirmwareStatusNotificationRequest) {
				notification.RequestID = &requestID
			})
		})
		if err != nil {
			return classify(err, ErrProtocol)
		}
	}
	s.emit("outbound", securefirmware.SignedFirmwareStatusNotificationFeatureName, string(status), nil, fmt.Sprintf("request_id=%d", requestID))
	return nil
}

// Extended trigger profile.
func (s *simulator) OnExtendedTriggerMessage(request *extendedtriggermessage.ExtendedTriggerMessageRequest) (*extendedtriggermessage.ExtendedTriggerMessageResponse, error) {
	status := extendedtriggermessage.ExtendedTriggerMessageStatusAccepted
	if request.ConnectorId != nil && *request.ConnectorId > 0 {
		s.mu.Lock()
		_, exists := s.connectors[*request.ConnectorId]
		s.mu.Unlock()
		if !exists {
			status = extendedtriggermessage.ExtendedTriggerMessageStatusRejected
		}
	}
	if !extendedTriggerSupported(request.RequestedMessage) {
		status = extendedtriggermessage.ExtendedTriggerMessageStatusNotImplemented
	}
	s.emit("inbound", extendedtriggermessage.ExtendedTriggerMessageFeatureName, string(status), request.ConnectorId, string(request.RequestedMessage))
	if status == extendedtriggermessage.ExtendedTriggerMessageStatusAccepted && s.cp.IsConnected() {
		message := request.RequestedMessage
		connectorID := request.ConnectorId
		go func() {
			if waitContext(s.runContext(), 25*time.Millisecond) {
				s.executeExtendedTrigger(message, connectorID)
			}
		}()
	}
	return extendedtriggermessage.NewExtendedTriggerMessageResponse(status), nil
}

func extendedTriggerSupported(message extendedtriggermessage.ExtendedTriggerMessageType) bool {
	switch message {
	case extendedtriggermessage.ExtendedTriggerMessageTypeBootNotification,
		extendedtriggermessage.ExtendedTriggerMessageTypeHeartbeat,
		extendedtriggermessage.ExtendedTriggerMessageTypeMeterValues,
		extendedtriggermessage.ExtendedTriggerMessageTypeStatusNotification,
		extendedtriggermessage.ExtendedTriggerMessageTypeFirmwareStatusNotification,
		extendedtriggermessage.ExtendedTriggerMessageTypeLogStatusNotification,
		extendedtriggermessage.ExtendedTriggerMessageTypeSignChargingStationCertificate:
		return true
	default:
		return false
	}
}

func (s *simulator) executeExtendedTrigger(message extendedtriggermessage.ExtendedTriggerMessageType, connectorID *int) {
	ctx := s.runContext()
	var err error
	switch message {
	case extendedtriggermessage.ExtendedTriggerMessageTypeBootNotification:
		_, err = s.sendBootNotification(ctx)
	case extendedtriggermessage.ExtendedTriggerMessageTypeHeartbeat:
		err = s.sendHeartbeat(ctx)
	case extendedtriggermessage.ExtendedTriggerMessageTypeMeterValues:
		err = s.sendTriggeredMeterValues(ctx, connectorID)
	case extendedtriggermessage.ExtendedTriggerMessageTypeStatusNotification:
		err = s.sendTriggeredStatuses(ctx, connectorID)
	case extendedtriggermessage.ExtendedTriggerMessageTypeFirmwareStatusNotification:
		s.mu.Lock()
		status := s.firmwareStatus
		s.mu.Unlock()
		err = s.sendFirmwareStatus(ctx, status)
	case extendedtriggermessage.ExtendedTriggerMessageTypeLogStatusNotification:
		s.mu.Lock()
		requestID, status := s.activeLogRequestID, s.logStatus
		s.mu.Unlock()
		err = s.sendLogStatus(ctx, requestID, status)
	case extendedtriggermessage.ExtendedTriggerMessageTypeSignChargingStationCertificate:
		var csr string
		csr, err = createCertificateSigningRequest(s.cfg.ChargePointID)
		if err == nil {
			_, err = await(ctx, func() (*security.SignCertificateResponse, error) { return s.cp.SignCertificate(csr) })
			if err == nil {
				s.emit("outbound", security.SignCertificateFeatureName, "Sent", nil, "generated ECDSA P-256 CSR")
			}
		}
	}
	if err != nil {
		s.emit("outbound", string(message), "Error", connectorID, err.Error())
	}
}

func createCertificateSigningRequest(commonName string) (string, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("generate CSR key: %w", err)
	}
	requestDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{Subject: pkix.Name{CommonName: commonName}}, privateKey)
	if err != nil {
		return "", fmt.Errorf("create CSR: %w", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: requestDER})), nil
}

var (
	_ security.ChargePointHandler               = (*simulator)(nil)
	_ certificates.ChargePointHandler           = (*simulator)(nil)
	_ logging.ChargePointHandler                = (*simulator)(nil)
	_ extendedtriggermessage.ChargePointHandler = (*simulator)(nil)
	_ securefirmware.ChargePointHandler         = (*simulator)(nil)
)
