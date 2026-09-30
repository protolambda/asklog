# asklog

Log config / utils for CLI tool building with
[`github.com/protolambda/ask`](https://github.com/protolambda/ask).

This uses [`github.com/protolambda/proto-log`](https://github.com/protolambda/proto-log),
fully compatible with `slog`.

## Usage

Add an `asklog.Config` field with the `ask:"."` tag to an `ask` command to get the `--log.*` flags,
and call `New` to create the logger. `ask` only reads tagged fields: without the tag, for example in a
plain Go embedding (`struct{ asklog.Config }`), the command gets no `--log.*` flags.

```go
type MainCmd struct {
	LogConfig asklog.Config `ask:"."`
	// other flags ...
}

func (c *MainCmd) Run(ctx context.Context) error {
	logger := c.LogConfig.New()
	logger.Info("Hello world!")
	return nil
}
```

`ask` applies `Config.Default` (terminal format, info level, time on, color if `Out` is a terminal)
before it parses the flags. To log to another writer than `os.Stdout`, set `Out` in the command literal.
Only an `Out` with a file descriptor, such as an `*os.File`, is detected as a terminal:
to color the logs of a writer that wraps one, set `Color` in the command's own `Default`.

The zero `Config` is usable too: it logs info and higher levels to `os.Stdout`, in the terminal format,
without color, time or source info.

## Example

See [`asklog_example_test.go`](./asklog_example_test.go).

## License

MIT, see [`LICENSE`](./LICENSE) file.

