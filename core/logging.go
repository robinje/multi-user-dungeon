package core

import (
	"context"
	"fmt"
	"log/slog"
	"os"
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

func InitializeLogging(cfg *Configuration) error {
	// Determine the log level
	var level slog.Level
	switch cfg.Logging.LogLevel {
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
	awsCfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(cfg.Aws.Region))
	if err != nil {
		return fmt.Errorf("unable to load SDK config: %w", err)
	}

	// Create CloudWatch Logs client
	client := cloudwatchlogs.NewFromConfig(awsCfg)

	// Create CloudWatch handler
	cwHandler := NewCloudWatchHandler(client, cfg.Logging.LogGroup, cfg.Logging.LogStream)

	// Create a multi-writer handler that writes to both CloudWatch and stdout
	multiHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}).WithAttrs([]slog.Attr{
		slog.String("application", cfg.Logging.ApplicationName),
		slog.String("region", cfg.Aws.Region),
	})

	// Initialize the Logger with both handlers
	Logger = slog.New(NewMultiHandler(multiHandler, cwHandler))
	slog.SetDefault(Logger)

	return nil
}

// GetEnv retrieves environment variables or returns a default value if not set
func GetEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func EnableXRay(cfg *Configuration) error {
	// Determine the log level
	var xrayLogLevel string
	switch cfg.Logging.LogLevel {
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
		LogLevel: xrayLogLevel,
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
