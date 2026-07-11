package ocppclient

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

type connectorState struct {
	availability  core.AvailabilityType
	status        core.ChargePointStatus
	transactionID int
	idTag         string
	meter         int
}

type simulator struct {
	cfg       config.ClientConfig
	opts      SimulatorOptions
	cp        ocpp16.ChargePoint
	events    chan SimulatorEvent
	closeOnce sync.Once

	eventMu      sync.RWMutex
	eventsClosed bool

	mu                   sync.Mutex
	ctx                  context.Context
	connectors           map[int]*connectorState
	configuration        map[string]core.ConfigurationKey
	reservations         map[int]*reservationState
	localAuthList        map[string]*types.IdTagInfo
	localAuthListVersion int
	chargingProfiles     map[int]storedChargingProfile
	diagnosticsStatus    firmware.DiagnosticsStatus
	firmwareStatus       firmware.FirmwareStatus
	lastBoot             BootResult
}

// NewSimulator creates a persistent OCPP 1.6 charge-point simulator.
func NewSimulator(cfg config.ClientConfig, opts SimulatorOptions) (Simulator, error) {
	if err := config.ValidateClientConfig(cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if opts.Connectors <= 0 {
		return nil, fmt.Errorf("%w: connectors must be greater than zero", ErrValidation)
	}
	if opts.HeartbeatInterval < 0 || opts.MeterInterval < 0 {
		return nil, fmt.Errorf("%w: intervals cannot be negative", ErrValidation)
	}
	if opts.MeterStart < 0 || opts.MeterStep < 0 {
		return nil, fmt.Errorf("%w: meter values cannot be negative", ErrValidation)
	}

	configureLogging(cfg.Verbose, cfg.Debug)
	wsClient, err := newWebSocketClient(cfg)
	if err != nil {
		return nil, err
	}
	cp := ocpp16.NewChargePoint(cfg.ChargePointID, nil, wsClient)
	s := &simulator{
		cfg:                cfg,
		opts:               opts,
		cp:                 cp,
		events:             make(chan SimulatorEvent, 256),
		connectors:         make(map[int]*connectorState, opts.Connectors),
		configuration:      make(map[string]core.ConfigurationKey),
		reservations:       make(map[int]*reservationState),
		localAuthList:      make(map[string]*types.IdTagInfo),
		chargingProfiles:   make(map[int]storedChargingProfile),
		diagnosticsStatus:  firmware.DiagnosticsStatusIdle,
		firmwareStatus:     firmware.FirmwareStatusIdle,
	}
	for id := 1; id <= opts.Connectors; id++ {
		s.connectors[id] = &connectorState{
			availability: core.AvailabilityTypeOperative,
			status:       core.ChargePointStatusAvailable,
			meter:        opts.MeterStart,
		}
	}

	s.addConfiguration("NumberOfConnectors", strconv.Itoa(opts.Connectors), true)
	s.addConfiguration("SupportedFeatureProfiles", "Core,FirmwareManagement,LocalAuthListManagement,Reservation,RemoteTrigger,SmartCharging", true)
	s.addConfiguration("AuthorizeRemoteTxRequests", "false", false)
	s.addConfiguration("HeartbeatInterval", strconv.Itoa(int(opts.HeartbeatInterval.Seconds())), false)
	s.addConfiguration("MeterValueSampleInterval", strconv.Itoa(int(opts.MeterInterval.Seconds())), false)
	s.addConfiguration("LocalAuthListEnabled", "true", false)
	s.addConfiguration("LocalAuthListMaxLength", "1000", true)
	s.addConfiguration("ReserveConnectorZeroSupported", "true", true)
	s.addConfiguration("ChargeProfileMaxStackLevel", "10", true)
	s.addConfiguration("ChargingScheduleAllowedChargingRateUnit", "Current,Power", true)
	s.addConfiguration("ChargingScheduleMaxPeriods", "24", true)
	s.addConfiguration("MaxChargingProfilesInstalled", "100", true)

	cp.SetCoreHandler(s)
	cp.SetFirmwareManagementHandler(s)
	cp.SetLocalAuthListHandler(s)
	cp.SetReservationHandler(s)
	cp.SetRemoteTriggerHandler(s)
	cp.SetSmartChargingHandler(s)
	return s, nil
}

func (s *simulator) Events() <-chan SimulatorEvent { return s.events }

func (s *simulator) Run(ctx context.Context) error {
	s.mu.Lock()
	s.ctx = ctx
	s.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil
	default:
	}
	if err := s.cp.Start(strings.TrimRight(s.cfg.CentralSystemURL, "/")); err != nil {
		return classify(err, ErrConnection)
	}
	defer s.close()
	s.emit("system", "Connect", "Connected", nil, s.cfg.CentralSystemURL+"/"+s.cfg.ChargePointID)

	bootConfirmation, err := s.sendBootNotification(ctx)
	if err != nil {
		return err
	}
	if bootConfirmation.Status == core.RegistrationStatusRejected {
		return fmt.Errorf("%w: BootNotification status is %s", ErrRejected, bootConfirmation.Status)
	}

	if err := s.sendInitialStatuses(ctx); err != nil {
		return err
	}

	heartbeatInterval := s.opts.HeartbeatInterval
	if heartbeatInterval == 0 && bootConfirmation.Interval > 0 {
		heartbeatInterval = time.Duration(bootConfirmation.Interval) * time.Second
	}
	heartbeatTicker, heartbeatC := optionalTicker(heartbeatInterval)
	defer stopTicker(heartbeatTicker)
	meterTicker, meterC := optionalTicker(s.opts.MeterInterval)
	defer stopTicker(meterTicker)
	reservationTicker := time.NewTicker(500 * time.Millisecond)
	defer reservationTicker.Stop()

	errorC := s.cp.Errors()
	for {
		select {
		case <-ctx.Done():
			s.emit("system", "Shutdown", "Stopped", nil, "context cancelled")
			return nil
		case err, ok := <-errorC:
			if ok && err != nil {
				return classify(err, ErrConnection)
			}
			errorC = nil
		case <-heartbeatC:
			if err := s.sendHeartbeat(ctx); err != nil {
				return err
			}
		case <-meterC:
			if err := s.sendMeterValues(ctx); err != nil {
				return err
			}
		case <-reservationTicker.C:
			s.expireReservations(ctx)
		}
	}
}

