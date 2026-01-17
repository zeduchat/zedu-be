package utility

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/davecgh/go-spew/spew"
	log "github.com/jeanphorn/log4go"
	"github.com/natefinch/lumberjack"
)

var (
	AppDir    string
	AppDirErr error
)

// Logger application logger
type Logger struct {
	logger *log.Filter
}

type LogRecord struct {
	Level    string // The log level
	Date     string // The time at which the log message was created (nanoseconds)
	Source   string // The message source
	Message  string // The log message
	Category string // The log group
}

// AuditLog Audit log
type AuditLog struct {
	Date           time.Time `json:"date"`
	Username       string    `json:"Username"`
	RequestHeader  any       `json:"request_header"`
	Request        any       `json:"request"`
	StatusCode     int       `json:"status_code"`
	ResponseHeader any       `json:"response_header"`
	Response       any       `json:"response"`
	ClientID       string    `json:"client_id"`
	Route          string    `json:"route"`
	Duration       float64   `json:"duration (seconds)"`
}

// NewLogger constructs a logger object
func NewLogger() *Logger {
	folder := "./logs"
	logSettingsPath := "./log.json"
	// appDir, err := os.Getwd()
	// if err != nil {
	// 	fmt.Printf("Could not load log location >> ", err)
	// }
	_, err := log.ReadFile(logSettingsPath)
	if err != nil {
		logSettingsPath = "../log.json"
		_, err := log.ReadFile(logSettingsPath)
		if err != nil {
			return &Logger{}
		} else {
			folder = "../logs"
		}
	}

	_ = os.Mkdir(folder, os.ModePerm)

	// log.LoadConfiguration(appDir + string(os.PathSeparator) + logSettingsPath)
	log.LoadConfiguration(logSettingsPath)

	return &Logger{
		logger: log.LOGGER("fileLogs"),
	}
}

// Close closes the logger and flushes all pending writes
func (l *Logger) Close() {
	if l.logger != nil {
		// Give time for pending log writes to complete
		time.Sleep(100 * time.Millisecond)
		l.logger.Close()
	}
}

// Info log information
func (l *Logger) Info(arg0 any, args ...any) {
	if l == nil || l.logger == nil {
		return
	}

	// Recover from "send on closed channel" panic from log4go
	defer func() {
		recover()
	}()

	var msg string

	if format, ok := arg0.(string); ok && len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	} else {
		msg = fmt.Sprint(append([]any{arg0}, args...)...)
	}

	l.logger.Log(log.INFO, getSource(), msg)
}

// Debug log debug
func (l *Logger) Debug(arg0 any, args ...any) {
	if l == nil || l.logger == nil {
		return
	}
	l.logger.Log(log.DEBUG, getSource(), fmt.Sprintf(arg0.(string), args...))
}

// Warning log warnings
func (l *Logger) Warning(arg0 any, args ...any) {
	if l == nil || l.logger == nil {
		return
	}
	l.logger.Log(log.WARNING, getSource(), fmt.Sprintf(arg0.(string), args...))
}

// Error log errors
func (l *Logger) Error(arg0 any, args ...any) {
	if l == nil || l.logger == nil {
		return
	}
	l.logger.Log(log.ERROR, getSource(), fmt.Sprintf(arg0.(string), args...))
}

// Error log errors
func (l *Logger) mongoError(arg0 any, args ...any) {
	l.logger.Log(log.ERROR, getSource(), fmt.Sprintf(arg0.(string), args...))
}

// Info log errors
func (l *Logger) mongoInfo(arg0 any, args ...any) {
	l.logger.Log(log.INFO, getSource(), fmt.Sprintf(arg0.(string), args...))
}

// Fatal log fatal errors
func (l *Logger) Fatal(arg0 any, args ...any) {
	//record := LogRecord{
	//	Level:   "FATAL",
	//	Date: time.Now().Local().String(),
	//	Source:  getSource(),
	//	Message: fmt.Sprintf(arg0.(string), args...),
	//}
	//go record.Save(l)
	l.logger.Log(log.CRITICAL, getSource(), fmt.Sprintf(arg0.(string), args...))
	l.logger.Close()
	os.Exit(1)
}

// Audit : log information on api request and response
func (l *Logger) Audit(record *AuditLog) {
	js, _ := json.Marshal(record)
	l.logger.Log(log.INFO, getSource(), string(js))
}

func (l *Logger) LogToStdout(arg0 any, args ...any) {

}

func Header2Map(header http.Header) map[string]any {
	head := make(map[string]any)
	for k, v := range header {
		head[k] = v
	}
	return head
}

func getSource() (source string) {
	if pc, _, line, ok := runtime.Caller(2); ok {
		source = fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), line)
	}
	return
}

func SpewResultForDebugging(description string, v any) {
	fmt.Println()
	fmt.Println("**** Start Result ******")
	fmt.Println(description)
	spew.Dump(v)
	fmt.Println("**** End Result ******")
}

func NewRotatingLogger() *slog.Logger {
	// Ensure log folder exists
	if err := os.MkdirAll("./river_queue_logs", 0o755); err != nil {
		panic(fmt.Sprintf("failed to create log folder: %v", err))
	}

	// Build file path with today's date
	dateStr := time.Now().Format("2006-01-02")
	logFile := filepath.Join("./river_queue_logs", fmt.Sprintf("%s.log", dateStr))

	// Use lumberjack for rotation
	rotator := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    100, // megabytes before rolling
		MaxBackups: 30,  // keep last 30 files
		MaxAge:     30,  // keep logs for 30 days
		Compress:   true,
	}

	// Setup slog with JSON logs (structured)
	handler := slog.NewJSONHandler(rotator, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	return slog.New(handler)
}
