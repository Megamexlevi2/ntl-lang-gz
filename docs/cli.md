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

Without arguments, reads `lunex.toml` in the current directory and
compiles its declared entry point (fails with an error if no `lunex.toml`
is found).

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

Exits with code `0` when the source and its dependency graph pass checking. Exits
with code `1` when any error is found. The checker parses imported `.lx` and `.nax`
modules, resolves local and installed imports, indexes declarations and exports,
checks scopes and references, validates function arity, immutable assignments,
module members, export/import consistency, control-flow context, duplicate names,
top-level rules, import cycles, and the executable `main` entry point without
running user code.

Use `--debug` for checker phase tracing and `--verbose` for detailed resolver and
symbol diagnostics.

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
lunex init <template> <name>
```

Without a template, creates `lunex.toml`, `main.lx`, `src/math.lx` (an
example local module), and `.gitignore` in a new folder named `name`
(defaults to the current directory name).

With a template (`http_server`, `database`, or `website`), scaffolds a
project for that use case instead.

---

### `lunex start`

Run the entry point declared in `lunex.toml`.

```
lunex start
```

Equivalent to `lunex run <entry>`, where `<entry>` is the `entry` field
under `[project]` in `lunex.toml` (defaults to `main.lx`).

---

### `lunex debug`

Run a file with full compile diagnostics — every compile error, not just
the first — and, if compilation succeeds, run it with debug mode enabled
so each execution step and any runtime error prints with a full trace.

```
lunex debug <file>
```

Use this when `lunex run` doesn't give you enough detail to find a bug.

---

### `lunex bench`

Run a file and print compile time and execution time.

```
lunex bench <file>
```

---

### Package Management

Package management is built into the `lunex` CLI and implemented in Go,
backed by `lunex.toml` and `lunex.lock`. See [`../modulesys.md`](../modulesys.md)
for the full picture.

```bash
lunex install                              # install everything in lunex.toml (local store)
lunex install -g <url>[@version]           # install one library globally, no lunex.toml required
lunex install -l <url>[@version]           # install one library locally for this project only
lunex add <url>[@version]                  # add a [libraries.*] entry to lunex.toml and install it
lunex remove <library>                     # remove a library from both stores
lunex update [library]                     # re-resolve one or all installed libraries
lunex list                                 # list installed libraries, with scope (local/global)
```

`<url>` accepts a full `https://github.com/owner/repo` URL or the
`owner/repo` shorthand; both resolve to a `github`-source library.
Libraries installed this way are stored one directory per version —
`<name>@<version>` — under either store, so different projects can depend
on different versions of the same library without conflict.

---

### `lunex env`

Show the module system's current state: global and local store paths and
how many versions are installed in each, and whether `lunex.toml` /
`lunex.lock` exist in the current directory.

```
lunex env
```

---

### `lunex link`

Publish every command listed under `[project.bin]` in `lunex.toml` as a
global shim, so it can be run from anywhere on the system.

```
lunex link
```

Requires `lunex.toml` to exist in the current directory with at least one
`[project.bin]` entry.

---

### `lunex pack`

Bundle a directory of `.lx` files into a single `.nax` archive.

```
lunex pack <directory> [-o <output.nax>]
```

---

### `lunex unpack`

Extract a `.nax` archive into a new directory, named after the archive
file (e.g. `app.nax` extracts to `./app/`).

```
lunex unpack <file.nax>
```

There is no `-o` flag — the output directory name is always derived from
the input file.

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
lunex set cache <dir>     # set custom runtime-cache directory
lunex set cache reset     # restore default runtime-cache directory
lunex cache               # inspect on-disk runtime cache
lunex cache clear         # clear on-disk runtime cache
lunex memcache            # inspect in-process memory cache
lunex memcache clear      # clear memory cache
```

This is the runtime/adapter cache (compiled artifacts, embedded runtime
files), separate from the package module stores shown by `lunex env`.

---

## Environment variables

| Variable                | Description                                              |
|----------------------------|------------------------------------------------------------------|
| `LUNEX_DATA_DIR`           | Override the base Lunex data directory (default: `~/.lunex`)      |
| `LUNEX_RT_DIR`             | Override where the embedded runtime is extracted/cached           |
| `LUNEX_USE_CWD_CACHE`      | Set to `1` to use a cache directory relative to the current working directory instead of the home-based one |
| `NTL_DEBUG`                | Set automatically by `lunex debug`; set to `1` yourself to get the same verbose diagnostics from any command |
| `GOGC`                     | Go GC percentage (Lunex sets `50` by default if unset)            |
| `GOMEMLIMIT`               | Go memory limit (Lunex sets `200MiB` by default if unset)         |

---

## Exit codes

| Code | Meaning                              |
|------|--------------------------------------|
| `0`  | Success                              |
| `1`  | Compile or runtime error             |
| `2`  | Usage error (bad flag or missing argument) |
