package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
	"github.com/DishanRajapaksha/ocpp-cli/internal/ocppclient"
)

type stationFactory func(config.ClientConfig) (ocppclient.Station, error)
type simulatorFactory func(config.ClientConfig, ocppclient.SimulatorOptions) (ocppclient.Simulator, error)

type App struct {
	out          io.Writer
	err          io.Writer
	newStation   stationFactory
	newSimulator simulatorFactory
}

func NewApp(out, err io.Writer) *App {
	return &App{out: out, err: err, newStation: ocppclient.New, newSimulator: ocppclient.NewSimulator}
}

func Main() {
	code := NewApp(os.Stdout, os.Stderr).Run(os.Args[1:])
	if code != 0 {
		os.Exit(code)
	}
}

func (a *App) Run(args []string) int {
	normalised, err := normaliseGlobalFlags(args)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return exitConfigError
	}
	args = normalised
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		a.printUsage()
		return exitSuccess
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintln(a.out, "ocpp-cli development")
		err = nil
	case "run":
		err = a.runSimulator(args[1:])
	case "init-config":
		err = a.initConfig(args[1:])
	case "validate-config":
		err = a.validateConfig(args[1:])
	case "test-connection":
		err = a.testConnection(args[1:])
	case "status":
		err = a.testConnection(args[1:])
	case "send":
		err = a.send(args[1:])
	case "boot-notification":
		err = a.bootNotification(args[1:])
	case "heartbeat":
		err = a.heartbeat(args[1:])
	case "authorize":
		err = a.authorize(args[1:])
	case "status-notification":
		err = a.statusNotification(args[1:])
	case "meter-values":
		err = a.meterValues(args[1:])
	case "start-transaction":
		err = a.startTransaction(args[1:])
	case "stop-transaction":
		err = a.stopTransaction(args[1:])
	case "data-transfer":
		err = a.dataTransfer(args[1:])
	case "diagnostics-status":
		err = a.diagnosticsStatus(args[1:])
	case "firmware-status":
		err = a.firmwareStatus(args[1:])
	case "security-event":
		err = a.securityEvent(args[1:])
	case "log-status":
		err = a.logStatus(args[1:])
	case "signed-firmware-status":
		err = a.signedFirmwareStatus(args[1:])
	case "sign-certificate":
		err = a.signCertificate(args[1:])
	case "completions":
		err = a.completions(args[1:])
	default:
		a.printUsage()
		fmt.Fprintf(a.err, "unknown command %q\n", args[0])
		return exitGeneralError
	}
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitSuccess
		}
		fmt.Fprintln(a.err, err)
		return mapExitCode(err)
	}
	return exitSuccess
}

