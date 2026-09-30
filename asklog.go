// Package asklog provides the log flags of command-line tools built with [github.com/protolambda/ask],
// and creates a [log.Logger] of [github.com/protolambda/proto-log/log] from them.
//
// Embed a [Config] in an ask command, with the `ask:"."` tag, to get the --log.* flags.
// Ask applies [Config.Default] before it parses the flags. The zero Config is usable as well.
package asklog

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"syscall"

	"golang.org/x/term"

	"github.com/protolambda/proto-log/log"
)

// Format is a log output format. As a flag value, it accepts the values of the Format constants.
type Format string

const (
	FormatTerminal Format = "terminal"
	FormatLogFmt   Format = "logfmt"
	FormatJSON     Format = "json"
)

// String returns the name of the format.
func (f *Format) String() string {
	if f == nil {
		return ""
	}
	return string(*f)
}

// Type describes the flag value in help output.
func (f *Format) Type() string {
	return "log format"
}

// Set sets the format by name. It returns an error for an unknown name.
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

// Handler returns the constructor of the log handler of the format,
// or nil if f is not one of FormatTerminal, FormatLogFmt and FormatJSON.
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

// Level is a log level. It is a slog.Leveler and a flag value.
type Level slog.Level

var _ slog.Leveler = Level(0)

// String returns the upper-case name of the level: "TRACE", "DEBUG", "INFO", "WARN", "ERROR" or "CRIT",
// and for the levels in between the name of slog.Level.String, such as "DEBUG-2" or "INFO+2".
// Set accepts every name that String returns; ask shows it as the default value in the help.
func (lvl Level) String() string {
	switch slog.Level(lvl) {
	case log.LevelTrace:
		return "TRACE"
	case log.LevelCrit:
		return "CRIT"
	default:
		return slog.Level(lvl).String()
	}
}

// Set sets the level by name: "trace", "debug", "info", "warn", "error" or "crit",
// in any case, or an alias of these (see log.LevelFromString).
// It also accepts a name with an offset, as parsed by slog.Level.UnmarshalText,
// such as "DEBUG-2" or "info+2", so that it accepts every name that String returns.
func (lvl *Level) Set(v string) error {
	if lvl == nil {
		return errors.New("cannot set nil Level")
	}
	x, err := log.LevelFromString(v)
	if err != nil {
		var sl slog.Level
		if sl.UnmarshalText([]byte(v)) != nil {
			return err
		}
		x = sl
	}
	*lvl = Level(x)
	return nil
}

// Type describes the flag value in help output.
func (lvl Level) Type() string {
	return "log level"
}

// Level returns the level as a slog.Level.
func (lvl Level) Level() slog.Level {
	return slog.Level(lvl)
}

// Config configures a logger, see New.
//
// The zero Config is usable: it logs info and higher levels to os.Stdout, in the terminal format,
// without color, time or source info.
// Default sets the command-line defaults instead.
type Config struct {
	// Out to write log data to. If nil, os.Stdout is used.
	Out io.Writer `ask:"-"`

	Level     Level  `ask:"--log.level" help:"Log level: 'trace', 'debug', 'info', 'warn', 'error' or 'crit'; aliases and mixed case are accepted"`
	Format    Format `ask:"--log.format" help:"Log format: 'terminal', 'logfmt' or 'json'"`
	Color     bool   `ask:"--log.color" help:"Enable log coloring (terminal format only)"`
	Time      bool   `ask:"--log.time" help:"Include time in logs"`
	Source    bool   `ask:"--log.src" help:"Include the source file and line number in logs"`
	SourceDir string `ask:"--log.src-dir" help:"Show source files (with --log.src) relative to this directory"`
}

// Default sets the default log configuration: Out is os.Stdout if nil,
// the terminal format, the info level, with time, without source info,
// and with color if Out is a terminal.
// Only an Out with a file descriptor, such as an *os.File, is detected as a terminal:
// a writer that wraps os.Stdout gets no color. A command that logs through such a writer
// can set Color in its own Default.
// Ask applies it automatically (see ask.InitDefault) when Config is embedded in a command,
// before the Default of the command itself, so the command may override these defaults.
// To log to another writer, set Out before Default runs, e.g. in the command literal.
func (c *Config) Default() {
	if c.Out == nil {
		c.Out = os.Stdout
	}
	c.Format = FormatTerminal
	c.Color = isTerminal(c.Out)
	c.Level = Level(slog.LevelInfo)
	c.Time = true
	c.Source = false
	c.SourceDir = ""
}

// isTerminal reports whether w is a terminal.
// Only a writer with a file descriptor, such as an *os.File, can be one.
// It uses SyscallConn rather than os.File.Fd, which would stop the deadlines of the file from working.
func isTerminal(w io.Writer) bool {
	sc, ok := w.(syscall.Conn)
	if !ok {
		return false
	}
	rc, err := sc.SyscallConn()
	if err != nil {
		return false
	}
	var terminal bool
	if err := rc.Control(func(fd uintptr) {
		terminal = term.IsTerminal(int(fd))
	}); err != nil {
		return false
	}
	return terminal
}

// New creates a logger from the config. It does not modify the config.
//
// A nil Out writes to os.Stdout, the empty Format selects FormatTerminal,
// and the zero Level is info (slog.LevelInfo).
// New panics if Format is neither empty nor a format that Format.Set accepts.
func (c *Config) New() log.Logger {
	out := c.Out
	if out == nil {
		out = os.Stdout
	}
	format := c.Format
	if format == "" {
		format = FormatTerminal
	}
	hFn := format.Handler()
	if hFn == nil {
		panic(fmt.Sprintf("asklog: unknown log format %q", string(format)))
	}
	h := hFn(out,
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
