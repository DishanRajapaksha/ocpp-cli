package ocppclient

import "errors"

var (
	ErrValidation   = errors.New("ocpp validation error")
	ErrConnection   = errors.New("ocpp connection error")
	ErrProtocol     = errors.New("ocpp protocol error")
	ErrAuthSecurity = errors.New("ocpp authentication or security error")
	ErrRejected     = errors.New("ocpp request rejected")
	ErrTimeout      = errors.New("ocpp operation timeout")
)
