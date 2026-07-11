package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigPath = "config.yaml"
	DefaultCentralSystemURL = "ws://localhost:8080/ocpp"
	DefaultChargePointID = "CP001"
)

type ClientConfig struct {
	CentralSystemURL string
	ChargePointID string
	Username string
	Password string
	CACertFile string
	ClientCertFile string
	ClientKeyFile string
	CACertPEM []byte
	ClientCertPEM []byte
	ClientKeyPEM []byte
	TLSServerName string
	InsecureSkipVerify bool
	Timeout time.Duration
	Format string
	Verbose bool
	Debug bool
	ChargePointModel string
	ChargePointVendor string
	FirmwareVersion string
	SerialNumber string
	MeterSerialNumber string
	MeterType string
}

func DefaultClientConfig() ClientConfig {
	return ClientConfig{CentralSystemURL: DefaultCentralSystemURL, ChargePointID: DefaultChargePointID, Timeout: 10 * time.Second, Format: "table", ChargePointModel: "ocpp-cli", ChargePointVendor: "DishanRajapaksha"}
}

type configFile struct {
	settings `yaml:",inline"`
	DefaultProfile string `yaml:"default_profile"`
	Profiles map[string]settings `yaml:"profiles"`
}

type settings struct {
	CentralSystemURL string `yaml:"central_system_url"`
	ChargePointID string `yaml:"charge_point_id"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	CACertFile string `yaml:"ca_cert"`
	ClientCertFile string `yaml:"client_cert"`
	ClientKeyFile string `yaml:"client_key"`
	CACertBase64 string `yaml:"ca_cert_base64"`
	ClientCertBase64 string `yaml:"client_cert_base64"`
	ClientKeyBase64 string `yaml:"client_key_base64"`
	TLSServerName string `yaml:"tls_server_name"`
	InsecureSkipVerify *bool `yaml:"insecure_skip_verify"`
	Timeout string `yaml:"timeout"`
	Format string `yaml:"format"`
	ChargePointModel string `yaml:"charge_point_model"`
	ChargePointVendor string `yaml:"charge_point_vendor"`
	FirmwareVersion string `yaml:"firmware_version"`
	SerialNumber string `yaml:"charge_point_serial_number"`
	MeterSerialNumber string `yaml:"meter_serial_number"`
	MeterType string `yaml:"meter_type"`
}

func LoadClientConfigForProfile(path, profile string) (ClientConfig, error) {
	cfg := DefaultClientConfig()
	if path == "" { return cfg, nil }
	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && path == DefaultConfigPath { return cfg, nil }
		return cfg, fmt.Errorf("%w: read config %q: %v", ErrConfig, path, err)
	}
	var file configFile
	if err := yaml.Unmarshal(contents, &file); err != nil { return cfg, fmt.Errorf("%w: parse config %q: %v", ErrConfig, path, err) }
	if err := applySettings(&cfg, file.settings); err != nil { return cfg, err }
	if len(file.Profiles) > 0 {
		selected := profile
		if selected == "" { selected = file.DefaultProfile }
		if selected == "" { return cfg, fmt.Errorf("%w: config has profiles but no profile was selected and default_profile is empty", ErrConfig) }
		profileSettings, ok := file.Profiles[selected]
		if !ok { return cfg, fmt.Errorf("%w: profile %q not found", ErrConfig, selected) }
		if err := applySettings(&cfg, profileSettings); err != nil { return cfg, fmt.Errorf("%w: profile %q: %v", ErrConfig, selected, err) }
	}
	if err := ValidateClientConfig(cfg); err != nil { return cfg, err }
	return cfg, nil
}

func applySettings(cfg *ClientConfig, s settings) error {
	if s.CentralSystemURL != "" { cfg.CentralSystemURL = s.CentralSystemURL }
	if s.ChargePointID != "" { cfg.ChargePointID = s.ChargePointID }
	if s.Username != "" { cfg.Username = s.Username }
	if s.Password != "" { cfg.Password = s.Password }
	if s.CACertFile != "" { cfg.CACertFile = s.CACertFile }
	if s.ClientCertFile != "" { cfg.ClientCertFile = s.ClientCertFile }
	if s.ClientKeyFile != "" { cfg.ClientKeyFile = s.ClientKeyFile }
	var err error
	if s.CACertBase64 != "" { cfg.CACertPEM, err = decodeBase64(s.CACertBase64); if err != nil { return fmt.Errorf("%w: parse ca_cert_base64: %v", ErrConfig, err) } }
	if s.ClientCertBase64 != "" { cfg.ClientCertPEM, err = decodeBase64(s.ClientCertBase64); if err != nil { return fmt.Errorf("%w: parse client_cert_base64: %v", ErrConfig, err) } }
	if s.ClientKeyBase64 != "" { cfg.ClientKeyPEM, err = decodeBase64(s.ClientKeyBase64); if err != nil { return fmt.Errorf("%w: parse client_key_base64: %v", ErrConfig, err) } }
	if s.TLSServerName != "" { cfg.TLSServerName = s.TLSServerName }
	if s.InsecureSkipVerify != nil { cfg.InsecureSkipVerify = *s.InsecureSkipVerify }
	if s.Timeout != "" { cfg.Timeout, err = time.ParseDuration(s.Timeout); if err != nil { return fmt.Errorf("%w: parse timeout %q: %v", ErrConfig, s.Timeout, err) } }
	if s.Format != "" { cfg.Format = s.Format }
	if s.ChargePointModel != "" { cfg.ChargePointModel = s.ChargePointModel }
	if s.ChargePointVendor != "" { cfg.ChargePointVendor = s.ChargePointVendor }
	if s.FirmwareVersion != "" { cfg.FirmwareVersion = s.FirmwareVersion }
	if s.SerialNumber != "" { cfg.SerialNumber = s.SerialNumber }
	if s.MeterSerialNumber != "" { cfg.MeterSerialNumber = s.MeterSerialNumber }
	if s.MeterType != "" { cfg.MeterType = s.MeterType }
	return nil
}

func ValidateClientConfig(cfg ClientConfig) error {
	if strings.TrimSpace(cfg.CentralSystemURL) == "" { return fmt.Errorf("%w: central_system_url cannot be empty", ErrConfig) }
	u, err := url.Parse(cfg.CentralSystemURL)
	if err != nil || (u.Scheme != "ws" && u.Scheme != "wss") || u.Host == "" { return fmt.Errorf("%w: central_system_url must be an absolute ws:// or wss:// URL", ErrConfig) }
	if strings.TrimSpace(cfg.ChargePointID) == "" { return fmt.Errorf("%w: charge_point_id cannot be empty", ErrConfig) }
	if strings.Contains(cfg.ChargePointID, "/") { return fmt.Errorf("%w: charge_point_id cannot contain '/'", ErrConfig) }
	if cfg.Timeout <= 0 { return fmt.Errorf("%w: timeout must be greater than zero", ErrConfig) }
	if (cfg.Username == "") != (cfg.Password == "") { return fmt.Errorf("%w: username and password must be configured together", ErrConfig) }
	if (cfg.ClientCertFile == "") != (cfg.ClientKeyFile == "") { return fmt.Errorf("%w: client_cert and client_key must be configured together", ErrConfig) }
	if (len(cfg.ClientCertPEM) == 0) != (len(cfg.ClientKeyPEM) == 0) { return fmt.Errorf("%w: client_cert_base64 and client_key_base64 must be configured together", ErrConfig) }
	if cfg.ClientCertFile != "" && len(cfg.ClientCertPEM) > 0 { return fmt.Errorf("%w: choose file-based or base64 client certificates, not both", ErrConfig) }
	if cfg.ChargePointModel == "" || len(cfg.ChargePointModel) > 20 { return fmt.Errorf("%w: charge_point_model must contain 1 to 20 characters", ErrConfig) }
	if cfg.ChargePointVendor == "" || len(cfg.ChargePointVendor) > 20 { return fmt.Errorf("%w: charge_point_vendor must contain 1 to 20 characters", ErrConfig) }
	return nil
}

func StarterConfigYAML() []byte {
	return []byte(`# ocpp-cli uses OCPP 1.6-J over WebSocket.
default_profile: local

profiles:
  local:
    central_system_url: ws://localhost:8080/ocpp
    charge_point_id: CP001
    timeout: 10s
    format: table

    # Optional HTTP Basic authentication for the WebSocket handshake.
    username: ""
    password: ""

    # TLS options for wss:// endpoints. PEM data may instead be supplied
    # through ca_cert_base64, client_cert_base64, and client_key_base64.
    ca_cert: ""
    client_cert: ""
    client_key: ""
    tls_server_name: ""
    insecure_skip_verify: false

    # BootNotification defaults.
    charge_point_model: ocpp-cli
    charge_point_vendor: DishanRajapaksha
    firmware_version: ""
    charge_point_serial_number: ""
    meter_serial_number: ""
    meter_type: ""
`)
}

func decodeBase64(value string) ([]byte, error) {
	compact := strings.Join(strings.Fields(value), "")
	decoded, err := base64.StdEncoding.DecodeString(compact)
	if err == nil { return decoded, nil }
	return base64.RawStdEncoding.DecodeString(compact)
}
