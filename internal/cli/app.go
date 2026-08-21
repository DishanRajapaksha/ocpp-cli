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
		a.writeRegistryUsageTo(a.err)
		fmt.Fprintf(a.err, "unknown command %q\n", args[0])
		return exitConfigError
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
	a.writeRegistryUsage()
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
