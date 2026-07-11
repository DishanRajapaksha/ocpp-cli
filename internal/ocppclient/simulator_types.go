package ocppclient

import (
	"context"
	"time"
)

// Simulator runs a persistent in-memory OCPP 1.6 charge point.
type Simulator interface {
	Run(context.Context) error
	Events() <-chan SimulatorEvent
}

// SimulatorOptions controls the station's in-memory connector and telemetry model.
type SimulatorOptions struct {
	Connectors        int
	HeartbeatInterval time.Duration
	MeterInterval     time.Duration
	MeterStart        int
	MeterStep         int
}

// SimulatorEvent is one line of the simulator event stream.
type SimulatorEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Direction   string    `json:"direction"`
	Action      string    `json:"action"`
	Status      string    `json:"status,omitempty"`
	ConnectorID *int      `json:"connector_id,omitempty"`
	Detail      string    `json:"detail,omitempty"`
}
