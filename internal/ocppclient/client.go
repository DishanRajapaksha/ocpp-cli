package ocppclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ws"
)

type client struct {
	cfg       config.ClientConfig
	cp        ocpp16.ChargePoint
	closeOnce sync.Once
}

func New(cfg config.ClientConfig) (Station, error) {
	if err := config.ValidateClientConfig(cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	configureLogging(cfg.Verbose, cfg.Debug)
	wsClient, err := newWebSocketClient(cfg)
	if err != nil {
		return nil, err
	}
	return &client{cfg: cfg, cp: ocpp16.NewChargePoint(cfg.ChargePointID, nil, wsClient)}, nil
}

func newWebSocketClient(cfg config.ClientConfig) (ws.Client, error) {
	var opts []ws.ClientOpt
	if strings.HasPrefix(strings.ToLower(cfg.CentralSystemURL), "wss://") {
		tlsConfig, err := buildTLSConfig(cfg)
		if err != nil {
			return nil, err
		}
		opts = append(opts, ws.WithClientTLSConfig(tlsConfig))
	}
	client := ws.NewClient(opts...)
	timeouts := ws.NewClientTimeoutConfig()
	timeouts.HandshakeTimeout = cfg.Timeout
	timeouts.WriteWait = cfg.Timeout
	client.SetTimeoutConfig(timeouts)
	if cfg.Username != "" {
		client.SetBasicAuth(cfg.Username, cfg.Password)
	}
	return client, nil
}

func buildTLSConfig(cfg config.ClientConfig) (*tls.Config, error) {
	roots, err := x509.SystemCertPool()
	if err != nil || roots == nil {
		roots = x509.NewCertPool()
	}
	caPEM := cfg.CACertPEM
	if cfg.CACertFile != "" {
		caPEM, err = os.ReadFile(cfg.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("%w: read CA certificate: %v", ErrAuthSecurity, err)
		}
	}
	if len(caPEM) > 0 && !roots.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("%w: CA certificate does not contain valid PEM certificates", ErrAuthSecurity)
	}
	var certificates []tls.Certificate
	if cfg.ClientCertFile != "" {
		certificate, err := tls.LoadX509KeyPair(cfg.ClientCertFile, cfg.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("%w: load client certificate: %v", ErrAuthSecurity, err)
		}
		certificates = append(certificates, certificate)
	} else if len(cfg.ClientCertPEM) > 0 {
		certificate, err := tls.X509KeyPair(cfg.ClientCertPEM, cfg.ClientKeyPEM)
		if err != nil {
			return nil, fmt.Errorf("%w: parse inline client certificate: %v", ErrAuthSecurity, err)
		}
		certificates = append(certificates, certificate)
	}
	return &tls.Config{RootCAs: roots, Certificates: certificates, ServerName: cfg.TLSServerName, InsecureSkipVerify: cfg.InsecureSkipVerify, MinVersion: tls.VersionTLS12}, nil
}

func (c *client) Connect(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("%w: %v", ErrTimeout, ctx.Err())
	default:
	}
	if err := c.cp.Start(strings.TrimRight(c.cfg.CentralSystemURL, "/")); err != nil {
		return classify(err, ErrConnection)
	}
	return nil
}

func (c *client) Close() {
	c.closeOnce.Do(func() {
		if c.cp.IsConnected() {
			c.cp.Stop()
		}
	})
}
