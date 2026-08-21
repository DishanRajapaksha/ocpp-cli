package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
)

func (a *App) flagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.err)
	return fs
}

func (a *App) initConfig(args []string) error {
	fs := a.flagSet("init-config")
	outputPath := fs.String("output", config.DefaultConfigPath, "output file")
	force := fs.Bool("force", false, "overwrite an existing file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: init-config does not accept positional arguments", config.ErrConfig)
	}
	flags := os.O_WRONLY | os.O_CREATE
	if *force {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	file, err := os.OpenFile(*outputPath, flags, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%w: %s already exists; use --force to overwrite it", config.ErrConfig, *outputPath)
		}
		return fmt.Errorf("%w: create %s: %v", config.ErrConfig, *outputPath, err)
	}
	defer file.Close()
	if _, err := file.Write(config.StarterConfigYAML()); err != nil {
		return fmt.Errorf("write %s: %w", *outputPath, err)
	}
	fmt.Fprintf(a.out, "wrote %s\n", *outputPath)
	return nil
}

func (a *App) validateConfig(args []string) error {
	fs := a.flagSet("validate-config")
	var common commonFlags
	addCommonFlags(fs, &common)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: validate-config does not accept positional arguments", config.ErrConfig)
	}
	cfg, format, err := resolveCommon(fs, common)
	if err != nil {
		return err
	}
	result := struct {
		Valid            bool   `json:"valid"`
		Protocol         string `json:"protocol"`
		CentralSystemURL string `json:"central_system_url"`
		ChargePointID    string `json:"charge_point_id"`
		Format           string `json:"format"`
	}{true, "ocpp1.6", cfg.CentralSystemURL, cfg.ChargePointID, format}
	return renderSnapshot(a.out, format, keyValueSnapshot(result,
		[2]string{"VALID", "true"}, [2]string{"PROTOCOL", "ocpp1.6"},
		[2]string{"CENTRAL_SYSTEM_URL", cfg.CentralSystemURL}, [2]string{"CHARGE_POINT_ID", cfg.ChargePointID}, [2]string{"FORMAT", format},
	))
}

func (a *App) testConnection(args []string) error {
	fs := a.flagSet("test-connection")
	var common commonFlags
	addCommonFlags(fs, &common)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: test-connection does not accept positional arguments", config.ErrConfig)
	}
	cfg, format, err := resolveCommon(fs, common)
	if err != nil {
		return err
	}
	station, _, cancel, err := a.connect(cfg)
	if err != nil {
		return err
	}
	cancel()
	station.Close()
	result := struct {
		Connected        bool   `json:"connected"`
		Protocol         string `json:"protocol"`
		CentralSystemURL string `json:"central_system_url"`
		ChargePointID    string `json:"charge_point_id"`
	}{true, "ocpp1.6", cfg.CentralSystemURL, cfg.ChargePointID}
	return renderSnapshot(a.out, format, keyValueSnapshot(result,
		[2]string{"CONNECTED", "true"}, [2]string{"PROTOCOL", "ocpp1.6"},
		[2]string{"CENTRAL_SYSTEM_URL", cfg.CentralSystemURL}, [2]string{"CHARGE_POINT_ID", cfg.ChargePointID},
	))
}
