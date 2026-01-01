package asklog

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"golang.org/x/term"

	"github.com/protolambda/proto-log/log"
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

func (f Format) Handler() func(writer io.Writer, opts ...log.FormatOption) slog.Handler {
	switch f {
	case FormatJSON:
		return log.JSONHandler
	case FormatTerminal:
		return log.TerminalHandler
	case FormatLogFmt:
		return log.LogfmtHandler
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
	x, err := log.LevelFromString(v)
	if err != nil {
		return err
	}
	*lvl = Level(x)
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

	Level     Level  `ask:"--log.level" help:"Log level: 'trace', 'debug', 'info', 'warn', 'error', 'crit'. Aliases and mixed-case are excepted."`
	Format    Format `ask:"--log.format" help:"Log format: 'text', 'terminal', 'logfmt', 'json'."`
	Color     bool   `ask:"--log.color" help:"Enable log coloring (terminal format only)"`
	Time      bool   `ask:"--log.time" help:"Include time in logs"`
	Source    bool   `ask:"--log.src" help:"Include source-file/number info in logs"`
	SourceDir string `ask:"--log.src-dir" help:"Resolve source-file info (if enabled) as relative to this dir"`
}

func (c *Config) Default() {
	if c.Out == nil {
		c.Out = os.Stdout
	}
	c.Format = FormatTerminal
	c.Color = term.IsTerminal(int(os.Stdout.Fd()))
	c.Level = Level(slog.LevelInfo)
	c.Time = true
	c.Source = false
	c.SourceDir = ""
}

func (c *Config) New() log.Logger {
	hFn := c.Format.Handler()
	h := hFn(c.Out,
		log.WithColor(c.Color),
		log.WithExcludeTime(!c.Time),
		log.WithIncludeSource(c.Source),
		log.WithSourceRelDir(c.SourceDir),
	)
	l := log.New(h,
		log.ContextMod(),
		log.LevelMod(c.Level.Level()),
	)
	return l
}
