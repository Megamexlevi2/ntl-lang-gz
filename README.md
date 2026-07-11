Lunex

A fast, stable scripting language for backend development.

Lunex is a statically scoped scripting language built in Go.
It is designed to be readable, practical, and consistent for everyday backend work. The language includes a built-in standard library with support for HTTP, file system access, cryptography, databases, WebSockets, and more.

It runs on Linux, macOS, Windows, and Android (Termux).

Two ready-made modules written in Lunex are included: lune-xml and lunex-cli.

---

## Installation

### Pre-built binary

Download the binary for your platform from the
[releases page](https://github.com/Megamexlevi2/lunex-language/releases).

### Build from source

Requires Go 1.23 or later.

```bash
git clone https://github.com/Megamexlevi2/lunex-language
cd lunex-language
./build.sh
```

#### Using Make

```bash
git clone https://github.com/Megamexlevi2/lunex-language
cd lunex-language
make
```

#### Using CMake

```bash
git clone https://github.com/Megamexlevi2/lunex-language
cd lunex-language
cmake -B build
cmake --build build
```


---

## Quick Start

```bash
cat << 'EOF' > hello.lx
val io = @import("std.io")

fn main() {
  io.log("Hello, World!")
}
EOF

lunex run hello.lx
```

---

## Language at a Glance

### Variables

`val` is immutable. `var` is mutable.

```lx
val name   = "Lunex"
val pi     = 3.14159
val active = true

var counter = 0
counter = counter + 1
```

Destructuring works on objects and arrays:

```lx
val { name, role } = user
val [first, second] = items
```

### Functions

The last expression in a function body is its return value.
Lunex has no `return` keyword.

```lx
fn add(a, b) {
  a + b          // returned automatically
}

val square = fn(x) { x * x }

io.log(add(2, 3))    // 5
io.log(square(5))    // 25
```

### Structs

No `class` keyword. Factory functions return a `struct`:

```lx
fn Animal(name, sound) {
  val self = struct {
    name  = name
    sound = sound

    fn speak() {
      self.name + " says " + self.sound
    }
  }
  self
}

val cat = Animal("Cat", "Meow")
io.log(cat.speak())   // Cat says Meow
```

### Control Flow

```lx
if n < 0 {
  "negative"
} else if n == 0 {
  "zero"
} else {
  "positive"
}
```

`guard` runs its `else` block when the condition is **false**:

```lx
guard user != null else {
  io.err("no user provided")
}
// execution continues here
```

`unless` runs its block when the condition is **false**:

```lx
unless connected {
  io.warn("not connected — retrying")
}
```

`match` tests exact values — top-to-bottom, first match wins:

```lx
val label = match status {
  "ok"      => "success"
  "pending" => "waiting"
  "fail"    => "error"
  _         => "unknown"
}
```

### Loops

```lx
var i = 0
while i < 10 {
  io.log(i)
  i = i + 1
}

each name in ["Alice", "Bob", "Carol"] {
  io.log("Hello, " + name + "!")
}
```

### Native Array and String Methods

Arrays and strings have built-in methods — no import needed:

```lx
val nums = [3, 1, 4, 1, 5]

nums.sort()                              // [1, 1, 3, 4, 5]
nums.map(fn(x) { x * 2 })              // [6, 2, 8, 2, 10]
nums.filter(fn(x) { x > 2 })           // [3, 4, 5]
nums.reduce(fn(acc, x) { acc + x }, 0) // 14
nums.includes(4)                         // true
nums.length                              // 5

"lunex".toUpperCase()                    // "LUNEX"
"  hello  ".trim()                       // "hello"
"lunex".startsWith("lun")               // true
```

### Concurrency

```lx
val ch = channel()

spawn fn() {
  ch.send(computeSomething())
}()

val result = ch.recv()
io.log(result)
```

### Defer

Schedules a block to run when the enclosing function exits:

```lx
fn process(path) {
  val fs = @import("std.fs")
  defer { io.log("finished:", path) }
  fs.readFile(path)
}
```

---

## CLI Reference

```
lunex run <file> [--emit ast|ir]   run a .lx source file or a .nax archive
lunex -e "<code>"                  run a code snippet directly
lunex build [file] [-o output]     compile source or create a .nax archive
lunex check <file>                 check for errors without running
lunex dis <file.nax>                inspect a compiled archive
lunex bench <file>                 run with timing output
lunex init [name]                  create a new project
lunex start                        run project entry from config.lx
lunex pack <dir>                   bundle a directory to .nax archive
lunex unpack <file.nax>            extract a .nax archive
lunex platform                     show platform diagnostics
lunex runtimes                     list available execution engines
lunex version                      print version
lunex help                         show full usage
```

> Package management is built into Lunex and implemented in Go.

---

## Package Management

Lunex includes a Go-based package manager in the CLI.

```bash
lunex install github.com/user/repo         # install from GitHub
lunex install github.com/user/repo@v1.2.0  # install a specific version
lunex remove <package>                     # remove a package
lunex update [package]                     # update one or all packages
lunex list                                 # list installed packages
```

Packages are stored in `~/.lunex/cache/` and resolved automatically when you use `@import("pkg-name")` in any `.lx` file.

---

## Module System

```lx
val io     = @import("std.io")
val http   = @import("std.http")
val crypto = @import("std.crypto")
```

Import a local source file or compiled archive:

```lx
val lib = @fimport("./src/utils.lx")
val pkg = @fimport("./dist/math.nax")
```

Import an external package installed by Lunex:

```lx
val xml = @import("lune-xml")   // after: lunex install https://github.com/Megamexlevi2/lunex-language/lune-xml
```

---

## Standard Library

| Module         | Purpose                                                  |
|----------------|----------------------------------------------------------|
| `std.io`       | Console output, input, colors, tables, spinner           |
| `std.fs`       | File system: read, write, list, stat                     |
| `std.http`     | HTTP client and server                                   |
| `std.crypto`   | Hashing, encoding, encryption, passwords, UUIDs          |
| `std.db`       | Built-in in-memory database                              |
| `std.ws`       | WebSocket server and client                              |
| `std.jwt`      | JSON Web Token sign and verify                           |
| `std.math`     | Math functions and constants                             |
| `std.datetime` | Date, time, formatting, arithmetic                       |
| `std.os`       | Process, environment variables, shell execution          |
| `std.regex`    | Pattern matching and replacement (RE2 syntax)            |
| `std.env`      | Environment variable access                              |
| `std.utils`    | Array, object, string, and functional helpers            |

---

## Examples

Check out [`examples/`](examples/) for runnable code covering everything from the basics to the more advanced stuff:

- Hello World and basic I/O
- Variables, destructuring, template strings
- Structs and factory functions
- Control flow: `if`, `while`, `each`, `guard`, `unless`, `match`, `defer`
- Standard library: math, crypto, fs, datetime, regex, os, http
- Higher-order functions: map, filter, reduce, compose, memoize
- Concurrent workers with `spawn` and `channel`
- WebSockets, HTTP servers, and REST APIs

---

## License

[Mozilla Public License Version 2.0](LICENSE)

© 2026 David Dev · [github.com/Megamexlevi2](https://github.com/Megamexlevi2)