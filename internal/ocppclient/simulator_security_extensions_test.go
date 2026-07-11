package ocppclient

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/certificates"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/extendedtriggermessage"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/logging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/securefirmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/security"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

func TestCertificateInstallListAndDelete(t *testing.T) {
	s := newExtensionSimulator(t)
	certificatePEM := testCertificatePEM(t, "Test Root")

	install, err := s.OnInstallCertificate(certificates.NewInstallCertificateRequest(types.CentralSystemRootCertificate, certificatePEM))
	if err != nil {
		t.Fatal(err)
	}
	if install.Status != certificates.CertificateStatusAccepted {
		t.Fatalf("install status = %s", install.Status)
	}

	list, err := s.OnGetInstalledCertificateIds(certificates.NewGetInstalledCertificateIdsRequest(types.CentralSystemRootCertificate))
	if err != nil {
		t.Fatal(err)
	}
	if list.Status != certificates.GetInstalledCertificateStatusAccepted || len(list.CertificateHashData) != 1 {
		t.Fatalf("unexpected certificate list: %#v", list)
	}

	deleteResponse, err := s.OnDeleteCertificate(certificates.NewDeleteCertificateRequest(list.CertificateHashData[0]))
	if err != nil {
		t.Fatal(err)
	}
	if deleteResponse.Status != certificates.DeleteCertificateStatusAccepted {
		t.Fatalf("delete status = %s", deleteResponse.Status)
	}

	missing, err := s.OnDeleteCertificate(certificates.NewDeleteCertificateRequest(list.CertificateHashData[0]))
	if err != nil {
		t.Fatal(err)
	}
	if missing.Status != certificates.DeleteCertificateStatusNotFound {
		t.Fatalf("second delete status = %s", missing.Status)
	}
}

func TestCertificateSignedValidation(t *testing.T) {
	s := newExtensionSimulator(t)

	rejected, err := s.OnCertificateSigned(security.NewCertificateSignedRequest("not a certificate"))
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Status != security.CertificateSignedStatusRejected {
		t.Fatalf("invalid chain status = %s", rejected.Status)
	}

	chain := testCertificatePEM(t, "Station")
	accepted, err := s.OnCertificateSigned(security.NewCertificateSignedRequest(chain))
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != security.CertificateSignedStatusAccepted {
		t.Fatalf("valid chain status = %s", accepted.Status)
	}
	if s.stationCertificateChain != chain {
		t.Fatal("station certificate chain was not retained")
	}
}

func TestLogRequestsCancelPreviousUpload(t *testing.T) {
	s := newExtensionSimulator(t)
	parameters := logging.LogParameters{RemoteLocation: "https://example.test/logs"}

	first, err := s.OnGetLog(logging.NewGetLogRequest(logging.LogTypeDiagnostics, 11, parameters))
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != logging.LogStatusAccepted || first.Filename == "" {
		t.Fatalf("unexpected first log response: %#v", first)
	}

	second, err := s.OnGetLog(logging.NewGetLogRequest(logging.LogTypeSecurity, 12, parameters))
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != logging.LogStatusAcceptedCanceled {
		t.Fatalf("second log status = %s", second.Status)
	}
	if s.activeLogRequestID != 12 || s.logStatus != logging.UploadLogStatusUploading {
		t.Fatalf("unexpected log state: request=%d status=%s", s.activeLogRequestID, s.logStatus)
	}
}

func TestSignedFirmwareValidationAndScheduling(t *testing.T) {
	s := newExtensionSimulator(t)
	retrieveDate := types.NewDateTime(time.Now().UTC())
	firmware := securefirmware.Firmware{Location: "https://example.test/firmware.bin", RetrieveDateTime: retrieveDate}

	invalid, err := s.OnSignedUpdateFirmware(securefirmware.NewSignedUpdateFirmwareRequest(7, firmware))
	if err != nil {
		t.Fatal(err)
	}
	if invalid.Status != securefirmware.UpdateFirmwareStatusInvalidCertificate {
		t.Fatalf("unsigned firmware status = %s", invalid.Status)
	}

	firmware.SigningCertificate = testCertificatePEM(t, "Firmware Signer")
	firmware.Signature = "c2lnbmF0dXJl"
	accepted, err := s.OnSignedUpdateFirmware(securefirmware.NewSignedUpdateFirmwareRequest(8, firmware))
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != securefirmware.UpdateFirmwareStatusAccepted {
		t.Fatalf("signed firmware status = %s", accepted.Status)
	}
	if s.signedFirmwareRequestID != 8 || s.signedFirmwareStatus != securefirmware.FirmwareStatusDownloadScheduled {
		t.Fatalf("unexpected signed firmware state: request=%d status=%s", s.signedFirmwareRequestID, s.signedFirmwareStatus)
	}
}

func TestExtendedTriggerDecisions(t *testing.T) {
	s := newExtensionSimulator(t)

	accepted, err := s.OnExtendedTriggerMessage(extendedtriggermessage.NewExtendedTriggerMessageRequest(extendedtriggermessage.ExtendedTriggerMessageTypeHeartbeat))
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != extendedtriggermessage.ExtendedTriggerMessageStatusAccepted {
		t.Fatalf("heartbeat trigger status = %s", accepted.Status)
	}

	unknown := extendedtriggermessage.NewExtendedTriggerMessageRequest(extendedtriggermessage.ExtendedTriggerMessageType("UnknownMessage"))
	notImplemented, err := s.OnExtendedTriggerMessage(unknown)
	if err != nil {
		t.Fatal(err)
	}
	if notImplemented.Status != extendedtriggermessage.ExtendedTriggerMessageStatusNotImplemented {
		t.Fatalf("unknown trigger status = %s", notImplemented.Status)
	}

	connectorID := 99
	badConnector := extendedtriggermessage.NewExtendedTriggerMessageRequest(extendedtriggermessage.ExtendedTriggerMessageTypeStatusNotification)
	badConnector.ConnectorId = &connectorID
	rejected, err := s.OnExtendedTriggerMessage(badConnector)
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Status != extendedtriggermessage.ExtendedTriggerMessageStatusRejected {
		t.Fatalf("bad connector trigger status = %s", rejected.Status)
	}
}

func newExtensionSimulator(t *testing.T) *simulator {
	t.Helper()
	instance, err := NewSimulator(config.DefaultClientConfig(), SimulatorOptions{Connectors: 1})
	if err != nil {
		t.Fatal(err)
	}
	s, ok := instance.(*simulator)
	if !ok {
		t.Fatalf("unexpected simulator type %T", instance)
	}
	return s
}

func testCertificatePEM(t *testing.T, commonName string) string {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(42),
		Subject:               pkix.Name{CommonName: commonName},
		Issuer:                pkix.Name{CommonName: commonName},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER}))
}
