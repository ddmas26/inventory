package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// slogGormLogger routes GORM's query logs through slog, so that every line the
// process prints — access logs, application logs and SQL — shares one JSON
// format. Without it, GORM would keep writing its own non-JSON text to stdout.
type slogGormLogger struct {
	level logger.LogLevel
}

// newSlogGormLogger builds a GORM logger that emits through the default slog logger.
func newSlogGormLogger(level logger.LogLevel) logger.Interface {
	return &slogGormLogger{level: level}
}

// LogMode implements logger.Interface; it returns a logger at the new level.
func (l *slogGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &slogGormLogger{level: level}
}

func (l *slogGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Info {
		slog.InfoContext(ctx, fmt.Sprintf(msg, data...))
	}
}

func (l *slogGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Warn {
		slog.WarnContext(ctx, fmt.Sprintf(msg, data...))
	}
}

func (l *slogGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Error {
		slog.ErrorContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Trace logs one SQL statement. Errors are always logged (except the benign
// record-not-found); successful statements are logged only at info level, which
// is exactly what EnableSQLLogging switches on.
func (l *slogGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		slog.ErrorContext(ctx, "sql error",
			"sql", sql, "rows", rows, "duration_ms", elapsed.Milliseconds(), "error", err)
	case l.level >= logger.Info:
		slog.InfoContext(ctx, "sql",
			"sql", sql, "rows", rows, "duration_ms", elapsed.Milliseconds())
	}
}
