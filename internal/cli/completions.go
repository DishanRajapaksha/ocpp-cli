package cli

import (
	"errors"
	"flag"
	"fmt"
	"strings"
)

func (a *App) completions(args []string) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(a.err, "Usage of completions:")
		fmt.Fprintln(a.err, "  ocpp-cli completions bash|zsh")
		return flag.ErrHelp
	}
	if len(args) != 1 {
		return errors.New("usage: ocpp-cli completions <bash|zsh>")
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "bash":
		_, err := fmt.Fprint(a.out, bashCompletionScript)
		return err
	case "zsh":
		_, err := fmt.Fprint(a.out, zshCompletionScript)
		return err
	default:
		return errors.New("unsupported shell, expected bash or zsh")
	}
}

const completionSubcommands = "boot-notification heartbeat authorize status-notification meter-values start-transaction stop-transaction data-transfer diagnostics-status firmware-status security-event log-status signed-firmware-status sign-certificate test-connection validate-config init-config completions help"
const completionCommonFlags = "--config --profile --central-system-url --charge-point-id --username --password --ca-cert --client-cert --client-key --tls-server-name --insecure-skip-verify --timeout --format --verbose --debug"

const bashCompletionScript = `#!/usr/bin/env bash
_ocpp_cli_completions() {
  local cur prev words cword
  _init_completion || return

  local subcommands="` + completionSubcommands + `"
  local common_flags="` + completionCommonFlags + `"

  if [[ ${cword} -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "${subcommands}" -- "${cur}") )
    return
  fi

  case "${words[1]}" in
    boot-notification)
      COMPREPLY=( $(compgen -W "${common_flags} --model --vendor --firmware-version --serial-number --meter-serial-number --meter-type" -- "${cur}") )
      ;;
    authorize)
      COMPREPLY=( $(compgen -W "${common_flags} --id-tag" -- "${cur}") )
      ;;
    status-notification)
      COMPREPLY=( $(compgen -W "${common_flags} --connector --status --error-code --info --vendor-id --vendor-error-code --timestamp" -- "${cur}") )
      ;;
    meter-values)
      COMPREPLY=( $(compgen -W "${common_flags} --connector --transaction-id --value --measurand --unit --context --location --phase --timestamp" -- "${cur}") )
      ;;
    start-transaction)
      COMPREPLY=( $(compgen -W "${common_flags} --connector --id-tag --meter-start --reservation-id --timestamp" -- "${cur}") )
      ;;
    stop-transaction)
      COMPREPLY=( $(compgen -W "${common_flags} --transaction-id --meter-stop --id-tag --reason --timestamp" -- "${cur}") )
      ;;
    data-transfer)
      COMPREPLY=( $(compgen -W "${common_flags} --vendor-id --message-id --data --data-file" -- "${cur}") )
      ;;
    diagnostics-status|firmware-status)
      COMPREPLY=( $(compgen -W "${common_flags} --status" -- "${cur}") )
      ;;
    security-event)
      COMPREPLY=( $(compgen -W "${common_flags} --type --tech-info --timestamp" -- "${cur}") )
      ;;
    log-status|signed-firmware-status)
      COMPREPLY=( $(compgen -W "${common_flags} --status --request-id" -- "${cur}") )
      ;;
    sign-certificate)
      COMPREPLY=( $(compgen -W "${common_flags} --csr-file --certificate-type" -- "${cur}") )
      ;;
    heartbeat|test-connection|validate-config)
      COMPREPLY=( $(compgen -W "${common_flags}" -- "${cur}") )
      ;;
    init-config)
      COMPREPLY=( $(compgen -W "--output --force" -- "${cur}") )
      ;;
    completions)
      COMPREPLY=( $(compgen -W "bash zsh" -- "${cur}") )
      ;;
  esac
}
complete -F _ocpp_cli_completions ocpp-cli
`

