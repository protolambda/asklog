package asklog_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/protolambda/ask"
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
}