func (s *simulator) close() {
	s.closeOnce.Do(func() {
		if s.cp.IsConnected() {
			s.cp.Stop()
		}
		s.eventMu.Lock()
		s.eventsClosed = true
		close(s.events)
		s.eventMu.Unlock()
	})
}

func optionalTicker(interval time.Duration) (*time.Ticker, <-chan time.Time) {
	if interval <= 0 {
		return nil, nil
	}
	ticker := time.NewTicker(interval)
	return ticker, ticker.C
}

func stopTicker(ticker *time.Ticker) {
	if ticker != nil {
		ticker.Stop()
	}
}

func (s *simulator) sendBootNotification(ctx context.Context) (*core.BootNotificationConfirmation, error) {
	confirmation, err := await(ctx, func() (*core.BootNotificationConfirmation, error) {
		return s.cp.BootNotification(s.cfg.ChargePointModel, s.cfg.ChargePointVendor, func(request *core.BootNotificationRequest) {
			request.FirmwareVersion = s.cfg.FirmwareVersion
			request.ChargePointSerialNumber = s.cfg.SerialNumber
			request.MeterSerialNumber = s.cfg.MeterSerialNumber
			request.MeterType = s.cfg.MeterType
		})
	})
	if err != nil {
		return nil, classify(err, ErrProtocol)
	}
	s.mu.Lock()
	s.lastBoot = BootResult{Status: string(confirmation.Status), Interval: confirmation.Interval}
	if confirmation.CurrentTime != nil {
		s.lastBoot.CurrentTime = confirmation.CurrentTime.Time
	}
	s.mu.Unlock()
	s.emit("outbound", core.BootNotificationFeatureName, string(confirmation.Status), nil, "")
	return confirmation, nil
}

