package cli

import "github.com/DishanRajapaksha/industrial-cli-kit/command"

var cliRegistry = command.Registry{
	Binary: "ocpp-cli",
	GlobalFlags: []command.Flag{
		{Name: "config", TakesValue: true, Summary: "YAML config file"},
		{Name: "profile", TakesValue: true, Summary: "config profile name"},
		{Name: "central-system-url", TakesValue: true, Summary: "central system WebSocket base URL"},
		{Name: "charge-point-id", TakesValue: true, Summary: "charge point identity"},
		{Name: "username", TakesValue: true, Summary: "HTTP Basic authentication username"},
		{Name: "password", TakesValue: true, Summary: "HTTP Basic authentication password"},
		{Name: "ca-cert", TakesValue: true, Summary: "CA certificate PEM file"},
		{Name: "client-cert", TakesValue: true, Summary: "client certificate PEM file"},
		{Name: "client-key", TakesValue: true, Summary: "client private key PEM file"},
		{Name: "tls-server-name", TakesValue: true, Summary: "TLS server-name override"},
		{Name: "insecure-skip-verify", Summary: "skip TLS certificate verification"},
		{Name: "timeout", TakesValue: true, Summary: "connection and request timeout"},
		{Name: "format", TakesValue: true, Summary: "output format"},
		{Name: "verbose", Summary: "print high-level connection decisions"},
		{Name: "debug", Summary: "enable protocol debug logging"},
	},
	Commands: []command.Command{
		{Name: "run", Summary: "Run a persistent charge point simulator", Flags: flags("connectors", "heartbeat-interval", "meter-interval", "meter-start", "meter-step", "duration", "model", "vendor", "firmware-version")},
		{Name: "init-config", Summary: "Write a starter config.yaml", Flags: flags("output", "force")},
		{Name: "validate-config", Summary: "Validate local configuration without connecting"},
		{Name: "test-connection", Summary: "Open and close an OCPP WebSocket connection"},
		{Name: "status", Summary: "Alias for test-connection"},
		{Name: "send", Summary: "Send a named OCPP operation", Subcommands: ocppOperations()},
		{Name: "boot-notification", Summary: "Send BootNotification", Flags: flags("model", "vendor", "firmware-version", "serial-number", "meter-serial-number", "meter-type")},
		{Name: "heartbeat", Summary: "Send Heartbeat"},
		{Name: "authorize", Summary: "Send Authorize", Flags: flags("id-tag")},
		{Name: "status-notification", Summary: "Send StatusNotification", Flags: flags("connector", "status", "error-code", "info", "vendor-id", "vendor-error-code", "timestamp")},
		{Name: "meter-values", Summary: "Send one MeterValues sample", Flags: flags("connector", "transaction-id", "value", "measurand", "unit", "context", "location", "phase", "timestamp")},
		{Name: "start-transaction", Summary: "Send StartTransaction", Flags: flags("connector", "id-tag", "meter-start", "reservation-id", "timestamp", "yes", "dry-run")},
		{Name: "stop-transaction", Summary: "Send StopTransaction", Flags: flags("transaction-id", "meter-stop", "id-tag", "reason", "timestamp", "yes", "dry-run")},
		{Name: "data-transfer", Summary: "Send vendor-specific JSON data", Flags: flags("vendor-id", "message-id", "data", "data-file")},
		{Name: "diagnostics-status", Summary: "Send DiagnosticsStatusNotification", Flags: flags("status")},
		{Name: "firmware-status", Summary: "Send FirmwareStatusNotification", Flags: flags("status")},
		{Name: "security-event", Summary: "Send SecurityEventNotification", Flags: flags("type", "tech-info", "timestamp")},
		{Name: "log-status", Summary: "Send LogStatusNotification", Flags: flags("status", "request-id")},
		{Name: "signed-firmware-status", Summary: "Send SignedFirmwareStatusNotification", Flags: flags("status", "request-id")},
		{Name: "sign-certificate", Summary: "Send a PEM certificate signing request", Flags: flags("csr-file", "certificate-type")},
		{Name: "completions", Summary: "Generate bash or zsh completion scripts"},
		{Name: "help", Summary: "Print help"},
		{Name: "version", Summary: "Print version information"},
	},
}

func flags(names ...string) []command.Flag {
	result := make([]command.Flag, 0, len(names))
	for _, name := range names {
		takesValue := name != "force" && name != "yes" && name != "dry-run"
		result = append(result, command.Flag{Name: name, TakesValue: takesValue})
	}
	return result
}

func ocppOperations() []command.Command {
	operations := []string{
		"boot-notification", "heartbeat", "authorize", "status-notification", "meter-values",
		"start-transaction", "stop-transaction", "data-transfer", "diagnostics-status",
		"firmware-status", "security-event", "log-status", "signed-firmware-status", "sign-certificate",
	}
	result := make([]command.Command, 0, len(operations))
	for _, operation := range operations {
		for _, candidate := range cliRegistry.Commands {
			if candidate.Name == operation {
				result = append(result, candidate)
				break
			}
		}
	}
	return result
}
