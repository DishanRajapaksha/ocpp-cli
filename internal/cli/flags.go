package cli

import (
	"flag"
	"fmt"
	"time"

	"github.com/DishanRajapaksha/industrial-cli-kit/command"
	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
)

type commonFlags struct {
	configPath         string
	profile            string
	centralSystemURL   string
	chargePointID      string
	username           string
	password           string
	caCert             string
	clientCert         string
	clientKey          string
	tlsServerName      string
	insecureSkipVerify bool
	timeout            time.Duration
	format             string
	verbose            bool
	debug              bool
}

func addCommonFlags(fs *flag.FlagSet, flags *commonFlags) {
	fs.StringVar(&flags.configPath, "config", config.DefaultConfigPath, "YAML config file")
	fs.StringVar(&flags.profile, "profile", "", "config profile name")
	fs.StringVar(&flags.centralSystemURL, "central-system-url", "", "central system WebSocket base URL")
	fs.StringVar(&flags.chargePointID, "charge-point-id", "", "charge point identity appended to the WebSocket URL")
	fs.StringVar(&flags.username, "username", "", "HTTP Basic authentication username")
	fs.StringVar(&flags.password, "password", "", "HTTP Basic authentication password")
	fs.StringVar(&flags.caCert, "ca-cert", "", "CA certificate PEM file")
	fs.StringVar(&flags.clientCert, "client-cert", "", "client certificate PEM file")
	fs.StringVar(&flags.clientKey, "client-key", "", "client private key PEM file")
	fs.StringVar(&flags.tlsServerName, "tls-server-name", "", "override TLS server name")
	fs.BoolVar(&flags.insecureSkipVerify, "insecure-skip-verify", false, "skip TLS certificate verification; diagnostics only")
	fs.DurationVar(&flags.timeout, "timeout", 0, "connection and request timeout")
	fs.StringVar(&flags.format, "format", "", "output format")
	fs.BoolVar(&flags.verbose, "verbose", false, "print high-level connection decisions")
	fs.BoolVar(&flags.debug, "debug", false, "enable protocol debug logging")
}

func resolveClientConfig(fs *flag.FlagSet, flags commonFlags) (config.ClientConfig, error) {
	cfg, err := config.LoadClientConfigForProfile(flags.configPath, flags.profile)
	if err != nil {
		return cfg, err
	}
	visited := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { visited[f.Name] = true })
	if visited["central-system-url"] {
		cfg.CentralSystemURL = flags.centralSystemURL
	}
	if visited["charge-point-id"] {
		cfg.ChargePointID = flags.chargePointID
	}
	if visited["username"] {
		cfg.Username = flags.username
	}
	if visited["password"] {
		cfg.Password = flags.password
	}
	if visited["ca-cert"] {
		cfg.CACertFile = flags.caCert
		cfg.CACertPEM = nil
	}
	if visited["client-cert"] {
		cfg.ClientCertFile = flags.clientCert
		cfg.ClientCertPEM = nil
	}
	if visited["client-key"] {
		cfg.ClientKeyFile = flags.clientKey
		cfg.ClientKeyPEM = nil
	}
	if visited["tls-server-name"] {
		cfg.TLSServerName = flags.tlsServerName
	}
	if visited["insecure-skip-verify"] {
		cfg.InsecureSkipVerify = flags.insecureSkipVerify
	}
	if visited["timeout"] {
		cfg.Timeout = flags.timeout
	}
	if visited["format"] {
		cfg.Format = flags.format
	}
	if visited["verbose"] {
		cfg.Verbose = flags.verbose
	}
	if visited["debug"] {
		cfg.Debug = flags.debug
	}
	if err := config.ValidateClientConfig(cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func resolveCommon(fs *flag.FlagSet, flags commonFlags) (config.ClientConfig, string, error) {
	cfg, err := resolveClientConfig(fs, flags)
	if err != nil {
		return cfg, "", err
	}
	format, err := validateSnapshotFormat(cfg.Format)
	if err != nil {
		return cfg, "", fmt.Errorf("%w: %v", config.ErrConfig, err)
	}
	return cfg, format, nil
}

func normaliseGlobalFlags(args []string) ([]string, error) {
	return command.NormalizeGlobalFlags(args, cliRegistry.GlobalFlags)
}
