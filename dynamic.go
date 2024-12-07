package asklog

import (
	"context"
	"log/slog"
	"sync/atomic"
)

type LvlSetter interface {
	SetLogLevel(lvl slog.Level)
}

// DynamicLogHandler allow runtime-configuration of the log handler.
type DynamicLogHandler struct {
	h      slog.Handler
	minLvl atomic.Int64 // slog.Level // shared with derived dynamic handlers
}

func NewDynamicLogHandler(lvl slog.Level, h slog.Handler) *DynamicLogHandler {
	d := &DynamicLogHandler{
		h: h,
	}
	d.minLvl.Store(int64(lvl))
	return d
}

func (d *DynamicLogHandler) SetLogLevel(lvl slog.Level) {
	d.minLvl.Store(int64(lvl))
}

func (d *DynamicLogHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level < slog.Level(d.minLvl.Load()) { // higher log level values are more critical
		return nil
	}
	return d.h.Handle(ctx, r) // process the log
}

func (d *DynamicLogHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return (lvl >= slog.Level(d.minLvl.Load())) && d.h.Enabled(ctx, lvl)
}

func (d *DynamicLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h := &DynamicLogHandler{
		h: d.h.WithAttrs(attrs),
	}
	h.minLvl.Store(d.minLvl.Load())
	return h
}

func (d *DynamicLogHandler) WithGroup(name string) slog.Handler {
	h := &DynamicLogHandler{
		h: d.h.WithGroup(name),
	}
	h.minLvl.Store(d.minLvl.Load())
	return h
}
