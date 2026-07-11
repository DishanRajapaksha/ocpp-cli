# Security-extension simulator profiles

The persistent `ocpp-cli run` simulator implements the OCPP 1.6-J security-extension profiles supported by the pinned `ocpp-go` dependency.

## Security and certificate signing

Supported incoming operations:

```text
CertificateSigned
```

The simulator parses the first PEM-encoded X.509 certificate from the supplied chain and retains the complete chain in memory. Invalid PEM or malformed certificates are rejected. The chain is not written to disk and is not added to the process TLS configuration.

## Certificate management

Supported incoming operations:

```text
InstallCertificate
GetInstalledCertificateIds
DeleteCertificate
```

Installed `CentralSystemRootCertificate` and `ManufacturerRootCertificate` entries are retained in memory. Certificate identifiers use SHA-256 hashes derived from the parsed certificate issuer and public-key material. This inventory is intended for protocol testing, not as a production trust store.

## Log retrieval

Supported incoming operation:

```text
GetLog
```

The simulator returns a generated filename and emits:

```text
Uploading
Uploaded
```

A new request received while another log request is active returns `AcceptedCanceled`. No file is created or uploaded.

## Signed firmware

Supported incoming operation:

```text
SignedUpdateFirmware
```

Requests without a signing certificate or signature return `InvalidCertificate`. Accepted requests simulate:

```text
DownloadScheduled
Downloading
Downloaded
SignatureVerified
Installing
Installed
```

The simulator honours `retrieveDateTime` and `installDateTime`. It does not download, verify, install, or execute firmware.

## Extended triggers

Supported incoming operation:

```text
ExtendedTriggerMessage
```

The simulator can trigger:

```text
BootNotification
Heartbeat
MeterValues
StatusNotification
FirmwareStatusNotification
LogStatusNotification
SignChargePointCertificate
```

`SignChargePointCertificate` generates an ephemeral ECDSA P-256 key and valid PEM certificate-signing request, then sends `SignCertificate`. The private key is discarded immediately after the CSR is created.

## State and output

All state is process-local and disappears when `ocpp-cli run` exits. Incoming decisions and generated outbound messages use the normal simulator stream contract:

```text
text, jsonl, csv
```