func (s *simulator) sendInitialStatuses(ctx context.Context) error {
	if err := s.notifyStatus(ctx, 0, core.ChargePointStatusAvailable); err != nil {
		return err
	}
	for id := 1; id <= s.opts.Connectors; id++ {
		if err := s.notifyStatus(ctx, id, core.ChargePointStatusAvailable); err != nil {
			return err
		}
	}
	return nil
}

func (s *simulator) sendHeartbeat(ctx context.Context) error {
	confirmation, err := await(ctx, func() (*core.HeartbeatConfirmation, error) {
		return s.cp.Heartbeat()
	})
	if err != nil {
		return classify(err, ErrProtocol)
	}
	detail := ""
	if confirmation.CurrentTime != nil {
		detail = confirmation.CurrentTime.FormatTimestamp()
	}
	s.emit("outbound", core.HeartbeatFeatureName, "Accepted", nil, detail)
	return nil
}

func (s *simulator) sendMeterValues(ctx context.Context) error {
	type sample struct {
		id    int
		meter int
		txID  int
	}
	s.mu.Lock()
	samples := make([]sample, 0, len(s.connectors))
	for id, connector := range s.connectors {
		connector.meter += s.opts.MeterStep
		samples = append(samples, sample{id: id, meter: connector.meter, txID: connector.transactionID})
	}
	s.mu.Unlock()
	sort.Slice(samples, func(i, j int) bool { return samples[i].id < samples[j].id })
	for _, current := range samples {
		if err := s.sendMeterSample(ctx, current.id, current.meter, current.txID, types.ReadingContextSamplePeriodic); err != nil {
			return err
		}
	}
	return nil
}

func (s *simulator) sendMeterSample(ctx context.Context, connectorID, meter, transactionID int, readingContext types.ReadingContext) error {
	sampled := types.SampledValue{
		Value:     strconv.Itoa(meter),
		Context:   readingContext,
		Format:    types.ValueFormatRaw,
		Measurand: types.MeasurandEnergyActiveImportRegister,
		Location:  types.LocationOutlet,
		Unit:      types.UnitOfMeasureWh,
	}
	meterValue := types.MeterValue{Timestamp: types.Now(), SampledValue: []types.SampledValue{sampled}}
	_, err := await(ctx, func() (*core.MeterValuesConfirmation, error) {
		return s.cp.MeterValues(connectorID, []types.MeterValue{meterValue}, func(request *core.MeterValuesRequest) {
			if transactionID > 0 {
				request.TransactionId = &transactionID
			}
		})
	})
	if err != nil {
		return classify(err, ErrProtocol)
	}
	s.emit("outbound", core.MeterValuesFeatureName, "Accepted", &connectorID, strconv.Itoa(meter)+" Wh")
	return nil
}

func (s *simulator) notifyStatus(ctx context.Context, connectorID int, status core.ChargePointStatus) error {
	_, err := await(ctx, func() (*core.StatusNotificationConfirmation, error) {
		return s.cp.StatusNotification(connectorID, core.NoError, status, func(request *core.StatusNotificationRequest) {
			request.Timestamp = types.Now()
		})
	})
	if err != nil {
		return classify(err, ErrProtocol)
	}
	s.emit("outbound", core.StatusNotificationFeatureName, string(status), &connectorID, "")
	return nil
}

func (s *simulator) notifyStatusAsync(connectorID int, status core.ChargePointStatus) {
	if !s.cp.IsConnected() {
		return
	}
	go func() { _ = s.notifyStatus(s.runContext(), connectorID, status) }()
}

func (s *simulator) emit(direction, action, status string, connectorID *int, detail string) {
	event := SimulatorEvent{Timestamp: time.Now().UTC(), Direction: direction, Action: action, Status: status, ConnectorID: connectorID, Detail: detail}
	s.eventMu.RLock()
	defer s.eventMu.RUnlock()
	if s.eventsClosed {
		return
	}
	select {
	case s.events <- event:
	default:
	}
}

func (s *simulator) addConfiguration(key, value string, readonly bool) {
	copyValue := value
	s.configuration[key] = core.ConfigurationKey{Key: key, Readonly: readonly, Value: &copyValue}
}
