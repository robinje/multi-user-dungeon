package core

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
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

// LOG_GROUP is the CloudWatch log group name
var LOG_GROUP = GetEnv("LOG_GROUP", "/"+APPLICATION_NAME)

// LOG_STREAM is the CloudWatch log stream name
var LOG_STREAM = GetEnv("LOG_STREAM", "application")

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

	// Initialize AWS SDK configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(REGION))
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}

	// Create CloudWatch Logs client
	client := cloudwatchlogs.NewFromConfig(cfg)

	// Create CloudWatch handler
	cwHandler := NewCloudWatchHandler(client, LOG_GROUP, LOG_STREAM)

	// Create a multi-writer handler that writes to both CloudWatch and stdout
	multiHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}).WithAttrs([]slog.Attr{
		slog.String("application", APPLICATION_NAME),
		slog.String("region", REGION),
	})

	// Initialize the Logger with both handlers
	Logger = slog.New(NewMultiHandler(multiHandler, cwHandler))
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

func NewCloudWatchHandler(client *cloudwatchlogs.Client, logGroup, logStream string) *CloudWatchHandler {
	return &CloudWatchHandler{
		client:    client,
		logGroup:  logGroup,
		logStream: logStream,
	}
}

func (h *CloudWatchHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *CloudWatchHandler) Handle(ctx context.Context, r slog.Record) error {
	message := r.Message
	for _, attr := range h.attrs {
		message += fmt.Sprintf(" %s=%v", attr.Key, attr.Value)
	}
	r.Attrs(func(a slog.Attr) bool {
		message += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		return true
	})

	_, err := h.client.PutLogEvents(ctx, &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(h.logGroup),
		LogStreamName: aws.String(h.logStream),
		LogEvents: []types.InputLogEvent{
			{
				Message:   aws.String(message),
				Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
			},
		},
	})
	return err
}

func (h *CloudWatchHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CloudWatchHandler{
		client:    h.client,
		logGroup:  h.logGroup,
		logStream: h.logStream,
		attrs:     append(h.attrs, attrs...),
	}
}

func (h *CloudWatchHandler) WithGroup(name string) slog.Handler {
	return h
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return NewMultiHandler(newHandlers...)
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return NewMultiHandler(newHandlers...)
}
