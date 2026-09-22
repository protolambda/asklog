package asklog_test

import (
	"context"

	"github.com/protolambda/ask"
	"github.com/protolambda/asklog"
)

type MainCmd struct {
	// Inline the log flags. Ask applies asklog.Config.Default() automatically,
	// before the Default of MainCmd (if any), so MainCmd can override the log defaults.
	LogConfig asklog.Config `ask:"."`

	Foobar string `ask:"--foobar" help:"Some other flag"`
}

func (m *MainCmd) Run(ctx context.Context) error {
	logger := m.LogConfig.New()
	logger.Info("Hello world!", "foobar", m.Foobar)
	logger.Trace("Trace everything!")
	logger.Debug("Catch the bugs!")
	// ...
	return nil
}

func ExampleConfig() {
	err := ask.Run(context.Background(), &MainCmd{},
		[]string{"--foobar=123", "--log.time=false", "--log.level=debug"})
	if err != nil {
		panic(err)
	}
	// Output:
	// INFO  Hello world!                             foobar=123
	// DEBUG Catch the bugs!
}
