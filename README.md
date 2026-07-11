# ocpp-cli

`ocpp-cli` is an OCPP 1.6-J charge-point command-line client and simulator written in Go. It is designed for CSMS integration diagnostics, commissioning scripts, protocol experiments, and reproducible station tests.

Most protocol commands are one-shot operations: they connect, send one charge-point-to-CSMS request, render the confirmation, and disconnect. The `run` command keeps a simulated station online and responds to incoming CSMS commands.

## Features

- OCPP 1.6-J over `ws://` and `wss://`
- YAML configuration with default and named profiles
- HTTP Basic authentication, custom CAs, and mutual TLS
- file-based or base64-encoded certificates
- table, text, JSON, and CSV snapshot output
- text, JSONL, and CSV stream output
- stable exit codes shared with the other protocol CLIs
- bash and zsh completion generation
- persistent in-memory charge-point simulator with:
  - automatic `BootNotification`
  - connector `StatusNotification` messages
  - periodic heartbeats and optional meter values
  - remote start and stop transaction handling
  - availability, configuration, cache, reset, unlock, and data-transfer handlers
- one-shot core operations:
  - `BootNotification`
  - `Heartbeat`
  - `Authorize`
  - `StatusNotification`
  - `MeterValues`
  - `StartTransaction`
  - `StopTransaction`
  - `DataTransfer`
  - `DiagnosticsStatusNotification`
  - `FirmwareStatusNotification`
- OCPP 1.6 security-extension operations:
  - `SecurityEventNotification`
  - `LogStatusNotification`
  - `SignedFirmwareStatusNotification`
  - `SignCertificate`

## Install

```bash
go install github.com/DishanRajapaksha/ocpp-cli@latest
```

Or build locally:

```bash
make build
./bin/ocpp-cli version
```

## Quick start

```bash
ocpp-cli init-config
ocpp-cli validate-config
ocpp-cli test-connection
ocpp-cli boot-notification
```

Run a persistent station with two connectors:

```bash
ocpp-cli run \
  --connectors 2 \
  --heartbeat-interval 60s \
  --meter-interval 30s \
  --meter-start 12000 \
  --meter-step 100
```

Use JSONL for automation:

```bash
ocpp-cli run --connectors 2 --meter-interval 10s --format jsonl
```

## Configuration

The default configuration path is `config.yaml`. Use `--config` to select another file and `--profile` to choose a named profile. Command-line flags override configuration values.

```yaml
default_profile: local

profiles:
  local:
    central_system_url: ws://localhost:8080/ocpp
    charge_point_id: CP001
    timeout: 10s
    format: table

    username: ""
    password: ""

    ca_cert: ""
    client_cert: ""
    client_key: ""
    tls_server_name: ""
    insecure_skip_verify: false

    charge_point_model: ocpp-cli
    charge_point_vendor: DishanRajapaksha
    firmware_version: ""
    charge_point_serial_number: ""
    meter_serial_number: ""
    meter_type: ""
```

The OCPP library appends `charge_point_id` to `central_system_url`. The example above connects to:

```text
ws://localhost:8080/ocpp/CP001
```

PEM certificate and key bytes may also be base64-encoded using `ca_cert_base64`, `client_cert_base64`, and `client_key_base64`. Do not commit credential-bearing configuration files.

## Persistent station simulator

```bash
ocpp-cli run [connection flags] [simulator flags]
```

| Flag | Default | Purpose |
|---|---:|---|
| `--connectors` | `1` | number of simulated connectors |
| `--heartbeat-interval` | `0` | heartbeat cadence; zero uses the interval returned by `BootNotification` |
| `--meter-interval` | `0` | periodic meter cadence; zero disables automatic samples |
| `--meter-start` | `0` | initial energy register value in Wh |
| `--meter-step` | `100` | Wh added to each periodic sample |
| `--duration` | `0` | stop after a duration; zero runs until interrupted |
| `--model` | config value | charge-point model override |
| `--vendor` | config value | charge-point vendor override |
| `--firmware-version` | config value | firmware version override |

The simulator boots once, publishes status for the station and each connector, then remains connected. It maintains connector availability, status, meter values, and transaction IDs in memory.

Supported incoming Core profile operations:

```text
ChangeAvailability
ChangeConfiguration
ClearCache
DataTransfer
GetConfiguration
RemoteStartTransaction
RemoteStopTransaction
Reset
UnlockConnector
```

Remote-started transactions produce `StartTransaction` and connector status messages. Remote stops produce `StopTransaction` and return the connector to `Available` or `Unavailable` according to its configured availability.

The simulator's stream formats are:

```text
text, jsonl, csv
```

CSV writes its header once. `table` and `json` are rejected for `run` because they are snapshot formats.

