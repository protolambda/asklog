package asklog_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/protolambda/ask"
	"github.com/protolambda/asklog"
	"github.com/protolambda/proto-log/log"
)

func TestMainCmdRun(t *testing.T) {
	var buf bytes.Buffer
	mainCmd := &MainCmd{}
	mainCmd.LogConfig.Out = &buf
	err := ask.Run(context.Background(), mainCmd, []string{
		"--log.level", "info",
		"--log.format", "terminal",
		"--log.color=true",
		"--foobar", "123",
	})
	if err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	t.Log("output:", output)
	if !strings.Contains(output, "INFO") {
		t.Fatal("expected info-levels log")
	}
	if !strings.Contains(output, "Hello world!") {
		t.Fatal("expected log message")
	}
	if !strings.Contains(output, "=123") {
		t.Fatal("expected log attribute")
	}
}

func TestMainCmdEnv(t *testing.T) {
	env := map[string]string{
		"LOG_LEVEL":  "debug",
		"LOG_FORMAT": "json",
	}
	ctx := ask.WithEnvFn(context.Background(), func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	})
	var buf bytes.Buffer
	mainCmd := &MainCmd{}
	mainCmd.LogConfig.Out = &buf
	if err := ask.Run(ctx, mainCmd, []string{"--foobar", "123"}); err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	t.Log("output:", output)
	if !strings.Contains(output, `"msg":"Catch the bugs!"`) {
		t.Fatal("expected debug-level JSON log message")
	}
}

func TestMainCmdHelp(t *testing.T) {
	err := ask.Run(context.Background(), &MainCmd{}, []string{"--help"})
	if !errors.Is(err, ask.HelpErr) {
		t.Fatalf("expected help error, got: %v", err)
	}
	output := ask.UsageFromErr(err)
	t.Log("help output:", output)
	for _, flag := range []string{
		"--log.level", "--log.format", "--log.color",
		"--log.time", "--log.src", "--log.src-dir", "--foobar",
	} {
		if !strings.Contains(output, flag) {
			t.Errorf("expected help option %q", flag)
		}
	}
	for _, text := range []string{
		"Log level: 'trace', 'debug', 'info', 'warn', 'error' or 'crit'; aliases and mixed case are accepted (default: INFO) (type: log level)",
		"Log format: 'terminal', 'logfmt' or 'json' (default: terminal) (type: log format)",
		"Include the source file and line number in logs (default: false)",
		"Show source files (with --log.src) relative to this directory (type: string)",
	} {
		if !strings.Contains(output, text) {
			t.Errorf("expected help text %q", text)
		}
	}
}

func TestConfigZero(t *testing.T) {
	var buf bytes.Buffer
	// Only Out is set: the zero Format, Level, Color, Time, Source and SourceDir must be usable.
	cfg := asklog.Config{Out: &buf}
	logger := cfg.New()
	logger.Info("Hello world!", "foobar", 123)
	logger.Debug("Catch the bugs!")
	got := buf.String()
	want := "INFO  Hello world!                             foobar=123\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if cfg != (asklog.Config{Out: &buf}) {
		t.Fatalf("New must not modify the config, got %+v", cfg)
	}
}

func TestConfigUnknownFormat(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic")
		}
		if msg := fmt.Sprint(r); !strings.Contains(msg, `unknown log format "xml"`) {
			t.Fatalf("unexpected panic message: %q", msg)
		}
	}()
	cfg := asklog.Config{Out: io.Discard, Format: "xml"}
	cfg.New()
}

func TestLevelString(t *testing.T) {
	for _, tc := range []struct {
		lvl  slog.Level
		want string
	}{
		{log.LevelMaxVerbosity, "DEBUG" + strconv.Itoa(math.MinInt-int(slog.LevelDebug))},
		{log.LevelTrace - 1, "DEBUG-5"},
		{log.LevelTrace, "TRACE"},
		{log.LevelTrace + 1, "DEBUG-3"},
		{log.LevelDebug, "DEBUG"},
		{log.LevelDebug + 2, "DEBUG+2"},
		{log.LevelInfo, "INFO"},
		{log.LevelInfo + 2, "INFO+2"},
		{log.LevelWarn, "WARN"},
		{log.LevelWarn + 2, "WARN+2"},
		{log.LevelError, "ERROR"},
		{log.LevelError + 2, "ERROR+2"},
		{log.LevelCrit, "CRIT"},
		{log.LevelCrit + 4, "ERROR+8"},
		{math.MaxInt, "ERROR+" + strconv.Itoa(math.MaxInt-int(slog.LevelError))},
	} {
		lvl := asklog.Level(tc.lvl)
		got := lvl.String()
		if got != tc.want {
			t.Errorf("level %d: got %q, want %q", tc.lvl, got, tc.want)
		}
		// The help shows String as the default value, so Set must accept it.
		var again asklog.Level
		if err := again.Set(got); err != nil {
			t.Fatalf("set %q: %v", got, err)
		}
		if again != lvl {
			t.Errorf("level %d: round trip of %q got %d", tc.lvl, got, again)
		}
	}
}

