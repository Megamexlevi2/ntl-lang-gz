# CLI Reference

Complete reference for the `lunex` command-line tool and its built-in package manager.

---

## Global flags

| Flag               | Description                                              |
|--------------------|----------------------------------------------------------|
| `--debug`, `-d`    | Enable debug output (AST, IR, and runtime traces)        |
| `--verbose`, `-V`  | Verbose debug output (implies `--debug`)                 |
| `--no-cache`       | Skip both disk and memory caches; force a fresh compile  |
| `--version`        | Print version and exit                                   |
| `--help`           | Print usage and exit                                     |

---

## Commands

### `lunex run`

Run a Lunex source file or archive.

```
lunex run <file> [--emit ast|ir]
```

| Flag          | Description                                        |
|---------------|----------------------------------------------------|
| `--emit ast`  | Print the parsed AST as JSON instead of running    |
| `--emit ir`   | Print the IR as JSON instead of running            |

Supported file extensions:

| Extension | Description               |
|-----------|---------------------------|
| `.lx`     | Lunex source file |
| `.nax`    | Compiled archive  |

**Examples:**

```bash
lunex run main.lx
lunex run build/app.nax
lunex run --emit ast main.lx
```

---

### `lunex repl`

Start the interactive REPL (Read-Eval-Print Loop).

```
lunex repl
```

Launches a persistent session where you can type Lunex code and see the
result immediately. All defined names persist across inputs within the session.

**REPL commands:**

| Command          | Description                                                |
|------------------|------------------------------------------------------------|
| `.help`          | Show available REPL commands                               |
| `.exit` / `.quit`| Exit the REPL                                              |
| `.clear`         | Reset the session (clears all variables and definitions)   |
| `.vars`          | List all currently defined names                           |
| `.history`       | Show input history for this session                        |
| `.load <file>`   | Load and evaluate a `.lx` file into the session            |
| `.type <expr>`   | Show the inferred type of an expression                    |
| `Ctrl+D`         | Exit (EOF)                                                 |

**Multi-line input:** open a `{` block and press Enter — the REPL keeps reading
until all braces are closed.

**Example session:**

```
lunex » val io = @import("std.io")
lunex » fn greet(name) { "Hello, " + name + "!" }
lunex » fn main() {
.....   val x = 42
.....   io.log(x * 2)
.....   greet("world")
..... }
84
← "Hello, world!"
```

---

### `lunex -e`

Run a code snippet directly from the command line.

```
lunex -e "<code>"
```

**Example:**

```bash
lunex -e 'val io = @import("std.io"); fn main() { io.log("hello") }'
```

---

### `lunex build`

Compile a `.lx` source file or project entry to an archive.

```
lunex build [file] [-o <output>]
```

Without arguments, reads `config.lx` and compiles the entry point.

| Flag            | Description                                          |
|-----------------|------------------------------------------------------|
| `-o <file>`     | Output path (default: `<input>.nax`) |
| `--format nax`  | Output as a `.nax` archive          |

**Examples:**

```bash
lunex build main.lx -o dist/app.nax
lunex build src/math.lx -o dist/math.nax --format nax
```

---

### `lunex check`

Check a file for errors without running it.

```
lunex check <file>
```

Exits with code `0` and prints `ok` on success. Exits with code `1` and prints
error details on failure. Also runs semantic checks on the AST.

---

### `lunex see_errors`

Show detailed compile errors with full context.

```
lunex see_errors <file>
```

---

### `lunex dis`

Inspect a compiled `.nax` archive file.

```
lunex dis <file.nax>
```

Writes an annotated file alongside the input showing the archive contents.

---

### `lunex init`

Create a new Lunex project.

```
lunex init [name]
```

Creates `main.lx`, `config.lx`, and a `src/` directory in a new folder named
`name` (defaults to the current directory name).

---

### `lunex start`

Run the entry point defined in `config.lx`.

```
lunex start
```

Equivalent to `lunex run <entry>` where `<entry>` is the `main` or `entry`
field of `config.lx`.

---

### `lunex bench`

Run a file and print compile time and execution time.

```
lunex bench <file>
```

---

### Package Management

Package management is built into the `lunex` CLI and implemented in Go.

```bash
lunex install github.com/user/repo         # install from GitHub
lunex install github.com/user/repo@v1.2.3  # install a specific version
lunex remove <package>                     # remove a package
lunex update [package]                     # update one or all packages
lunex list                                 # list installed packages
```

Packages are stored in `~/.lunex/cache/` and resolved automatically when you use `@import("pkg-name")` in any `.lx` file.

---

### `lunex pack`

Bundle a directory of `.lx` files into a single `.nax` archive.

```
lunex pack <directory> [-o <output.nax>]
```

---

### `lunex unpack`

Extract a `.nax` archive to a directory.

```
lunex unpack <file.nax> [-o <directory>]
```

---

### `lunex version`

Print version information.

```
lunex version
```

Output includes the version number, build date, Go runtime version, operating
system, and architecture.

---

### `lunex platform`

Print platform and adapter diagnostics.

```
lunex platform
```

---

### `lunex runtimes`

List available execution engines (interpreter, archive loader).

```
lunex runtimes
```

---

## Cache management

```
lunex set cache <dir>     # set custom cache directory
lunex set cache reset     # restore default cache directory
lunex cache               # inspect on-disk archive cache
lunex cache clear         # clear on-disk cache
lunex memcache            # inspect in-process memory cache
lunex memcache clear      # clear memory cache
```

---

## Environment variables

| Variable         | Description                                              |
|------------------|----------------------------------------------------------|
| `LUNEX_HOME`     | Override the Lunex data directory (default: `~/.lunex`)  |
| `LUNEX_CACHE`    | Override the cache directory                             |
| `LUNEX_DEBUG`    | Set to `1` to enable debug output globally               |
| `LUNEX_VERBOSE`  | Set to `1` to enable verbose debug output                |
| `GOGC`           | Go GC percentage (Lunex sets `50` by default)            |
| `GOMEMLIMIT`     | Go memory limit (Lunex sets `200 MiB` by default)        |

---

## Exit codes

| Code | Meaning                              |
|------|--------------------------------------|
| `0`  | Success                              |
| `1`  | Compile or runtime error             |
| `2`  | Usage error (bad flag or missing argument) |
