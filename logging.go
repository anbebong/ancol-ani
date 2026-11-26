package main

// Single shared log file for the widget process + its local HTTP server
// (not the settings window process - main.go spawns that separately as
// `<exe> settings`, see runSettingsApp - and not cmd/syncserver, which has
// its own unrelated stdout logging). Replaces the old debugLogEmit, which
// was TEMPORARY and only ever logged emitServerText calls: this one is
// meant to stay, is leveled/timestamped, and caps its own size instead of
// growing forever on a machine that leaves the widget running for weeks.

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	logFileName = "app.log"
	// maxLogBytes: once a write would push the file over this, it's
	// truncated back to empty first. Simpler than rotating to .1/.2 -
	// losing old lines on truncation is fine for a desktop widget's own
	// diagnostic log, this isn't an audit trail.
	maxLogBytes = 2 << 20 // 2MB
)

var logMu sync.Mutex

func logFilePath() string {
	return filepath.Join(appDataDir(), logFileName)
}

// logLine is the one place every log write in this app goes through.
// Format: "2006-01-02 15:04:05.000 [LEVEL] message\n" - exactly one line
// per call; a message containing its own "\n" breaks that and is the
// caller's mistake, not something this function fixes up.
func logLine(level, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s\n", time.Now().Format("2006-01-02 15:04:05.000"), level, msg)

	logMu.Lock()
	defer logMu.Unlock()

	path := logFilePath()
	if info, err := os.Stat(path); err == nil && info.Size()+int64(len(line)) > maxLogBytes {
		os.Remove(path)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(line)
}

func logInfo(format string, args ...any)  { logLine("INFO", format, args...) }
func logWarn(format string, args ...any)  { logLine("WARN", format, args...) }
func logError(format string, args ...any) { logLine("ERROR", format, args...) }