const zshCompletionScript = `#compdef ocpp-cli
_ocpp_cli_completions() {
  local -a subcommands
  subcommands=(
    'boot-notification:Send BootNotification'
    'heartbeat:Send Heartbeat'
    'authorize:Send Authorize'
    'status-notification:Send StatusNotification'
    'meter-values:Send MeterValues'
    'start-transaction:Send StartTransaction'
    'stop-transaction:Send StopTransaction'
    'data-transfer:Send vendor-specific JSON data'
    'diagnostics-status:Send diagnostics upload status'
    'firmware-status:Send firmware update status'
    'security-event:Send security event notification'
    'log-status:Send log upload status'
    'signed-firmware-status:Send signed firmware status'
    'sign-certificate:Submit a certificate signing request'
    'test-connection:Run connection diagnostics'
    'validate-config:Validate local config'
    'init-config:Write starter YAML config'
    'completions:Generate shell completion scripts'
  )

  local -a common_flags
  common_flags=(
    '--config[YAML config file]:config file:_files'
    '--profile[Config profile name]:profile:'
    '--central-system-url[WebSocket base URL]:url:'
    '--charge-point-id[Charge point identity]:id:'
    '--username[HTTP Basic username]:username:'
    '--password[HTTP Basic password]:password:'
    '--ca-cert[CA certificate file]:cert:_files'
    '--client-cert[Client certificate file]:cert:_files'
    '--client-key[Client private key file]:key:_files'
    '--tls-server-name[TLS server-name override]:name:'
    '--insecure-skip-verify[Skip TLS certificate verification]'
    '--timeout[Connection and request timeout]:duration:'
    '--format[Output format]:format:(table text json csv)'
    '--verbose[Print high-level connection decisions]'
    '--debug[Enable lower-level protocol logging]'
  )

  if (( CURRENT == 2 )); then
    _describe 'subcommand' subcommands
    return
  fi

  case $words[2] in
    boot-notification)
      _arguments $common_flags '--model[charge point model]:model:' '--vendor[charge point vendor]:vendor:' '--firmware-version[firmware version]:version:' '--serial-number[charge point serial]:serial:' '--meter-serial-number[meter serial]:serial:' '--meter-type[meter type]:type:'
      ;;
    authorize)
      _arguments $common_flags '--id-tag[identifier]:id tag:'
      ;;
    status-notification)
      _arguments $common_flags '--connector[connector ID]:id:' '--status[connector status]:status:(Available Preparing Charging SuspendedEVSE SuspendedEV Finishing Reserved Unavailable Faulted)' '--error-code[error code]:error:' '--info[additional information]:info:' '--vendor-id[vendor ID]:vendor:' '--vendor-error-code[vendor error code]:code:' '--timestamp[RFC3339 timestamp or now]:timestamp:'
      ;;
    meter-values)
      _arguments $common_flags '--connector[connector ID]:id:' '--transaction-id[transaction ID]:id:' '--value[sample value]:value:' '--measurand[OCPP measurand]:measurand:' '--unit[OCPP unit]:unit:' '--context[reading context]:context:' '--location[measurement location]:location:' '--phase[electrical phase]:phase:' '--timestamp[RFC3339 timestamp or now]:timestamp:'
      ;;
    start-transaction)
      _arguments $common_flags '--connector[connector ID]:id:' '--id-tag[identifier]:id tag:' '--meter-start[start meter value]:value:' '--reservation-id[reservation ID]:id:' '--timestamp[RFC3339 timestamp or now]:timestamp:'
      ;;
    stop-transaction)
      _arguments $common_flags '--transaction-id[transaction ID]:id:' '--meter-stop[final meter value]:value:' '--id-tag[identifier]:id tag:' '--reason[stop reason]:reason:' '--timestamp[RFC3339 timestamp or now]:timestamp:'
      ;;
    data-transfer)
      _arguments $common_flags '--vendor-id[vendor identifier]:vendor:' '--message-id[message identifier]:message:' '--data[inline JSON payload]:json:' '--data-file[JSON payload file]:file:_files'
      ;;
    diagnostics-status)
      _arguments $common_flags '--status[diagnostics status]:status:(Idle Uploaded UploadFailed Uploading)'
      ;;
    firmware-status)
      _arguments $common_flags '--status[firmware status]:status:(Downloaded DownloadFailed Downloading Idle InstallationFailed Installing Installed)'
      ;;
    security-event)
      _arguments $common_flags '--type[security event type]:type:' '--tech-info[technical information]:info:' '--timestamp[RFC3339 timestamp or now]:timestamp:'
      ;;
    log-status)
      _arguments $common_flags '--status[log upload status]:status:(BadMessage Idle NotSupportedOperation PermissionDenied Uploaded UploadFailure Uploading)' '--request-id[GetLog request ID]:id:'
      ;;
    signed-firmware-status)
      _arguments $common_flags '--status[signed firmware status]:status:' '--request-id[optional request ID]:id:'
      ;;
    sign-certificate)
      _arguments $common_flags '--csr-file[PEM CSR file]:file:_files' '--certificate-type[certificate signing use]:type:(ChargingStationCertificate)'
      ;;
    heartbeat|test-connection|validate-config)
      _arguments $common_flags
      ;;
    init-config)
      _arguments '--output[output YAML config file]:file:_files' '--force[overwrite output file]'
      ;;
    completions)
      _arguments '1:shell:(bash zsh)'
      ;;
  esac
}
_ocpp_cli_completions "$@"
`
