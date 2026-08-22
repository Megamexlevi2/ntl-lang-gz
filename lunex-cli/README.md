# lunex-cli

`lunex-cli` is a compact command-line helper module for Lunex applications.
It provides command registration, aliases, parsing, help rendering, default
commands, and a small args helper for handlers.

## Example

```lx
val cli = @fimport("./lunex-cli/main.lx")
val io = @import("std.io")

fn setupCli() {
  cli.name("demo")
  cli.version("1.0.0")
  cli.description("Demo CLI module")
  cli.usage("demo <command> [options]")
  cli.footer("Use 'demo help' for more information.")
  cli.defaultCommand("help")

  cli.command("help", fn(args) {
    io.log("Available commands: help, hello, sum")
  }, "Show help")

  cli.command("hello", fn(args) {
    io.log("Hello", args.get("name"))
  }, "Say hello", ["hi"])

  cli.command("sum", fn(args) {
    val a = args.get("a")
    val b = args.get("b")
    io.log("Result:", a + b)
  }, "Add two values", ["add"])
}

fn main() {
  setupCli()
  cli.run()
}
```

## API

- `cli.name(text?)`
- `cli.version(text?)`
- `cli.description(text?)`
- `cli.usage(text?)`
- `cli.footer(text?)`
- `cli.defaultCommand(name?)`
- `cli.reset()`
- `cli.command(name, handler, description?, aliases?)`
- `cli.alias(commandName, aliasName)`
- `cli.has(name)`
- `cli.find(name)`
- `cli.commands()`
- `cli.commandCount()`
- `cli.parse(argv?)`
- `cli.help()`
- `cli.run(argv?)`

## Parsed args

Inside a command handler, the `args` object exposes:

- `args.command()`
- `args.get(name)`
- `args.has(name)`
- `args.flag(name)`
- `args.string(name)`
- `args.number(name)`
- `args.int(name)`
- `args.arg(index)`
- `args.count()`
- `args.rest()`
- `args.helpRequested()`
- `args.versionRequested()`
- `args.toObject()`

## Tests

Run the suite with:

```bash
lunex run lunex-cli/test/run_all.lx
```

## Package contents

- `src/core.lx`: main implementation
- `main.lx`: public exports
- `examples/basic_cli.lx`: standalone example
- `test/*.lx`: test suites