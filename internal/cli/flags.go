package cli

import (
	"flag"
	"fmt"
	"strings"
	"time"

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
	fs.StringVar(&flags.format, "format", "", "output format: table, text, json, or csv")
	fs.BoolVar(&flags.verbose, "verbose", false, "print high-level connection decisions")
	fs.BoolVar(&flags.debug, "debug", false, "enable protocol debug logging")
}

func resolveCommon(fs *flag.FlagSet, flags commonFlags) (config.ClientConfig, string, error) {
	cfg, err := config.LoadClientConfigForProfile(flags.configPath, flags.profile)
	if err != nil {
		return cfg, "", err
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
		return cfg, "", err
	}
	format, err := validateSnapshotFormat(cfg.Format)
	if err != nil {
		return cfg, "", fmt.Errorf("%w: %v", config.ErrConfig, err)
	}
	return cfg, format, nil
}

func normaliseGlobalFlags(args []string) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}
	var globals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("command is required after --")
			}
			return appendCommandGlobals(args[i+1:], globals), nil
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			return appendCommandGlobals(args[i:], globals), nil
		}
		if arg == "--help" || arg == "-h" || arg == "--version" || arg == "-v" {
			return args[i:], nil
		}
		name, inline, hasInline := strings.Cut(arg, "=")
		switch name {
		case "--verbose", "--debug", "--insecure-skip-verify":
			if hasInline {
				return nil, fmt.Errorf("%s does not take a value", name)
			}
			globals = append(globals, name)
		case "--config", "--profile", "--central-system-url", "--charge-point-id", "--username", "--password", "--ca-cert", "--client-cert", "--client-key", "--tls-server-name", "--timeout", "--format":
			value := inline
			if !hasInline {
				i++
				if i >= len(args) || strings.HasPrefix(args[i], "-") {
					return nil, fmt.Errorf("%s requires a value", name)
				}
				value = args[i]
			}
			if value == "" {
				return nil, fmt.Errorf("%s requires a value", name)
			}
			globals = append(globals, name, value)
		default:
			return nil, fmt.Errorf("unknown global flag %q", name)
		}
	}
	return nil, fmt.Errorf("command is required")
}

func appendCommandGlobals(args, globals []string) []string {
	if len(args) == 0 || len(globals) == 0 {
		return args
	}
	if !commandSupportsCommonFlags(args[0]) {
		return args
	}
	out := make([]string, 0, len(args)+len(globals))
	out = append(out, args[0])
	out = append(out, globals...)
	out = append(out, args[1:]...)
	return out
}

func commandSupportsCommonFlags(command string) bool {
	switch command {
	case "validate-config", "test-connection", "boot-notification", "heartbeat", "authorize", "status-notification", "meter-values", "start-transaction", "stop-transaction", "data-transfer", "diagnostics-status", "firmware-status", "security-event", "log-status", "signed-firmware-status", "sign-certificate":
		return true
	default:
		return false
	}
}
