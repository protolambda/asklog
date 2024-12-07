package asklog_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/protolambda/ask"
)

func TestMainCmdRun(t *testing.T) {
	var buf bytes.Buffer
	mainCmd := &MainCmd{}
	mainCmd.LogConfig.Out = &buf
	descr, err := ask.Load(mainCmd)
	if err != nil {
		t.Fatal(err)
	}
	_, err = descr.Execute(context.Background(), &ask.ExecutionOptions{},
		"--log.level", "info",
		"--log.format", "terminal",
		"--log.color", "true",
		"--foobar", "123")
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

func TestMainCmdHelp(t *testing.T) {
	descr, err := ask.Load(&MainCmd{})
	if err != nil {
		t.Fatal(err)
	}
	output := descr.Usage(false)
	t.Log("help output:", output)
	if !strings.Contains(output, "log.level") {
		t.Fatal("expected log level help option")
	}
}
