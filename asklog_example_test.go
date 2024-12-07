package asklog_test

import (
	"context"

	"github.com/protolambda/ask"
	"github.com/protolambda/asklog"
)

type MainCmd struct {
	LogConfig asklog.Config `ask:"."`

	Foobar string `ask:"--foobar" help:"Some other flag"`
}

func (m *MainCmd) Default() {
	m.LogConfig.Default()
}

func (m *MainCmd) Run(ctx context.Context, args ...string) error {
	logger := m.LogConfig.New()
	logger.Info("Hello world!", "foobar", m.Foobar)
	// ...
	return nil
}

func Example() {
	ask.Run(&MainCmd{})
}
