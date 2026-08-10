package server

import (
	"net/http"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogRequestFailureLevels(t *testing.T) {
	core, observed := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	logRequestFailure(logger, http.StatusRequestTimeout, zap.String("path", "/"))
	logRequestFailure(logger, http.StatusInternalServerError, zap.String("path", "/api"))

	entries := observed.All()
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[0].Level != zapcore.DebugLevel || entries[0].Message != "http connection timed out" {
		t.Fatalf("timeout entry = %#v", entries[0])
	}
	if entries[1].Level != zapcore.ErrorLevel || entries[1].Message != "http request failed" {
		t.Fatalf("server error entry = %#v", entries[1])
	}
}
