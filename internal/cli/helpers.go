package cli

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
	"github.com/DishanRajapaksha/ocpp-cli/internal/ocppclient"
)

func (a *App) connect(cfg config.ClientConfig) (ocppclient.Station, context.Context, context.CancelFunc, error) {
	station, err := a.newStation(cfg)
	if err != nil { return nil, nil, nil, err }
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	if cfg.Verbose { fmt.Fprintf(a.err, "connecting to %s/%s using ocpp1.6\n", strings.TrimRight(cfg.CentralSystemURL, "/"), cfg.ChargePointID) }
	if err := station.Connect(ctx); err != nil {
		cancel()
		station.Close()
		return nil, nil, nil, err
	}
	return station, ctx, cancel, nil
}

func wasSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) { if f.Name == name { set = true } })
	return set
}

func parseTimestamp(value string) (time.Time, error) {
	if value == "" || strings.EqualFold(value, "now") { return time.Now().UTC(), nil }
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil { return time.Time{}, fmt.Errorf("%w: invalid timestamp %q: use RFC3339 or now", config.ErrConfig, value) }
	return parsed, nil
}

func formatTime(value time.Time) string {
	if value.IsZero() { return "" }
	return value.UTC().Format(time.RFC3339)
}

func validStatus(value string) bool { return in(value, "Available", "Preparing", "Charging", "SuspendedEVSE", "SuspendedEV", "Finishing", "Reserved", "Unavailable", "Faulted") }
func validErrorCode(value string) bool { return in(value, "ConnectorLockFailure", "EVCommunicationError", "GroundFailure", "HighTemperature", "InternalError", "LocalListConflict", "NoError", "OtherError", "OverCurrentFailure", "OverVoltage", "PowerMeterFailure", "PowerSwitchFailure", "ReaderFailure", "ResetFailure", "UnderVoltage", "WeakSignal") }
func validMeasurand(value string) bool { return in(value, "Current.Export", "Current.Import", "Current.Offered", "Energy.Active.Export.Register", "Energy.Active.Import.Register", "Energy.Reactive.Export.Register", "Energy.Reactive.Import.Register", "Energy.Active.Export.Interval", "Energy.Active.Import.Interval", "Energy.Reactive.Export.Interval", "Energy.Reactive.Import.Interval", "Frequency", "Power.Active.Export", "Power.Active.Import", "Power.Factor", "Power.Offered", "Power.Reactive.Export", "Power.Reactive.Import", "RPM", "SoC", "Temperature", "Voltage") }
func validUnit(value string) bool { return in(value, "Wh", "kWh", "varh", "kvarh", "W", "kW", "VA", "kVA", "var", "kvar", "A", "V", "Celsius", "Fahrenheit", "K", "Percent") }
func validReadingContext(value string) bool { return in(value, "Interruption.Begin", "Interruption.End", "Other", "Sample.Clock", "Sample.Periodic", "Transaction.Begin", "Transaction.End", "Trigger") }
func validLocation(value string) bool { return value == "" || in(value, "Body", "Cable", "EV", "Inlet", "Outlet") }
func validPhase(value string) bool { return value == "" || in(value, "L1", "L2", "L3", "N", "L1-N", "L2-N", "L3-N", "L1-L2", "L2-L3", "L3-L1") }
func validStopReason(value string) bool { return in(value, "DeAuthorized", "EmergencyStop", "EVDisconnected", "HardReset", "Local", "Other", "PowerLoss", "Reboot", "Remote", "SoftReset", "UnlockCommand") }
func in(value string, allowed ...string) bool {
	for _, candidate := range allowed { if value == candidate { return true } }
	return false
}
