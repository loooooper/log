package log

import (
	"context"
	"io"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	MAXSIZE    = 1024
	MAXBACKUPS = 7
	MAXAGE     = 30
)

// ContextLogger 定义一个封装类型
type ContextLogger struct {
	base zerolog.Logger
}

// NewContextLogger 构造函数
func NewContextLogger(base zerolog.Logger) *ContextLogger {
	return &ContextLogger{base: base}
}

func (l *ContextLogger) Info(ctx context.Context, msg string) {
	if ctxLogger := zerolog.Ctx(ctx); ctxLogger != nil {
		ctxLogger.Info().Msg(msg)
	} else {
		l.base.Info().Msg(msg)
	}
}

func (l *ContextLogger) Error(ctx context.Context, err error, msg string) {
	if ctxLogger := zerolog.Ctx(ctx); ctxLogger != nil {
		ctxLogger.Error().Err(err).Msg(msg)
	} else {
		l.base.Error().Err(err).Msg(msg)
	}
}

type Config struct {
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

type ctxKeyLogger struct{}

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}
		reqLogger := log.Logger.With().Str("traceId", traceID).Logger()
		ctx := reqLogger.WithContext(r.Context())

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Init(serviceName string, config Config) *ContextLogger {
	var (
		maxSize    = MAXSIZE
		maxBackups = MAXBACKUPS
		maxAge     = MAXAGE
	)
	if config.MaxSize > 0 {
		maxSize = config.MaxSize
	}

	if config.MaxBackups > 0 {
		maxBackups = config.MaxBackups
	}

	if config.MaxAge > 0 {
		maxAge = config.MaxAge
	}

	rotator := &lumberjack.Logger{
		Filename:   serviceName + ".log",
		MaxSize:    maxSize, // MB
		MaxBackups: maxBackups,
		MaxAge:     maxAge, // days
		Compress:   true,
	}
	zerolog.NewLevelHook()

	multi := io.MultiWriter(os.Stdout, rotator)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	globalLogger := zerolog.New(multi).With().Timestamp().Logger()

	log.Logger = globalLogger
	return NewContextLogger(globalLogger)
}
