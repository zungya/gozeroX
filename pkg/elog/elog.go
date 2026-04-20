package elog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	centralLogDir  = "logs"
	centralLogFile = "error.log"
)

type centralErrorWriter struct {
	file    *os.File
	service string
	mu      sync.Mutex
}

// Setup initializes centralized error logging. Call AFTER logx.MustSetup or
// after the framework has initialized its default writer (rest.MustNewServer /
// zrpc.MustNewServer). All Error/Severe/Stack/Alert logs will be duplicated
// to logs/error.log in addition to the per-service error log.
func Setup(serviceName string) {
	if err := os.MkdirAll(centralLogDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "elog: failed to create log directory: %v\n", err)
		return
	}

	f, err := os.OpenFile(
		filepath.Join(centralLogDir, centralLogFile),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "elog: failed to open central error log: %v\n", err)
		return
	}

	logx.AddWriter(&centralErrorWriter{
		file:    f,
		service: serviceName,
	})
}

func (w *centralErrorWriter) Error(v any, fields ...logx.LogField) {
	w.writeEntry("error", v, fields...)
}

func (w *centralErrorWriter) Severe(v any) {
	w.writeEntry("severe", v)
}

func (w *centralErrorWriter) Stack(v any) {
	w.writeEntry("stack", v)
}

func (w *centralErrorWriter) Alert(v any) {
	w.writeEntry("alert", v)
}

// Non-error methods: no-op
func (w *centralErrorWriter) Debug(v any, fields ...logx.LogField) {}
func (w *centralErrorWriter) Info(v any, fields ...logx.LogField)  {}
func (w *centralErrorWriter) Slow(v any, fields ...logx.LogField)  {}
func (w *centralErrorWriter) Stat(v any, fields ...logx.LogField)  {}

func (w *centralErrorWriter) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func (w *centralErrorWriter) writeEntry(level string, v any, fields ...logx.LogField) {
	entry := make(map[string]any, len(fields)+4)
	entry["@timestamp"] = time.Now().Format(time.RFC3339)
	entry["level"] = level
	entry["service"] = w.service
	entry["content"] = fmt.Sprintf("%v", v)
	for _, f := range fields {
		entry[f.Key] = fmt.Sprintf("%v", f.Value)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	w.mu.Lock()
	w.file.Write(data)
	w.file.Write([]byte("\n"))
	w.mu.Unlock()
}
