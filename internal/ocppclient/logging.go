package ocppclient

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/lorenzodonini/ocpp-go/ocppj"
	"github.com/lorenzodonini/ocpp-go/ws"
)

type protocolLogger struct {
	out     io.Writer
	verbose bool
	debug   bool
	mu      sync.Mutex
}

func configureLogging(verbose, debug bool) {
	logger := &protocolLogger{out: os.Stderr, verbose: verbose || debug, debug: debug}
	ws.SetLogger(logger)
	ocppj.SetLogger(logger)
}

func (l *protocolLogger) write(prefix, format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.out, "%s", prefix)
	fmt.Fprintf(l.out, format, args...)
	fmt.Fprintln(l.out)
}

func (l *protocolLogger) Debug(args ...interface{}) {
	if l.debug {
		l.write("ocpp debug: ", "%s", fmt.Sprint(args...))
	}
}
func (l *protocolLogger) Debugf(format string, args ...interface{}) {
	if l.debug {
		l.write("ocpp debug: ", format, args...)
	}
}
func (l *protocolLogger) Info(args ...interface{}) {
	if l.verbose {
		l.write("ocpp: ", "%s", fmt.Sprint(args...))
	}
}
func (l *protocolLogger) Infof(format string, args ...interface{}) {
	if l.verbose {
		l.write("ocpp: ", format, args...)
	}
}
func (l *protocolLogger) Error(args ...interface{}) {
	if l.verbose {
		l.write("ocpp error: ", "%s", fmt.Sprint(args...))
	}
}
func (l *protocolLogger) Errorf(format string, args ...interface{}) {
	if l.verbose {
		l.write("ocpp error: ", format, args...)
	}
}
