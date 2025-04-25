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

// ContextLogger 定义一个封装类型
type ContextLogger struct {
	base zerolog.Logger
}

// NewContextLogger 构造函数，接入你自己的全局 logger
func NewContextLogger(base zerolog.Logger) *ContextLogger {
	return &ContextLogger{base: base}
}

// Info 方法：优先从 ctx 里取 logger，否则用 base
func (l *ContextLogger) Info(ctx context.Context, msg string) {
	if ctxLogger := zerolog.Ctx(ctx); ctxLogger != nil {
		ctxLogger.Info().Msg(msg)
	} else {
		l.base.Info().Msg(msg)
	}
}

// Error 方法：同理，还可以带 err
func (l *ContextLogger) Error(ctx context.Context, err error, msg string) {
	if ctxLogger := zerolog.Ctx(ctx); ctxLogger != nil {
		ctxLogger.Error().Err(err).Msg(msg)
	} else {
		l.base.Error().Err(err).Msg(msg)
	}
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

func Init(serviceName string, maxSize int, maxBackups int, maxAge int) *ContextLogger {
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

	// 关键：替换包级默认 Logger
	log.Logger = globalLogger

	// ——创建一个支持 ctx 的全局 Logger——
	return NewContextLogger(globalLogger)
}
