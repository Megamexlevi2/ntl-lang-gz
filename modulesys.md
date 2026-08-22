# Lunex Module System

Lunex resolves four kinds of imports: standard library modules, local file
imports, and libraries declared in `lunex.toml` and installed either
globally or per-project.

---

## Standard Library Modules

Always available, no installation and no `lunex.toml` entry required:

```lx
val io     = @import("std.io")
val http   = @import("std.http")
val fs     = @import("std.fs")
val crypto = @import("std.crypto")
val math   = @import("std.math")
val db     = @import("std.db")
val os     = @import("std.os")
val regex  = @import("std.regex")
val utils  = @import("std.utils")
val dt     = @import("std.datetime")
val env    = @import("std.env")
val ws     = @import("std.ws")
val jwt    = @import("std.jwt")
val json   = @import("std.json")
val buffer = @import("std.buffer")
val ints   = @import("std.ints")
val runtime = @import("runtime")
```

The complete list of standard library modules is documented in
`docs/stdlib.md`.

---

## Local File Imports (`.lx` and `.nax`)

```lx
val utils  = @fimport("./src/utils.lx")
val mylib  = @fimport("./mylib.nax")
val shared = @fimport("../shared/utils.nax")
```

A `.nax` file is a compiled Lunex archive produced by `lunex build`. It
bundles one or more `.lx` source files with their compiled bytecode into a
single binary archive:

```bash
lunex build math.lx -o math.nax
```

```lx
val io   = @import("std.io")
val math = @fimport("./math.nax")

fn main() {
  io.log(math.divide(50, 2))
  io.log(math.add(10, 5))
}
```

---

## Project Layout

```
my-project/
├── lunex.toml       # you edit this
├── lunex.lock       # Lunex writes this — exact resolved versions
├── main.lx
└── .lunex/
    └── modules/     # dependencies installed locally to this project
```

`lunex.toml` declares project metadata and dependencies. `lunex.lock` is
generated automatically by `lunex install`/`lunex add` — it pins the exact
version, source, and content hash of every installed library so a build
stays reproducible across machines.

### `lunex.toml`

```toml
[project]
name = "my-app"
version = "1.0.0"
description = "My Lunex application"
license = "MIT"
repository = "https://github.com/user/my-app"
entry = "main.lx"

[lunex]
min_version = "0.9.2"
max_version = "1.x"

# GitHub repository, default branch or a tag/range in `version`
[libraries.http_client]
url = "https://github.com/user/http-client"
version = "1.4.0"

# Always resolve to the newest tag
[libraries.logger]
url = "https://github.com/user/logger"
version = "latest"

# A specific GitHub release, downloaded as a release asset
[libraries.database]
source = "github-release"
url = "https://github.com/user/database"
release = "v2.1.0"

# Only a subdirectory of the repository
[libraries.ui]
source = "github"
url = "https://github.com/user/framework"
path = "modules/ui"
version = "0.5.0"

# A version range
[libraries.auth]
url = "https://github.com/user/auth"
version = ">=1.0.0 <2.0.0"

# A path on disk instead of a remote source
[libraries.test]
source = "local"
path = "./modules/test"
```

Standard library modules (`std.io`, `std.http`, `std.fs`, `std.crypto`,
`std.db`, `std.os`, `std.regex`, `std.utils`, `std.datetime`, `std.env`,
`std.ws`, `std.jwt`, `std.math`, `std.json`, `std.buffer`, `std.ints`, and
`runtime`) never need a `[libraries.*]` entry.

### `lunex.lock`

```toml
[modules.logger]
version = "1.3.2"
hash = "sha256:..."
source = "github:user/logger"
url = "https://github.com/user/logger"
# commit = "..."   # present only when the resolved install is pinned to a commit
```

Don't edit `lunex.lock` by hand — it's regenerated on every install.

---

## Global vs. Local Installs

Every installed version is kept isolated on disk under its own
`<name>@<version>` directory, so two projects — or two dependencies of the
same project — can each depend on a different version of the same library
without conflict.

