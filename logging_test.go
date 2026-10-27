package main

import (
	"log/slog"
	"testing"
)

func TestParseLogLevel(t *testing.T) {
	cases := []struct {
		input string
		want  slog.Level
	}{
		{input: "debug", want: slog.LevelDebug},
		{input: "INFO", want: slog.LevelInfo},
		{input: "warn", want: slog.LevelWarn},
		{input: "warning", want: slog.LevelWarn},
		{input: "error", want: slog.LevelError},
	}

	for _, testCase := range cases {
		t.Run(testCase.input, func(t *testing.T) {
			got, err := parseLogLevel(testCase.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != testCase.want {
				t.Fatalf("parseLogLevel(%q) = %v, want %v", testCase.input, got, testCase.want)
			}
		})
	}

	if _, err := parseLogLevel("verbose"); err == nil {
		t.Fatal("parseLogLevel(verbose) returned no error")
	}
}

func TestDefaultLogFormat(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	if got := defaultLogFormat(); got != "json" {
		t.Fatalf("defaultLogFormat() = %q, want json", got)
	}

	t.Setenv("APP_ENV", "development")
	if got := defaultLogFormat(); got != "text" {
		t.Fatalf("defaultLogFormat() = %q, want text", got)
	}
}

func TestNewLoggerRejectsInvalidConfig(t *testing.T) {
	t.Setenv("LOG_LEVEL", "verbose")
	if _, err := newLogger(); err == nil {
		t.Fatal("newLogger() accepted an invalid LOG_LEVEL")
	}

	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("LOG_FORMAT", "xml")
	if _, err := newLogger(); err == nil {
		t.Fatal("newLogger() accepted an invalid LOG_FORMAT")
	}

	t.Setenv("LOG_FORMAT", "json")
	if _, err := newLogger(); err != nil {
		t.Fatalf("newLogger() returned an error: %v", err)
	}
}