func TestLevelSet(t *testing.T) {
	for _, tc := range []struct {
		name string
		want slog.Level
	}{
		{"trace", log.LevelTrace},
		{"debug", log.LevelDebug},
		{"info", log.LevelInfo},
		{"warn", log.LevelWarn},
		{"error", log.LevelError},
		{"crit", log.LevelCrit},
		{"Trace", log.LevelTrace},
		{"CRIT", log.LevelCrit},
		{"dbg", log.LevelDebug},
		{"wrn", log.LevelWarn},
		{"err", log.LevelError},
		{"debug-2", log.LevelDebug - 2},
		{"Info+2", log.LevelInfo + 2},
		// The slog.Level.String names of trace and crit.
		{"DEBUG-4", log.LevelTrace},
		{"ERROR+4", log.LevelCrit},
	} {
		var lvl asklog.Level
		if err := lvl.Set(tc.name); err != nil {
			t.Fatalf("set %q: %v", tc.name, err)
		}
		if lvl.Level() != tc.want {
			t.Errorf("set %q: got %d, want %d", tc.name, lvl, tc.want)
		}
	}
	for _, name := range []string{
		"", "verbose", "debug-", "info+", "info+x", "+2", "-4",
		"debug-" + strconv.FormatUint(math.MaxUint64, 10), // offset out of range
	} {
		lvl := asklog.Level(log.LevelWarn)
		err := lvl.Set(name)
		if err == nil {
			t.Fatalf("set %q: expected an error, got level %d", name, lvl)
		}
		if !strings.Contains(err.Error(), "unknown level") {
			t.Errorf("set %q: unexpected error: %v", name, err)
		}
		if lvl != asklog.Level(log.LevelWarn) {
			t.Errorf("set %q: the level changed to %d on error", name, lvl)
		}
	}
}

// verboseCmd logs everything by default: a level between the named ones.
type verboseCmd struct {
	LogConfig asklog.Config `ask:"."`
}

func (c *verboseCmd) Default() {
	c.LogConfig.Level = asklog.Level(log.LevelMaxVerbosity)
}

func (c *verboseCmd) Run(ctx context.Context) error {
	return nil
}

func TestLevelHelpDefault(t *testing.T) {
	err := ask.Run(context.Background(), &verboseCmd{}, []string{"--help"})
	if !errors.Is(err, ask.HelpErr) {
		t.Fatalf("expected help error, got: %v", err)
	}
	output := ask.UsageFromErr(err)
	name := asklog.Level(log.LevelMaxVerbosity).String()
	if !strings.Contains(output, "(default: "+name+")") {
		t.Fatalf("expected default %q in help output:\n%s", name, output)
	}
	// The default that the help shows is a valid flag value.
	cmd := &verboseCmd{}
	if err := ask.Run(context.Background(), cmd, []string{"--log.level", name}); err != nil {
		t.Fatal(err)
	}
	if got := cmd.LogConfig.Level.Level(); got != log.LevelMaxVerbosity {
		t.Fatalf("got level %d, want %d", got, log.LevelMaxVerbosity)
	}
}

// connWriter is a writer that, like *os.File, offers access to its file descriptor,
// and counts the attempts.
type connWriter struct {
	io.Writer
	calls int
}

func (w *connWriter) SyscallConn() (syscall.RawConn, error) {
	w.calls++
	return nil, errors.New("no file descriptor")
}

func TestConfigDefaultColor(t *testing.T) {
	out := &connWriter{Writer: io.Discard}
	cfg := asklog.Config{Out: out}
	cfg.Default()
	if out.calls != 1 {
		t.Fatalf("Default must check whether Out is a terminal, got %d checks", out.calls)
	}
	if cfg.Color {
		t.Fatal("expected no color: Out is not a terminal")
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = r.Close()
		_ = w.Close()
	})
	cfg = asklog.Config{Out: w}
	cfg.Default()
	if cfg.Color {
		t.Fatal("expected no color: a pipe is not a terminal")
	}

	cfg = asklog.Config{Out: new(bytes.Buffer)}
	cfg.Default()
	if cfg.Color {
		t.Fatal("expected no color: a buffer is not a terminal")
	}
}
