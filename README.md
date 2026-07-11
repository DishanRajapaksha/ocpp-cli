# ocpp-cli

`ocpp-cli` is a small OCPP 1.6-J charge-point command-line client written in Go. It is intended for integration diagnostics, CSMS testing, commissioning scripts, and reproducible protocol experiments.

The initial release deliberately implements the charge-point-to-central-system direction. Each command opens an OCPP WebSocket connection, performs one operation, prints the confirmation, and disconnects.

## Features

- OCPP 1.6-J over `ws://` and `wss://`
- YAML configuration with named profiles
- HTTP Basic authentication for the WebSocket handshake
- custom CA certificates and mutual TLS
- file-based or base64-encoded certificates in configuration
- table, text, JSON, and CSV snapshot output
- stable exit codes shared with the other protocol CLIs
- core charge-point operations:
  - `BootNotification`
  - `Heartbeat`
  - `Authorize`
  - `StatusNotification`
  - `MeterValues`
  - `StartTransaction`
  - `StopTransaction`

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

Create a starter configuration:

```bash
ocpp-cli init-config
```

Edit `config.yaml`, then validate it without opening a network connection:

```bash
ocpp-cli validate-config
```

Test the WebSocket and OCPP subprotocol handshake:

```bash
ocpp-cli test-connection
```

Send the station boot notification:

```bash
ocpp-cli boot-notification
```

## Configuration

By default, the CLI reads `config.yaml` from the current directory. Use `--config` to select another file and `--profile` to select a named profile. CLI flags override file values.

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

The library appends `charge_point_id` to `central_system_url`. With the example above, the WebSocket URL is:

```text
ws://localhost:8080/ocpp/CP001
```

For a single self-contained secret file, PEM certificate and key bytes may be base64-encoded using:

```yaml
ca_cert_base64: "..."
client_cert_base64: "..."
client_key_base64: "..."
```

Do not commit credential-bearing configuration files.

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

### Metering

```bash
ocpp-cli meter-values \
  --connector 1 \
  --value 12345 \
  --measurand Energy.Active.Import.Register \
  --unit Wh \
  --context Sample.Periodic \
  --location Outlet
```

Attach a sample to an active transaction with `--transaction-id`.

### Transactions

```bash
ocpp-cli start-transaction \
  --connector 1 \
  --id-tag ABC123 \
  --meter-start 12345

ocpp-cli stop-transaction \
  --transaction-id 42 \
  --meter-stop 12800 \
  --reason Local
```

Timestamps default to the current UTC time. Pass `--timestamp` with an RFC 3339 value to reproduce historical or scripted scenarios.

## Global flags

Global flags may appear before or after the command:

```bash
ocpp-cli --profile lab --format json heartbeat
ocpp-cli heartbeat --profile lab --format json
```

Important flags:

| Flag | Purpose |
|---|---|
| `--config` | YAML configuration path, default `config.yaml` |
| `--profile` | named configuration profile |
| `--central-system-url` | CSMS WebSocket base URL |
| `--charge-point-id` | charge point identity |
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

All current commands produce one snapshot and support:

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

A rejected `BootNotification`, `Authorize`, or transaction authorization is still printed before the process exits with code 7. This makes the response available to scripts without disguising the rejection as success.

## Current boundaries

This first version is not a persistent station simulator or CSMS server. It does not yet handle incoming remote commands, maintain connector state between invocations, schedule recurring heartbeats, or stream raw message traffic. Those features require a long-running process and a state model rather than another pile of one-shot flags.

## Development

```bash
make fmt
make test
make build
```

The implementation wraps `github.com/lorenzodonini/ocpp-go` behind an internal interface so CLI and output contracts can be tested without a live charging backend.
