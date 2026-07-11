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

type App struct {
	out        io.Writer
	err        io.Writer
	newStation stationFactory
}

func NewApp(out, err io.Writer) *App {
	return &App{out: out, err: err, newStation: ocppclient.New}
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
	case "init-config":
		err = a.initConfig(args[1:])
	case "validate-config":
		err = a.validateConfig(args[1:])
	case "test-connection":
		err = a.testConnection(args[1:])
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
	fmt.Fprintln(a.out, `ocpp-cli is an OCPP 1.6-J charge point command-line client.

Usage:
  ocpp-cli [global flags] <command> [flags]
  ocpp-cli test-connection --profile local
  ocpp-cli boot-notification --profile local
  ocpp-cli heartbeat --profile local
  ocpp-cli authorize --profile local --id-tag ABC123
  ocpp-cli status-notification --connector 1 --status Available
  ocpp-cli meter-values --connector 1 --value 12345 --unit Wh
  ocpp-cli start-transaction --connector 1 --id-tag ABC123 --meter-start 12345
  ocpp-cli stop-transaction --transaction-id 42 --meter-stop 12800
  ocpp-cli validate-config --profile local
  ocpp-cli init-config

Commands:
  version               Print version information
  init-config           Write a starter config.yaml
  validate-config       Validate local configuration without connecting
  test-connection       Open and close an OCPP WebSocket connection
  boot-notification     Send BootNotification
  heartbeat             Send Heartbeat
  authorize             Send Authorize
  status-notification   Send StatusNotification
  meter-values          Send one MeterValues sample
  start-transaction     Send StartTransaction
  stop-transaction      Send StopTransaction

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
  --format               table, text, json, or csv
  --verbose              Print high-level connection decisions
  --debug                Enable lower-level protocol logging

CLI flags override values loaded from --config and --profile.
All current commands are snapshots; jsonl is reserved for future stream commands.`)
}