func (a *App) printUsage() {
	fmt.Fprintln(a.out, `ocpp-cli is an OCPP 1.6-J charge point command-line client and simulator.

Usage:
  ocpp-cli [global flags] <command> [flags]
  ocpp-cli run --connectors 2 --meter-interval 30s --format jsonl
  ocpp-cli test-connection --profile local
  ocpp-cli status --profile local
  ocpp-cli send heartbeat --profile local
  ocpp-cli boot-notification --profile local
  ocpp-cli heartbeat --profile local
  ocpp-cli authorize --profile local --id-tag ABC123
  ocpp-cli status-notification --connector 1 --status Available
  ocpp-cli meter-values --connector 1 --value 12345 --unit Wh
  ocpp-cli start-transaction --connector 1 --id-tag ABC123 --meter-start 12345
  ocpp-cli stop-transaction --transaction-id 42 --meter-stop 12800
  ocpp-cli data-transfer --vendor-id example.org --message-id Ping --data '{"value":1}'
  ocpp-cli diagnostics-status --status Uploaded
  ocpp-cli firmware-status --status Installed
  ocpp-cli security-event --type InvalidFirmwareSignature
  ocpp-cli log-status --request-id 7 --status Uploaded
  ocpp-cli signed-firmware-status --request-id 8 --status SignatureVerified
  ocpp-cli sign-certificate --csr-file station.csr
  ocpp-cli completions zsh
  ocpp-cli validate-config --profile local
  ocpp-cli init-config

Commands:
  version                    Print version information
  run                        Run a persistent in-memory charge point simulator
  init-config                Write a starter config.yaml
  validate-config            Validate local configuration without connecting
  test-connection            Open and close an OCPP WebSocket connection
  status                     Alias for test-connection
  send                       Send a named OCPP operation
  boot-notification          Send BootNotification
  heartbeat                  Send Heartbeat
  authorize                  Send Authorize
  status-notification        Send StatusNotification
  meter-values               Send one MeterValues sample
  start-transaction          Send StartTransaction
  stop-transaction           Send StopTransaction
  data-transfer              Send vendor-specific JSON data
  diagnostics-status         Send DiagnosticsStatusNotification
  firmware-status            Send FirmwareStatusNotification
  security-event             Send SecurityEventNotification
  log-status                 Send LogStatusNotification
  signed-firmware-status     Send SignedFirmwareStatusNotification
  sign-certificate           Send a PEM certificate signing request
  completions                Generate bash or zsh completion scripts

Global flags:
  --config               YAML config file, defaults to config.yaml
  --profile              Config profile name; uses default_profile when omitted
  --central-system-url   WebSocket base URL; charge_point_id is appended
  --charge-point-id      OCPP charge point identity
  --username             HTTP Basic authentication username
  --password             HTTP Basic authentication password
  --ca-cert              CA certificate PEM file
  --client-cert          Client certificate PEM file
  --client-key           Client private key PEM file
  --tls-server-name      TLS server-name override
  --insecure-skip-verify Skip TLS verification for diagnostics only
  --timeout              Connection and request timeout
  --format               snapshot: table/text/json/csv; stream: text/jsonl/csv
  --verbose              Print high-level connection decisions
  --debug                Enable lower-level protocol logging

CLI flags override values loaded from --config and --profile.
Snapshot commands support table, text, json, and csv. The run stream supports text, jsonl, and csv.`)
}

// send dispatches the canonical namespaced OCPP operations. The historical
// top-level commands remain supported as compatibility aliases.
func (a *App) send(args []string) error {
	normalised, err := normaliseGlobalFlags(args)
	if err != nil {
		return err
	}
	if len(normalised) == 0 || normalised[0] == "help" || normalised[0] == "--help" || normalised[0] == "-h" {
		fmt.Fprintln(a.out, "Usage: ocpp-cli send <operation> [flags]")
		fmt.Fprintln(a.out, "Operations: boot-notification heartbeat authorize status-notification meter-values start-transaction stop-transaction data-transfer diagnostics-status firmware-status security-event log-status signed-firmware-status sign-certificate")
		return nil
	}

	operation, args := normalised[0], normalised[1:]
	switch operation {
	case "boot-notification":
		return a.bootNotification(args)
	case "heartbeat":
		return a.heartbeat(args)
	case "authorize":
		return a.authorize(args)
	case "status-notification":
		return a.statusNotification(args)
	case "meter-values":
		return a.meterValues(args)
	case "start-transaction":
		return a.startTransaction(args)
	case "stop-transaction":
		return a.stopTransaction(args)
	case "data-transfer":
		return a.dataTransfer(args)
	case "diagnostics-status":
		return a.diagnosticsStatus(args)
	case "firmware-status":
		return a.firmwareStatus(args)
	case "security-event":
		return a.securityEvent(args)
	case "log-status":
		return a.logStatus(args)
	case "signed-firmware-status":
		return a.signedFirmwareStatus(args)
	case "sign-certificate":
		return a.signCertificate(args)
	default:
		return fmt.Errorf("unknown OCPP operation %q", operation)
	}
}
