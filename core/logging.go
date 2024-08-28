package core

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/aws/aws-xray-sdk-go/xray"
)

// Global variables
var (
	Logger *slog.Logger
)

// APPLICATION_NAME is the name of the application, defaulting to "dark_relics"
var APPLICATION_NAME = GetEnv("APPLICATION_NAME", "dark_relics")

// REGION is the primary AWS region for Observatory resources
var REGION = GetEnv("AWS_REGION", "us-east-1")

// LOG_LEVEL is the logging level (default: 20)
var LOG_LEVEL, _ = strconv.Atoi(GetEnv("LOGGING", "20"))

func init() {

	// Determine the log level
	var level slog.Level
	switch LOG_LEVEL {
	case 10:
		level = slog.LevelDebug
	case 20:
		level = slog.LevelInfo
	case 30:
		level = slog.LevelWarn
	case 40:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Initialize the Logger
	Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(Logger)

}

// GetEnv retrieves environment variables or returns a default value if not set
func GetEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func EnableXRay() error {
	// Determine the log level
	var xrayLogLevel string
	switch LOG_LEVEL {
	case 10:
		xrayLogLevel = "debug"
	case 20:
		xrayLogLevel = "info"
	case 30:
		xrayLogLevel = "warn"
	case 40:
		xrayLogLevel = "error"
	default:
		xrayLogLevel = "info"
	}

	Logger.Info("Configuring AWS X-Ray", "logLevel", xrayLogLevel)

	err := xray.Configure(xray.Config{
		LogLevel:       xrayLogLevel,
		ServiceVersion: APPLICATION_NAME, // Assuming APPLICATION_NAME is your service version
	})

	if err != nil {
		Logger.Error("Failed to configure AWS X-Ray", "error", err)
		return fmt.Errorf("failed to configure AWS X-Ray: %w", err)
	}

	Logger.Info("AWS X-Ray successfully configured")

	return nil
}
