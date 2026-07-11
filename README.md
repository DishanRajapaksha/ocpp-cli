# ocpp-cli

`ocpp-cli` is a small OCPP 1.6-J charge-point command-line client written in Go. It is designed for CSMS integration diagnostics, commissioning scripts, protocol experiments, and reproducible one-shot tests.

Each protocol command opens an OCPP WebSocket connection, performs one charge-point-to-central-system operation, renders the confirmation, and disconnects.

## Features

- OCPP 1.6-J over `ws://` and `wss://`
- YAML configuration with default and named profiles
- HTTP Basic authentication for the WebSocket handshake
- custom CA certificates and mutual TLS
- file-based or base64-encoded certificates
- table, text, JSON, and CSV snapshot output
- stable exit codes shared with the other protocol CLIs
- bash and zsh completion generation
- core charge-point operations:
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
ocpp-cli heartbeat
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

## Commands

### Connection and configuration

```bash
ocpp-cli init-config [--output config.yaml] [--force]
ocpp-cli validate-config [connection flags]
ocpp-cli test-connection [connection flags]
```

### Boot and liveness

```bash
ocpp-cli boot-notification \
  --model ocpp-cli \
  --vendor DishanRajapaksha \
  --firmware-version 0.1.0

ocpp-cli heartbeat
```

### Authorization and connector state

```bash
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

`--data` and `--data-file` are mutually exclusive. To send a JSON string, include the JSON quotes, for example `--data '"hello"'`.

### Diagnostics and firmware status

```bash
ocpp-cli diagnostics-status --status Uploading
ocpp-cli diagnostics-status --status Uploaded

ocpp-cli firmware-status --status Downloading
ocpp-cli firmware-status --status Installing
ocpp-cli firmware-status --status Installed
```

### Security extension

```bash
ocpp-cli security-event \
  --type InvalidFirmwareSignature \
  --tech-info "signature verification failed"

ocpp-cli log-status \
  --request-id 7 \
  --status Uploaded

ocpp-cli signed-firmware-status \
  --request-id 8 \
  --status SignatureVerified

ocpp-cli sign-certificate \
  --csr-file station.csr \
  --certificate-type ChargingStationCertificate
```

These commands use the OCPP 1.6 security extension and require matching CSMS support.

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
| `--format` | `table`, `text`, `json`, or `csv` |
| `--verbose` | connection-level logging |
| `--debug` | lower-level WebSocket and OCPP-J logging |

## Output contract

All current protocol commands produce a snapshot and support:

```text
table, text, json, csv
```

`jsonl` is reserved for future streaming commands and is rejected for snapshots.

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

Rejected `BootNotification`, `Authorize`, transaction, `DataTransfer`, and `SignCertificate` confirmations are rendered before the process exits with code 7. Scripts therefore receive the actual protocol response without treating a rejection as success.

## Shell completions

```bash
ocpp-cli completions bash > ~/.local/share/bash-completion/completions/ocpp-cli
ocpp-cli completions zsh > ~/.zfunc/_ocpp-cli
```

For zsh, ensure `~/.zfunc` is present in `fpath` before `compinit` runs.

## Current boundaries

The CLI is still a one-shot charge-point client. It does not yet provide a persistent station simulator, incoming CSMS command handlers, connector-state persistence, scheduled heartbeats, CSMS-server mode, or raw message streaming. Those features require a long-running state model rather than another stack of one-shot flags.

## Development

```bash
make fmt
make test
make build
```

CI runs dependency resolution, tests with pipe failure propagation, `go vet`, and a production build against the real upstream `lorenzodonini/ocpp-go` dependency.
