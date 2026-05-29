// Package logger provides a structured zap.Logger constructor for dependency
// injection via uber-fx, and a gRPC unary server interceptor for request logging.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New constructs a production-grade *zap.Logger with JSON encoding.
// The log level defaults to InfoLevel and is configurable via the level parameter.
//
// It is intended to be registered as an fx.Provider so that the single logger
// instance is shared across the application.
func New(level zapcore.Level) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return cfg.Build()
}
