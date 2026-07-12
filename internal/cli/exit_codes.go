package cli

import (
	"context"
	"errors"
	"flag"
	"strings"

	"github.com/DishanRajapaksha/industrial-cli-kit/exitcode"
	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
	"github.com/DishanRajapaksha/ocpp-cli/internal/ocppclient"
	"github.com/DishanRajapaksha/ocpp-cli/internal/output"
)

const (
	exitSuccess         = int(exitcode.Success)
	exitGeneralError    = int(exitcode.General)
	exitConfigError     = int(exitcode.Config)
	exitConnection      = int(exitcode.Connection)
	exitProtocolRequest = int(exitcode.Request)
	exitAuthSecurity    = int(exitcode.AuthSecurity)
	exitResourceMissing = int(exitcode.ResourceMissing)
	exitRejected        = int(exitcode.Rejected)
	exitTimeout         = int(exitcode.Timeout)
	exitOutputError     = int(exitcode.Output)
)

func mapExitCode(err error) int {
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
		return exitSuccess
	case isFlagParseError(err), errors.Is(err, config.ErrConfig), errors.Is(err, ocppclient.ErrValidation):
		return exitConfigError
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, ocppclient.ErrTimeout):
		return exitTimeout
	case errors.Is(err, output.ErrOutput):
		return exitOutputError
	case errors.Is(err, ocppclient.ErrAuthSecurity):
		return exitAuthSecurity
	case errors.Is(err, ocppclient.ErrRejected):
		return exitRejected
	case errors.Is(err, ocppclient.ErrProtocol):
		return exitProtocolRequest
	case errors.Is(err, ocppclient.ErrConnection):
		return exitConnection
	default:
		return exitGeneralError
	}
}

func isFlagParseError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "flag provided but not defined") ||
		strings.Contains(msg, "flag needs an argument") ||
		strings.Contains(msg, "invalid value") ||
		strings.Contains(msg, "is required")
}
