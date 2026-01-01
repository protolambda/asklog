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
	logger.Trace("Trace everything!")
	logger.Debug("Catch the bugs!")
	// ...
	return nil
}

func ExampleConfig() {
	d, err := ask.Load(&MainCmd{})
	if err != nil {
		panic(err)
	}
	if _, err := d.Execute(context.Background(), &ask.ExecutionOptions{},
		"example-cmd", "--foobar=123",
		"--log.time=false", "--log.level=debug"); err != nil {
		panic(err)
	}
	// Output:
	// INFO  Hello world!                             foobar=123
	// DEBUG Catch the bugs!
}