| Store  | Location            | Scope                                |
|--------|----------------------|---------------------------------------|
| Global | `~/.lunex/modules`   | shared across every project on the machine |
| Local  | `./.lunex/modules`   | this project only                     |

Resolution checks the local store first, then the global store.

```bash
lunex install -g https://github.com/user/logger         # global, latest
lunex install -g https://github.com/user/logger@1.3.2   # global, pinned
lunex install -l https://github.com/user/logger         # local to this project only
```

`-g`/`-l` installs work without a `lunex.toml` at all — useful for a quick
one-off script. `lunex install -l` also records the library in
`lunex.toml` if one exists in the current directory.

### Installing everything a project declares

```bash
lunex install
```

Reads every `[libraries.*]` entry in `lunex.toml`, installs each one into
the local store, and writes `lunex.lock`.

### Adding a new dependency

```bash
lunex add https://github.com/user/repo            # latest
lunex add https://github.com/user/repo@v1.2.3      # specific version
```

Adds a `[libraries.*]` entry to `lunex.toml` and installs it locally in
the same step.

### Managing installed libraries

```bash
lunex list              # installed libraries, with scope (local/global)
lunex remove logger      # remove a library from both stores
lunex update             # re-resolve every library against lunex.toml
lunex update logger      # re-resolve one library
```

---

## Importing Libraries

```lx
val xml = @import("lune-xml")
xml.parse("<root/>")
```

`@import("name")` is resolved in this order:

1. Standard library (always wins, can't be shadowed)
2. Local store (`./.lunex/modules`), matching the version pinned in
   `lunex.lock` if one is present
3. Global store (`~/.lunex/modules`)

If nothing resolves, Lunex prints:

```
hint: library "pkg-name" not found — add it to lunex.toml with:
  lunex add https://github.com/<owner>/pkg-name
  lunex install
```

---

## `.nax` File Format

A `.nax` file is a custom binary format — not a zip or tar archive, and not
readable by standard archive tools. Only the Lunex runtime can read it. It
stores compiled bytecode chunks, embedded source text for debugging, and a
module export table.

```bash
lunex run mylib.nax
```

---

## Running and Debugging a Project

```bash
lunex start              # run the entry point declared in lunex.toml
lunex debug main.lx      # show every compile error, then run with a full trace
```

`lunex debug` compiles with complete diagnostics (not just the first
error) and, if compilation succeeds, runs the file with debug mode
enabled so every execution step and any runtime error is printed with a
full trace — useful when `lunex run` gives you too little detail to find
a bug.

---

## Executable Commands (`bin`)

`lunex.toml` can declare `bin`, the same idea as `"bin"` in `package.json`:

```toml
[project]
bin = "./cli.lx"
```

for a single command named after the project, or a table for multiple
named commands:

```toml
[project.bin]
build = "./bin/build.lx"
serve = "./bin/serve.lx"
```

When a library that declares `bin` is installed — via `lunex install`,
`lunex install -g/-l`, or as a `lunex.toml` dependency — Lunex writes an
executable shim per command into the bin directory of whichever store it
was installed into:

| Store  | Bin directory     |
|--------|--------------------|
| Global | `~/.lunex/bin`     |
| Local  | `./.lunex/bin`     |

Each shim is a small shell script that runs `lunex run <entry>`. Add
`~/.lunex/bin` to your `PATH` to run globally installed commands directly:

```bash
export PATH="$HOME/.lunex/bin:$PATH"
```

### Developing a command locally

```bash
lunex link
```

Reads `lunex.toml` in the current directory and links its `[project.bin]`
commands into `~/.lunex/bin` immediately, pointing at your working
directory — the same idea as `npm link`. No install step, and edits to
your source take effect the next time the command runs.

Removing a library (`lunex remove <name>`) also removes any command
shims it registered.

---

## Example: End-to-End Workflow

```bash
lunex init my-app
cd my-app
lunex add https://github.com/Megamexlevi2/lune-xml
lunex install
```

```lx
val io  = @import("std.io")
val xml = @import("lune-xml")

fn main() {
  val doc = xml.parse("<greet>Hello, Lunex!</greet>")
  io.log(doc.root.text)
}
```

```bash
lunex start
```
