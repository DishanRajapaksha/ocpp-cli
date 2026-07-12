package cli

import sharedhelp "github.com/DishanRajapaksha/industrial-cli-kit/help"

func (a *App) writeRegistryUsage() {
	_ = sharedhelp.Write(a.out, cliRegistry, sharedhelp.Options{
		Description: "ocpp-cli is an OCPP 1.6-J charge point command-line client and simulator.",
		Usage: []string{
			"ocpp-cli [global flags] <command> [flags]",
		},
		Examples: []string{
			"ocpp-cli run --connectors 2 --meter-interval 30s --format jsonl",
			"ocpp-cli test-connection --profile local",
			"ocpp-cli status --profile local",
			"ocpp-cli send heartbeat --profile local",
			"ocpp-cli send authorize --id-tag ABC123",
			"ocpp-cli send status-notification --connector 1 --status Available",
			"ocpp-cli send start-transaction --connector 1 --id-tag ABC123 --meter-start 12345 --yes",
			"ocpp-cli send stop-transaction --transaction-id 42 --meter-stop 12800 --yes",
			"ocpp-cli completions zsh",
		},
	})
}
