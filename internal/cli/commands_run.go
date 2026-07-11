package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
	"github.com/DishanRajapaksha/ocpp-cli/internal/ocppclient"
	"github.com/DishanRajapaksha/ocpp-cli/internal/output"
)

func (a *App) runSimulator(args []string) error {
	fs := a.flagSet("run")
	var common commonFlags
	addCommonFlags(fs, &common)
	connectors := fs.Int("connectors", 1, "number of simulated connectors")
	heartbeatInterval := fs.Duration("heartbeat-interval", 0, "heartbeat interval; zero uses the BootNotification interval")
	meterInterval := fs.Duration("meter-interval", 0, "meter-value interval; zero disables automatic samples")
	meterStart := fs.Int("meter-start", 0, "initial meter value in Wh")
	meterStep := fs.Int("meter-step", 100, "Wh added to each periodic meter sample")
	duration := fs.Duration("duration", 0, "stop after this duration; zero runs until interrupted")
	model := fs.String("model", "", "charge point model override")
	vendor := fs.String("vendor", "", "charge point vendor override")
	firmware := fs.String("firmware-version", "", "firmware version override")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: run does not accept positional arguments", config.ErrConfig)
	}
	if *connectors <= 0 {
		return fmt.Errorf("%w: connectors must be greater than zero", config.ErrConfig)
	}
	if *heartbeatInterval < 0 || *meterInterval < 0 || *duration < 0 {
		return fmt.Errorf("%w: intervals and duration cannot be negative", config.ErrConfig)
	}
	if *meterStart < 0 || *meterStep < 0 {
		return fmt.Errorf("%w: meter-start and meter-step cannot be negative", config.ErrConfig)
	}

	cfg, err := resolveClientConfig(fs, common)
	if err != nil {
		return err
	}
	if wasSet(fs, "model") {
		cfg.ChargePointModel = *model
	}
	if wasSet(fs, "vendor") {
		cfg.ChargePointVendor = *vendor
	}
	if wasSet(fs, "firmware-version") {
		cfg.FirmwareVersion = *firmware
	}
	if err := config.ValidateClientConfig(cfg); err != nil {
		return err
	}

	formatValue := cfg.Format
	if !wasSet(fs, "format") {
		switch output.NormaliseFormat(formatValue) {
		case output.FormatText, output.FormatJSONL, output.FormatCSV:
		default:
			formatValue = output.FormatText
		}
	}
	format, err := validateStreamFormat(formatValue)
	if err != nil {
		return fmt.Errorf("%w: %v", config.ErrConfig, err)
	}

	simulator, err := a.newSimulator(cfg, ocppclient.SimulatorOptions{
		Connectors:        *connectors,
		HeartbeatInterval: *heartbeatInterval,
		MeterInterval:     *meterInterval,
		MeterStart:        *meterStart,
		MeterStep:         *meterStep,
	})
	if err != nil {
		return err
	}

	ctx := context.Background()
	var cancel context.CancelFunc = func() {}
	if *duration > 0 {
		ctx, cancel = context.WithTimeout(ctx, *duration)
	}
	defer cancel()
	ctx, stopSignals := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	writer := newSimulatorStreamWriter(a.out, format)
	runResult := make(chan error, 1)
	go func() { runResult <- simulator.Run(ctx) }()

	for {
		select {
		case event, ok := <-simulator.Events():
			if !ok {
				err := <-runResult
				return err
			}
			if err := writer.Write(event); err != nil {
				stopSignals()
				return err
			}
		case err := <-runResult:
			for event := range simulator.Events() {
				if writeErr := writer.Write(event); writeErr != nil {
					return writeErr
				}
			}
			return err
		}
	}
}
