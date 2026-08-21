package testutil

import (
	"sync"

	"github.com/obalunenko/logger"
)

// LogRecord is one detached entry captured by RecordingLogger.
type LogRecord struct {
	Level   string
	Message string
	Fields  map[string]any
}

type recordingLoggerState struct {
	mu      sync.Mutex
	records []LogRecord
}

// RecordingLogger is a concurrency-safe logger.Logger test double.
type RecordingLogger struct {
	state  *recordingLoggerState
	fields map[string]any
}

var _ logger.Logger = (*RecordingLogger)(nil)

// NewRecordingLogger constructs an empty structured-log recorder.
func NewRecordingLogger() *RecordingLogger {
	return &RecordingLogger{state: &recordingLoggerState{}}
}

// Records returns detached records in emission order.
func (log *RecordingLogger) Records() []LogRecord {
	if log == nil || log.state == nil {
		return nil
	}
	log.state.mu.Lock()
	defer log.state.mu.Unlock()
	records := make([]LogRecord, len(log.state.records))
	for index, record := range log.state.records {
		records[index] = LogRecord{
			Level:   record.Level,
			Message: record.Message,
			Fields:  cloneLogFields(record.Fields),
		}
	}
	return records
}

func (log *RecordingLogger) Debug(message string) { log.record("debug", message) }
func (log *RecordingLogger) Info(message string)  { log.record("info", message) }
func (log *RecordingLogger) Warn(message string)  { log.record("warn", message) }
func (log *RecordingLogger) Error(message string) { log.record("error", message) }
func (log *RecordingLogger) Fatal(message string) { log.record("fatal", message) }

func (log *RecordingLogger) WithError(err error) logger.Logger {
	return log.WithField("error", err)
}

func (log *RecordingLogger) WithField(key string, value any) logger.Logger {
	return log.WithFields(logger.Fields{key: value})
}

func (log *RecordingLogger) WithFields(fields logger.Fields) logger.Logger {
	derived := &RecordingLogger{state: log.state, fields: cloneLogFields(log.fields)}
	if derived.fields == nil {
		derived.fields = make(map[string]any, len(fields))
	}
	for key, value := range fields {
		derived.fields[key] = value
	}
	return derived
}

func (log *RecordingLogger) record(level, message string) {
	if log == nil || log.state == nil {
		return
	}
	log.state.mu.Lock()
	defer log.state.mu.Unlock()
	log.state.records = append(log.state.records, LogRecord{
		Level: level, Message: message, Fields: cloneLogFields(log.fields),
	})
}

func cloneLogFields(fields map[string]any) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	clone := make(map[string]any, len(fields))
	for key, value := range fields {
		clone[key] = value
	}
	return clone
}
