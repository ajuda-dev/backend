package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoggerWritesJSONToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.log")
	t.Setenv(LOG_OUTPUT, path)
	t.Setenv(LOG_LEVEL, "info")
	Configure()
	t.Cleanup(releaseLogFile)

	Info("logger_file_probe")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected log file: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, `"message":"logger_file_probe"`) {
		t.Fatalf("missing probe message in %s: %s", path, body)
	}
	if !strings.Contains(body, `"level":"info"`) {
		t.Fatalf("expected info level in %s: %s", path, body)
	}
}

func TestLoggerErrorLevelInFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.log")
	t.Setenv(LOG_OUTPUT, path)
	t.Setenv(LOG_LEVEL, "info")
	Configure()
	t.Cleanup(releaseLogFile)

	Error("logger_error_probe", os.ErrPermission)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected log file: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, `"level":"error"`) {
		t.Fatalf("expected error level in %s: %s", path, body)
	}
	if !strings.Contains(body, "logger_error_probe") {
		t.Fatalf("missing error message in %s: %s", path, body)
	}
}

func TestLoggerHonorsErrorLevel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.log")
	t.Setenv(LOG_OUTPUT, path)
	t.Setenv(LOG_LEVEL, "error")
	Configure()
	t.Cleanup(releaseLogFile)

	Info("should_not_appear")
	Error("must_appear", os.ErrClosed)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected log file: %v", err)
	}
	body := string(data)
	if strings.Contains(body, "should_not_appear") {
		t.Fatalf("info log leaked at error level: %s", body)
	}
	if !strings.Contains(body, "must_appear") {
		t.Fatalf("missing error log: %s", body)
	}
}

func TestFileLogPathStdoutIsIgnored(t *testing.T) {
	t.Setenv(LOG_OUTPUT, "stdout")
	if got := fileLogPath(); got != "" {
		t.Fatalf("stdout should not be a file path, got %q", got)
	}
}

func releaseLogFile() {
	_ = os.Unsetenv(LOG_OUTPUT)
	Configure()
}
