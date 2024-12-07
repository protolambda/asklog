package asklog

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/ethereum/go-ethereum/log"
)

type Format string

const (
	FormatTerminal Format = "terminal"
	FormatLogFmt   Format = "logfmt"
	FormatJSON     Format = "json"
)

func (f *Format) String() string {
	if f == nil {
		return ""
	}
	return string(*f)
}

func (f *Format) Type() string {
	return "Logging format"
}

func (f *Format) Set(v string) error {
	if f == nil {
		return errors.New("cannot set nil Format")
	}
	x := Format(v)
	switch x {
	case FormatTerminal, FormatLogFmt, FormatJSON:
		*f = x
	default:
		return fmt.Errorf("unrecognized format: %q", v)
	}
	return nil
}

func (f Format) Handler(color bool) func(writer io.Writer) slog.Handler {
	switch f {
	case FormatJSON:
		return log.JSONHandler
	case FormatTerminal:
		return func(w io.Writer) slog.Handler {
			return log.NewTerminalHandler(w, color)
		}
	case FormatLogFmt:
		return func(w io.Writer) slog.Handler {
			return log.LogfmtHandlerWithLevel(w, log.LevelTrace)
		}
	default:
		return nil
	}
}

type Level slog.Level

func (lvl Level) String() string {
	return slog.Level(lvl).String()
}

func (lvl *Level) Set(v string) error {
	if lvl == nil {
		return errors.New("cannot set nil Level")
	}
	lower := strings.ToLower(v) // ignore case
	switch lower {
	case "trace", "trce":
		*lvl = Level(log.LevelTrace)
	case "debug", "dbug":
		*lvl = Level(log.LevelDebug)
	case "info":
		*lvl = Level(log.LevelInfo)
	case "warn":
		*lvl = Level(log.LevelWarn)
	case "error", "eror", "err":
		*lvl = Level(log.LevelError)
	case "crit":
		*lvl = Level(log.LevelCrit)
	default:
		return fmt.Errorf("unknown level: %q", v)
	}
	return nil
}

func (lvl Level) Type() string {
	return "log level"
}

func (lvl Level) Level() slog.Level {
	return slog.Level(lvl)
}

type Config struct {
	// Out to write log data to. If nil, os.Stdout will be used.
	Out io.Writer `ask:"-"`

	Level  Level  `ask:"--log.level" help:"Log level: 'trace', 'debug', 'info', 'warn', 'error', 'crit'. Aliases and mixed-case are excepted."`
	Format Format `ask:"--log.format" help:"Log format: 'text', 'terminal', 'logfmt', 'json'."`
	Color  bool   `ask:"--log.color" help:"Enable log coloring (terminal format only)"`
}

func (c *Config) Default() {
	if c.Out == nil {
		c.Out = os.Stdout
	}
	c.Format = FormatTerminal
	c.Color = term.IsTerminal(int(os.Stdout.Fd()))
	c.Level = Level(slog.LevelInfo)
}

func (c *Config) New() log.Logger {
	hFn := c.Format.Handler(c.Color)
	h := hFn(c.Out)
	h = NewDynamicLogHandler(c.Level.Level(), h)
	l := log.NewLogger(h)
	return l
}