## One-shot commands

### Connection and configuration

```bash
ocpp-cli init-config [--output config.yaml] [--force]
ocpp-cli validate-config [connection flags]
ocpp-cli test-connection [connection flags]
```

### Boot, authorization, and status

```bash
ocpp-cli boot-notification \
  --model ocpp-cli \
  --vendor DishanRajapaksha \
  --firmware-version 0.1.0

ocpp-cli heartbeat
ocpp-cli authorize --id-tag ABC123

ocpp-cli status-notification \
  --connector 1 \
  --status Available \
  --error-code NoError
```

### Metering and transactions

```bash
ocpp-cli meter-values \
  --connector 1 \
  --value 12345 \
  --measurand Energy.Active.Import.Register \
  --unit Wh \
  --context Sample.Periodic \
  --location Outlet

ocpp-cli start-transaction \
  --connector 1 \
  --id-tag ABC123 \
  --meter-start 12345

ocpp-cli stop-transaction \
  --transaction-id 42 \
  --meter-stop 12800 \
  --reason Local
```

Timestamps default to the current UTC time. Pass `--timestamp` with an RFC 3339 value for scripted or historical scenarios.

### Vendor data transfer

Inline payloads must be valid JSON:

```bash
ocpp-cli data-transfer \
  --vendor-id example.org \
  --message-id SetMode \
  --data '{"mode":"eco"}'
```

A JSON file may be used instead:

```bash
ocpp-cli data-transfer \
  --vendor-id example.org \
  --message-id SetMode \
  --data-file payload.json
```

`--data` and `--data-file` are mutually exclusive.

### Diagnostics, firmware, and security extension

```bash
ocpp-cli diagnostics-status --status Uploaded
ocpp-cli firmware-status --status Installed

ocpp-cli security-event \
  --type InvalidFirmwareSignature \
  --tech-info "signature verification failed"

ocpp-cli log-status --request-id 7 --status Uploaded
ocpp-cli signed-firmware-status --request-id 8 --status SignatureVerified
ocpp-cli sign-certificate --csr-file station.csr
```

Security commands require matching OCPP 1.6 security-extension support in the CSMS.

## Global flags

Global flags may appear before or after the command:

```bash
ocpp-cli --profile lab --format json heartbeat
ocpp-cli heartbeat --profile lab --format json
```

| Flag | Purpose |
|---|---|
| `--config` | YAML configuration path, default `config.yaml` |
| `--profile` | named configuration profile |
| `--central-system-url` | CSMS WebSocket base URL |
| `--charge-point-id` | charge-point identity |
| `--username`, `--password` | HTTP Basic authentication |
| `--ca-cert` | custom CA PEM file |
| `--client-cert`, `--client-key` | mutual-TLS identity |
| `--tls-server-name` | TLS hostname override |
| `--insecure-skip-verify` | disable TLS verification for diagnostics only |
| `--timeout` | connection and request timeout |
| `--format` | snapshot: `table`, `text`, `json`, `csv`; stream: `text`, `jsonl`, `csv` |
| `--verbose` | connection-level logging |
| `--debug` | lower-level WebSocket and OCPP-J logging |

## Output contract

Snapshot commands support:

```text
table, text, json, csv
```

Streaming commands support:

```text
text, jsonl, csv
```

The CLI rejects stream-only formats on snapshots and snapshot-only formats on streams.

## Exit codes

| Code | Meaning |
|---:|---|
| 0 | success |
| 1 | general error |
| 2 | usage or configuration error |
| 3 | transport or connection error |
| 4 | OCPP protocol or request error |
| 5 | authentication or TLS security error |
| 6 | resource not found, reserved |
| 7 | request or authorization rejected |
| 8 | operation timeout |
| 9 | output or formatting error |

Rejected `BootNotification`, `Authorize`, transaction, `DataTransfer`, and `SignCertificate` confirmations are rendered before exit code 7.

## Shell completions

```bash
ocpp-cli completions bash > ~/.local/share/bash-completion/completions/ocpp-cli
ocpp-cli completions zsh > ~/.zfunc/_ocpp-cli
```

For zsh, ensure `~/.zfunc` is present in `fpath` before `compinit` runs.

## Current boundaries

The simulator is intentionally in-memory. It does not persist connector state, emulate physical charging limits, download firmware, upload diagnostics, or implement every optional OCPP profile. It currently handles the incoming Core profile while the existing one-shot commands cover charge-point-originated firmware and security-extension messages.

## Development

```bash
make fmt
make test
make build
```

CI runs dependency resolution, tests with pipe failure propagation, `go vet`, and a production build against the real upstream `lorenzodonini/ocpp-go` dependency.
