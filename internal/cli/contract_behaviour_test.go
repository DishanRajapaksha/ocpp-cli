package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/DishanRajapaksha/industrial-cli-kit/contracttest"
	"github.com/DishanRajapaksha/industrial-cli-kit/exitcode"
	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
	"github.com/DishanRajapaksha/ocpp-cli/internal/ocppclient"
)

func writeContractConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, config.StarterConfigYAML(), 0o600); err != nil {
		t.Fatalf("write contract config: %v", err)
	}
	return path
}

func TestSharedGlobalFlagContract(t *testing.T) {
	configPath := writeContractConfig(t)
	run := func(args ...string) contracttest.Result {
		var out, errOut bytes.Buffer
		app := NewApp(&out, &errOut)
		code := app.Run(append([]string{"--config", configPath}, args...))
		return contracttest.Result{Code: code, Stdout: out.String(), Stderr: errOut.String()}
	}
	contracttest.GlobalFlagPositioning(t, run, contracttest.GlobalFlagOptions{
		Success: []string{"validate-config"},
		Probes:  []contracttest.GlobalFlagProbe{{Name: "config", Value: configPath}, {Name: "verbose"}},
	})
	contracttest.RejectsUnsupportedGlobalFlag(t, run)
}

func TestSharedSnapshotFormatContract(t *testing.T) {
	configPath := writeContractConfig(t)
	run := func(args ...string) contracttest.Result {
		var out, errOut bytes.Buffer
		app := NewApp(&out, &errOut)
		code := app.Run(append([]string{"--config", configPath}, args...))
		return contracttest.Result{Code: code, Stdout: out.String(), Stderr: errOut.String()}
	}
	contracttest.Formats(t, run, contracttest.FormatOptions{
		Command:      []string{"validate-config"},
		Kind:         contracttest.SnapshotFormats,
		AcceptedCode: int(exitcode.Success),
	})
}

func TestSharedOutputSeparation(t *testing.T) {
	run := func(args ...string) contracttest.Result {
		var out, errOut bytes.Buffer
		app := NewApp(&out, &errOut)
		code := app.Run(args)
		return contracttest.Result{Code: code, Stdout: out.String(), Stderr: errOut.String()}
	}
	contracttest.OutputSeparation(t, run, []string{"completions", "bash"})
	contracttest.ErrorSeparation(t, run, []string{"definitely-not-a-command"}, exitcode.Config)
}

func TestSharedUsageExitCodeContract(t *testing.T) {
	run := func(args ...string) contracttest.Result {
		var out, errOut bytes.Buffer
		app := NewApp(&out, &errOut)
		code := app.Run(args)
		return contracttest.Result{Code: code, Stdout: out.String(), Stderr: errOut.String()}
	}
	contracttest.ExitCodes(t, run, contracttest.UsageExitCodeScenarios())
}

func TestSharedMutatingSafetyContract(t *testing.T) {
	configPath := writeContractConfig(t)
	connected := false
	newStation := func(config.ClientConfig) (ocppclient.Station, error) {
		connected = true
		return &fakeStation{}, nil
	}
	run := func(args ...string) contracttest.Result {
		connected = false
		var out, errOut bytes.Buffer
		app := NewApp(&out, &errOut)
		app.newStation = newStation
		code := app.Run(append([]string{"--config", configPath}, args...))
		return contracttest.Result{Code: code, Stdout: out.String(), Stderr: errOut.String()}
	}
	contracttest.MutatingSafety(t, run, contracttest.MutatingSafetyOptions{
		Args:                []string{"send", "start-transaction", "--connector", "1", "--id-tag", "ABC123", "--meter-start", "100"},
		Executed:            func() bool { return connected },
		ExecutedWithYesCode: int(exitcode.Success),
		ExecutedWithYes:     true,
	})
}
