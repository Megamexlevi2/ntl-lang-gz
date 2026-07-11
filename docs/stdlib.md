# Standard Library Reference

Complete API reference for all built-in modules in Lunex v0.9.0.

All modules are embedded in the Lunex binary — no installation required.
Import any module with `@import("std.<name>")`.

---

## `std.io` — Console I/O

```lx
val io = @import("std.io")
```

### Output

| Function                   | Description                                        |
|----------------------------|----------------------------------------------------|
| `io.log(...args)`          | Print to stdout, space-separated                   |
| `io.err(...args)`          | Print to stderr with a red `[ERROR]` prefix        |
| `io.warn(...args)`         | Print to stdout with a yellow `[WARN]` prefix      |
| `io.info(...args)`         | Print to stdout with a cyan `[INFO]` prefix        |
| `io.success(...args)`      | Print to stdout with a green `✔` prefix            |
| `io.write(s)`              | Write a string to stdout with no trailing newline  |
| `io.newline()`             | Print a blank line                                 |
| `io.table(rows)`           | Render an array of structs/objects as a table      |
| `io.json(val)`             | Pretty-print any value as formatted JSON           |
| `io.hr(char?, len?)`       | Print a horizontal rule                            |
| `io.banner(text)`          | Print a highlighted banner                         |
| `io.clear()`               | Clear the terminal screen                          |

### Spinner

`io.spinner(label)` starts an animated spinner and returns a control object:

```lx
val sp = io.spinner("Loading data...")
// ... do work ...
sp.tick()   // finish with a ✓
sp.stop()   // stop without marking done
```

### Progress bar

```lx
io.progress(current, total, label?)
```

Renders a progress bar for `current` out of `total` steps.

### Input

| Function                  | Returns  | Description                          |
|---------------------------|----------|--------------------------------------|
| `io.read(prompt?)`        | `string` | Read a line from stdin               |
| `io.readLine(prompt?)`    | `string` | Alias for `io.read`                  |
| `io.readInt(prompt?)`     | `number` | Read a line and parse it as integer  |

### Formatting

```lx
io.format("Hello, {}! You are {} years old.", name, age)
```

### Colors

These functions return a colorized string (no side effects — you can nest them):

```lx
io.red(s)      io.green(s)    io.yellow(s)
io.blue(s)     io.magenta(s)  io.cyan(s)
io.white(s)    io.gray(s)     io.bold(s)
io.dim(s)      io.italic(s)
io.color(s, "red")   // named color
io.strip(s)          // remove ANSI codes from a string
```

```lx
io.log(io.green("✓"), "Build passed")
io.log(io.bold("=== Report ==="))
io.log(io.red("error:"), "something went wrong")
```

### Terminal detection

```lx
io.isTerminal()   // true if stdout is an interactive terminal
```

---

## `std.math` — Mathematics

```lx
val math = @import("std.math")
```

### Constants

| Name      | Value               |
|-----------|---------------------|
| `math.PI` | 3.141592653589793   |
| `math.E`  | 2.718281828459045   |

### Basic functions

| Function              | Description                            |
|-----------------------|----------------------------------------|
| `math.abs(x)`         | Absolute value                         |
| `math.ceil(x)`        | Round up to nearest integer            |
| `math.floor(x)`       | Round down to nearest integer          |
| `math.round(x)`       | Round to nearest integer               |
| `math.trunc(x)`       | Truncate fractional part               |
| `math.sign(x)`        | −1, 0, or 1                            |
| `math.sqrt(x)`        | Square root                            |
| `math.cbrt(x)`        | Cube root                              |
| `math.pow(x, y)`      | x to the power y                       |
| `math.hypot(a, b)`    | sqrt(a² + b²)                          |
| `math.min(a, b, ...)` | Smallest of all arguments              |
| `math.max(a, b, ...)` | Largest of all arguments               |
| `math.clamp(v, lo, hi)` | Clamp v to [lo, hi]                  |
| `math.lerp(a, b, t)`  | Linear interpolation between a and b  |

### Exponentials and logarithms

| Function        | Description             |
|-----------------|-------------------------|
| `math.exp(x)`   | e^x                     |
| `math.exp2(x)`  | 2^x                     |
| `math.log(x)`   | Natural logarithm       |
| `math.log2(x)`  | Base-2 logarithm        |
| `math.log10(x)` | Base-10 logarithm       |

### Trigonometry

| Function            | Description                    |
|---------------------|--------------------------------|
| `math.sin(x)`       | Sine (radians)                 |
| `math.cos(x)`       | Cosine (radians)               |
| `math.tan(x)`       | Tangent (radians)              |
| `math.asin(x)`      | Arcsine                        |
| `math.acos(x)`      | Arccosine                      |
| `math.atan(x)`      | Arctangent                     |
| `math.atan2(y, x)`  | Two-argument arctangent        |
| `math.sinh(x)`      | Hyperbolic sine                |
| `math.cosh(x)`      | Hyperbolic cosine              |
| `math.tanh(x)`      | Hyperbolic tangent             |
| `math.degToRad(d)`  | Degrees to radians             |
| `math.radToDeg(r)`  | Radians to degrees             |

### Random

| Function                   | Description                          |
|----------------------------|--------------------------------------|
| `math.random()`            | Uniform float in [0, 1)              |
| `math.randomInt(min, max)` | Random integer in [min, max]         |
| `math.seed(n)`             | Seed the random number generator     |

### Number theory

| Function                   | Description                           |
|----------------------------|---------------------------------------|
| `math.gcd(a, b)`           | Greatest common divisor               |
| `math.lcm(a, b)`           | Least common multiple                 |
| `math.isPrime(n)`          | True if n is prime                    |
| `math.factorial(n)`        | n!                                    |
| `math.combinations(n, k)`  | Binomial coefficient C(n, k)         |
| `math.permutations(n, k)`  | P(n, k)                              |

### Type checks

| Function          | Description                        |
|-------------------|------------------------------------|
| `math.isNaN(v)`   | True if the value is NaN           |
| `math.isFinite(v)`| True if the value is finite        |

---

## `std.json` — JSON serialization

```lx
val json = @import("std.json")
```

### Parsing and formatting

| Function                         | Description                                     |
|----------------------------------|-------------------------------------------------|
| `json.parse(text)`               | Parse a JSON string into Lunex values           |
| `json.stringify(value)`          | Pretty-print JSON with 2-space indentation      |
| `json.stringify(value, indent)`  | Pretty-print JSON using `indent` spaces         |
| `json.pretty(value)`             | Alias for `json.stringify(value)`               |
| `json.compact(value)`            | Minified JSON output (no whitespace)            |
| `json.isValid(text)`             | True if the text is valid JSON                  |
| `json.toJSON(value)`             | Alias for `json.stringify(value)`               |
| `json.fromJSON(text)`            | Alias for `json.parse(text)`                    |
| `json.writeFile(path, value)`    | Save pretty JSON to a file                      |
| `json.writeFile(path, value, indent)` | Save JSON with custom indentation          |
| `json.save(path, value)`         | Alias for `json.writeFile(path, value)`         |

> `json.stringify` skips function fields and undefined values in objects, and
> writes array holes as `null`, keeping output readable and consistent.

---

## `std.utils` — Utilities

```lx
val utils = @import("std.utils")
```

> Common array and string operations (`map`, `filter`, `reduce`, `sort`,
> `push`, `includes`, etc.) are available as **native methods** directly on
> the value — no import needed. See the language reference for the full list.
> `std.utils` provides higher-level helpers that go beyond native methods.

### Array helpers

| Function                         | Description                                    |
|----------------------------------|------------------------------------------------|
| `utils.range(n)`                 | Array `[0, 1, …, n−1]`                        |
| `utils.range(start, end)`        | Array `[start, …, end−1]`                     |
| `utils.chunk(arr, n)`            | Split into chunks of size n                    |
| `utils.flatten(arr)`             | Flatten one level of nesting                   |
| `utils.flatMap(arr, fn)`         | Map then flatten one level                     |
| `utils.zip(a, b)`                | Pair elements: `[[a0,b0], [a1,b1], …]`        |
| `utils.unzip(pairs)`             | Inverse of zip: returns `[keys, values]`       |
| `utils.intersection(a, b)`       | Elements present in both arrays                |
| `utils.difference(a, b)`         | Elements in a not in b                         |
| `utils.union(a, b)`              | All unique elements from both arrays           |
| `utils.uniq(arr)`                | Remove duplicate values                        |
| `utils.uniqBy(arr, fn)`          | Remove duplicates by key function              |
| `utils.shuffle(arr)`             | Return a randomly shuffled copy                |
| `utils.sample(arr)`              | Pick one random element                        |
| `utils.sampleSize(arr, n)`       | Pick n random elements                         |
| `utils.sortBy(arr, fn)`          | Sort by a key function                         |
| `utils.sortBy(arr, fn, "desc")`  | Sort descending by a key function              |
| `utils.groupBy(arr, fn)`         | Group elements into an object by key           |
| `utils.countBy(arr, fn)`         | Count elements per group                       |
| `utils.partition(arr, fn)`       | Split into `[pass, fail]` by predicate         |

### Numeric helpers

| Function                  | Description                                |
|---------------------------|--------------------------------------------|
| `utils.sum(arr)`          | Sum of a numeric array                     |
| `utils.mean(arr)`         | Arithmetic mean                            |
| `utils.median(arr)`       | Median value                               |
| `utils.min(arr)`          | Minimum value                              |
| `utils.max(arr)`          | Maximum value                              |
| `utils.clamp(v, lo, hi)`  | Clamp v to [lo, hi]                        |
| `utils.lerp(a, b, t)`     | Linear interpolation                       |
| `utils.random(min?, max?)`| Random float in [min, max)                 |
| `utils.randInt(min, max)` | Random integer in [min, max]               |
| `utils.formatNumber(n)`   | Format with thousands separator            |
| `utils.formatBytes(n)`    | Human-readable byte size (KB, MB, …)       |

### Object helpers

| Function                      | Description                                    |
|-------------------------------|------------------------------------------------|
| `utils.keys(obj)`             | Array of own keys                              |
| `utils.values(obj)`           | Array of own values                            |
| `utils.entries(obj)`          | Array of `[key, value]` pairs                  |
| `utils.fromEntries(pairs)`    | Build an object from `[key, value]` pairs      |
| `utils.has(obj, key)`         | True if key exists on the object               |
| `utils.pick(obj, keys)`       | New object with only the specified keys        |
| `utils.omit(obj, keys)`       | New object without the specified keys          |
| `utils.merge(a, b)`           | Merge b into a (shallow, returns new object)   |
| `utils.assign(target, source)`| Copy source properties into target             |
| `utils.invert(obj)`           | Swap keys and values                           |
| `utils.mapValues(obj, fn)`    | Transform each value with fn                   |

### String helpers

| Function                  | Description                                     |
|---------------------------|-------------------------------------------------|
| `utils.camelCase(s)`      | `"hello world"` → `"helloWorld"`               |
| `utils.snakeCase(s)`      | `"helloWorld"` → `"hello_world"`               |
| `utils.kebabCase(s)`      | `"helloWorld"` → `"hello-world"`               |
| `utils.titleCase(s)`      | `"hello world"` → `"Hello World"`              |
| `utils.slugify(s)`        | `"Hello, World!"` → `"hello-world"`            |
| `utils.truncate(s, n)`    | Truncate to n chars with `…` suffix             |
| `utils.pad(s, n, char?)`  | Pad to width n (centered)                       |
| `utils.padStart(s, n, char?)` | Pad to width n on the left                  |
| `utils.padEnd(s, n, char?)`   | Pad to width n on the right                 |
| `utils.repeat(s, n)`      | Repeat string n times                           |
| `utils.template(s, obj)`  | Fill `{{key}}` placeholders from obj            |

### Functional helpers

| Function              | Description                                             |
|-----------------------|---------------------------------------------------------|
| `utils.pipe(fns)`     | Return a function that passes input through each fn     |
| `utils.compose(fns)`  | Like pipe but in reverse order                          |
| `utils.memoize(fn)`   | Return a version of fn with cached results              |
| `utils.once(fn)`      | Return a version of fn that only runs once              |
| `utils.negate(fn)`    | Return a function that inverts the boolean result       |
| `utils.times(n, fn)`  | Call fn n times with index, return results array        |

### Identity and time

| Function              | Description                                     |
|-----------------------|-------------------------------------------------|
| `utils.uuid()`        | Generate a RFC-4122 UUID v4                     |
| `utils.now()`         | Current Unix timestamp in milliseconds          |
| `utils.timestamp()`   | Alias for `utils.now()`                         |
| `utils.sleep(ms)`     | Pause execution for ms milliseconds             |
| `utils.noop()`        | A function that does nothing                    |
| `utils.identity(x)`   | A function that returns its argument            |

---

## `std.datetime` — Date and time

```lx
val datetime = @import("std.datetime")
```

### Creating datetime values

| Function                           | Returns   | Description                          |
|------------------------------------|-----------|--------------------------------------|
| `datetime.now()`                   | datetime  | Current local date-time              |
| `datetime.utcNow()`                | datetime  | Current UTC date-time                |
| `datetime.fromTimestamp(ms)`       | datetime  | Parse a Unix timestamp (ms)          |
| `datetime.fromTimestamp(ms, "s")`  | datetime  | Parse a Unix timestamp in seconds    |
| `datetime.parse(s)`                | datetime  | Parse an ISO 8601 string             |

A datetime value has these readable fields:

```lx
val now = datetime.now()
io.log(now.iso)        // "2026-06-25T14:30:00Z"
io.log(now.year)       // 2026
io.log(now.month)      // 6
io.log(now.day)        // 25
io.log(now.unix)       // Unix timestamp in seconds
io.log(now.timestamp)  // Unix timestamp in milliseconds
```

### Formatting

```lx
io.log(datetime.format(now, "YYYY-MM-DD HH:mm:ss"))
```

**Layout tokens:** `YYYY` `MM` `DD` `HH` `mm` `ss` `Z` `MMM`

### Converting

| Function                          | Returns | Description                          |
|-----------------------------------|---------|--------------------------------------|
| `datetime.toTimestamp(dt)`        | number  | Milliseconds since Unix epoch        |
| `datetime.toTimestamp(dt, "s")`   | number  | Seconds since Unix epoch             |
| `datetime.format(dt, layout)`     | string  | Format using layout tokens           |

### Arithmetic

| Function                      | Returns  | Description                       |
|-------------------------------|----------|-----------------------------------|
| `datetime.add(dt, n, unit?)`  | datetime | Add n units (default: `"ms"`)     |
| `datetime.subtract(dt, n, unit?)` | datetime | Subtract n units              |
| `datetime.diff(a, b, unit?)`  | number   | Difference from a to b (default: `"ms"`) |

**Unit values:** `"ms"` `"s"` `"m"` `"h"` `"d"` `"w"` `"month"` `"year"`

### Comparison

| Function                    | Returns | Description                          |
|-----------------------------|---------|--------------------------------------|
| `datetime.isBefore(a, b)`   | boolean | True if a is before b                |
| `datetime.isAfter(a, b)`    | boolean | True if a is after b                 |
| `datetime.isEqual(a, b)`    | boolean | True if a and b are the same instant |
| `datetime.compare(a, b)`    | number  | −1, 0, or 1                          |

### Inspection

| Function                  | Returns | Description                                  |
|---------------------------|---------|----------------------------------------------|
| `datetime.weekday(dt)`    | number  | Day of week (0=Sunday … 6=Saturday)          |
| `datetime.weekdayName(dt)`| string  | `"Monday"`, `"Tuesday"`, etc.                |
| `datetime.monthName(dt)`  | string  | `"January"`, `"February"`, etc.              |
| `datetime.dayOfYear(dt)`  | number  | Day number within the year (1–366)           |
| `datetime.weekOfYear(dt)` | number  | ISO week number (1–53)                       |
| `datetime.daysInMonth(dt)`| number  | Days in the month of dt                      |
| `datetime.isLeapYear(dt)` | boolean | True if the year is a leap year              |
| `datetime.isWeekend(dt)`  | boolean | True if Saturday or Sunday                   |
| `datetime.isValid(dt)`    | boolean | True if the value is a valid datetime        |

### Rounding

| Function                      | Returns  | Description                        |
|-------------------------------|----------|------------------------------------|
| `datetime.startOf(dt, unit)`  | datetime | Start of the given unit            |
| `datetime.endOf(dt, unit)`    | datetime | End of the given unit              |

### Other

| Function                | Description                                        |
|-------------------------|----------------------------------------------------|
| `datetime.timezone(tz)` | Get a datetime helper scoped to a timezone         |
| `datetime.sleep(ms)`    | Pause execution for ms milliseconds                |

---

## `std.crypto` — Cryptography

```lx
val crypto = @import("std.crypto")
```

### Hashing

| Function                    | Description                                              |
|-----------------------------|----------------------------------------------------------|
| `crypto.sha256(s)`          | SHA-256 hex digest                                       |
| `crypto.sha512(s)`          | SHA-512 hex digest                                       |
| `crypto.sha1(s)`            | SHA-1 hex digest                                         |
| `crypto.md5(s)`             | MD5 hex digest                                           |
| `crypto.hash(algo, s)`      | Hash with named algorithm: `"sha256"`, `"sha512"`, etc. |
| `crypto.hmac(algo, key, data)` | HMAC hex digest with the given algorithm             |

```lx
val hash = crypto.sha256("hello")
val hmac = crypto.hmac("sha256", "my-secret-key", "Hello, Lunex!")
```

### Encoding

| Function                    | Description                            |
|-----------------------------|----------------------------------------|
| `crypto.base64Encode(s)`    | Standard Base64 encode                 |
| `crypto.base64Decode(s)`    | Standard Base64 decode                 |
| `crypto.base64UrlEncode(s)` | URL-safe Base64 encode (no padding)    |
| `crypto.base64UrlDecode(s)` | URL-safe Base64 decode                 |
| `crypto.toHex(s)`           | Convert bytes to hex string            |
| `crypto.fromHex(s)`         | Convert hex string to bytes            |

### Symmetric encryption

| Function                         | Description                                     |
|----------------------------------|-------------------------------------------------|
| `crypto.encrypt(plaintext, key)` | AES-256 encrypt; returns base64 ciphertext      |
| `crypto.decrypt(ciphertext, key)`| AES-256 decrypt; returns plaintext              |

```lx
val key        = "my-32-char-key-here-padding-ok!!"
val ciphertext = crypto.encrypt("top secret message", key)
val plaintext  = crypto.decrypt(ciphertext, key)
```

### Random values

| Function             | Description                                     |
|----------------------|-------------------------------------------------|
| `crypto.randomUUID()`| Generate a RFC-4122 UUID v4                     |
| `crypto.randomBytes(n)` | n cryptographically random bytes as hex      |
| `crypto.randomHex(n)`| n random bytes as a hex string                  |
| `crypto.token(n)`    | Random URL-safe token of n bytes                |

### Password hashing

| Function                            | Description                         |
|-------------------------------------|-------------------------------------|
| `crypto.hashPassword(password)`     | Bcrypt hash at cost 10              |
| `crypto.verifyPassword(pwd, hash)`  | Verify a bcrypt hash                |

### JWT (embedded in crypto)

`std.crypto` also exposes a `jwt` sub-object:

```lx
val crypto = @import("std.crypto")

val token   = crypto.jwt.sign({ userId: 42, role: "admin" }, "secret")
val payload = crypto.jwt.verify(token, "secret")  // object or null
```

For a dedicated JWT module, see `std.jwt` below.

---

## `std.fs` — File system

```lx
val fs = @import("std.fs")
```

### Reading

| Function               | Returns | Description                        |
|------------------------|---------|------------------------------------|
| `fs.readFile(path)`    | string  | Read entire file as a UTF-8 string |
| `fs.readLines(path)`   | array   | Read file and split by newline     |

### Writing

| Function                    | Description                   |
|-----------------------------|-------------------------------|
| `fs.writeFile(path, data)`  | Write (overwrite) a file      |
| `fs.appendFile(path, data)` | Append to a file              |

### File operations

| Function              | Description                             |
|-----------------------|-----------------------------------------|
| `fs.delete(path)`     | Delete a file                           |
| `fs.deleteAll(path)`  | Delete a file or directory recursively  |
| `fs.rename(src, dst)` | Rename a file or directory              |
| `fs.moveFile(src, dst)` | Move a file to a new path             |
| `fs.copy(src, dst)`   | Copy a file                             |
| `fs.copyFile(src, dst)` | Alias for `fs.copy`                   |

### Directory operations

| Function             | Returns | Description                         |
|----------------------|---------|-------------------------------------|
| `fs.mkdir(path)`     | —       | Create directory and all parents    |
| `fs.rmdir(path)`     | —       | Remove an empty directory           |
| `fs.list(path)`      | array   | List directory entries              |
| `fs.readDir(path)`   | array   | Alias for `fs.list`                 |

Each entry returned by `fs.list` / `fs.readDir` is an object:

```lx
{ name, path, isDir, isFile, size }
```

### Metadata

| Function            | Returns | Description                                           |
|---------------------|---------|-------------------------------------------------------|
| `fs.exists(path)`   | boolean | True if path exists                                   |
| `fs.stat(path)`     | object  | `{ name, size, isDir, isFile, mode, modTime }`        |
| `fs.isDir(path)`    | boolean | True if path is a directory                           |
| `fs.isFile(path)`   | boolean | True if path is a regular file                        |
| `fs.size(path)`     | number  | File size in bytes                                    |

---

## `std.http` — HTTP client and server

```lx
val http = @import("std.http")
```

### Client

| Function                          | Returns  | Description        |
|-----------------------------------|----------|--------------------|
| `http.get(url, headers?)`         | response | GET request        |
| `http.post(url, body, headers?)`  | response | POST request       |
| `http.put(url, body, headers?)`   | response | PUT request        |
| `http.delete(url, headers?)`      | response | DELETE request     |
| `http.patch(url, body, headers?)` | response | PATCH request      |

Response object: `{ status, body, headers }`.

### Server

Create a server with `http.createServer(handler)`, then start it with
`http.listen(server, port, host, onReady?)`:

```lx
val server = http.createServer(fn(req, res) {
  if req.url == "/" {
    http.text(res, "Hello from Lunex!", 200)
  } else if req.url == "/ping" {
    http.json(res, struct { status = "ok" }, 200)
  } else {
    http.json(res, struct { error = "not found" }, 404)
  }
})

http.listen(server, 3000, "0.0.0.0", fn() {
  io.log("Listening on http://localhost:3000")
})
```

**Request object:** `{ method, url, headers, body }`

**Response helpers:** `http.text(res, text, status)`, `http.json(res, value, status)`

---

## `std.ws` — WebSockets

```lx
val ws = @import("std.ws")
```

### Server

```lx
val server = ws.listen(8081)
server.on("connect",    fn(client) { io.log("client connected") })
server.on("message",    fn(client, msg) { client.send("echo: " + msg) })
server.on("disconnect", fn(client) { io.log("client left") })
server.start()
```

### Client

```lx
val client = ws.connect("ws://localhost:8081")
client.on("message", fn(msg) { io.log("server:", msg) })
client.send("hello")
client.close()
```

---

## `std.db` — In-memory database

```lx
val db = @import("std.db")
```

`db.table(name)` returns a table object you can query and mutate:

```lx
val users = db.table("users")
```

### Schema definition (optional)

```lx
users.define({
  id:    { type: "string", default: "$uuid" }
  name:  { type: "string", required: true }
  email: { type: "string", required: true, unique: true }
  age:   { type: "number", default: 0 }
})
```

### Table methods

| Method                         | Description                              |
|--------------------------------|------------------------------------------|
| `table.insert(record)`         | Insert a record                          |
| `table.find(query)`            | Find all records matching a query struct |
| `table.findOne(query)`         | Find the first matching record           |
| `table.update(query, patch)`   | Update matching records                  |
| `table.delete(query)`          | Delete matching records                  |
| `table.all()`                  | Return all records                       |
| `table.count(query?)`          | Count matching records                   |
| `table.clear()`                | Remove all records                       |

```lx
val users = db.table("users")
users.insert(struct { name = "Alice", email = "alice@example.com", age = 30 })
users.insert(struct { name = "Bob",   email = "bob@example.com",   age = 25 })

val alice = users.findOne(struct { email = "alice@example.com" })
io.log(alice.name)  // Alice

users.update(
  struct { email = "alice@example.com" },
  struct { age = 31 }
)

io.log("total:", users.count())
io.table(users.all())
```

---

## `std.jwt` — JSON Web Tokens

```lx
val jwt = @import("std.jwt")
```

| Function                      | Returns         | Description                                   |
|-------------------------------|-----------------|-----------------------------------------------|
| `jwt.sign(payload, secret)`   | string          | Sign a payload; returns a JWT string          |
| `jwt.verify(token, secret)`   | object \| null  | Verify token; returns payload or null         |
| `jwt.decode(token)`           | object          | Decode without signature verification         |

---

## `std.os` — Operating system

```lx
val os = @import("std.os")
```

### Process

| Function         | Returns | Description                  |
|------------------|---------|------------------------------|
| `os.getpid()`    | number  | Current process ID           |
| `os.getppid()`   | number  | Parent process ID            |
| `os.exit(code?)` | —       | Exit the process             |
| `os.args()`      | array   | Command-line arguments       |

### Platform info

| Function         | Returns | Description                                              |
|------------------|---------|----------------------------------------------------------|
| `os.platform()`  | string  | `"linux"`, `"darwin"`, `"windows"`, `"android"`          |
| `os.arch()`      | string  | `"amd64"`, `"arm64"`, etc.                               |
| `os.hostname()`  | string  | Machine hostname                                         |
| `os.cpus()`      | number  | Number of logical CPUs                                   |

### Working directory

| Function         | Returns | Description                        |
|------------------|---------|------------------------------------|
| `os.cwd()`       | string  | Current working directory          |
| `os.chdir(path)` | —       | Change working directory           |

### Environment variables

| Function                    | Returns         | Description                                    |
|-----------------------------|-----------------|------------------------------------------------|
| `os.getenv(key)`            | string \| null  | Read environment variable                      |
| `os.setenv(key, value)`     | —               | Write an environment variable                  |
| `os.unsetenv(key)`          | —               | Remove an environment variable                 |
| `os.environ()`              | object          | All environment variables as an object         |
| `os.expandEnv(s)`           | string          | Expand `$VAR` and `${VAR}` in a string         |

### Shell execution

| Function                  | Returns | Description                          |
|---------------------------|---------|--------------------------------------|
| `os.exec(cmd, opts?)`     | object  | Run a command synchronously          |
| `os.execSync(cmd, opts?)` | object  | Alias for `os.exec`                  |
| `os.spawn(cmd, opts?)`    | object  | Run a command in the background      |

`os.exec` returns `{ stdout, stderr, code, ok }`.
`os.spawn` returns `{ pid, wait(), kill() }`.

Optional opts object: `{ cwd, env, timeout }`.

```lx
val result = os.exec("git --version")
if result.ok {
  io.success(result.stdout)
} else {
  io.warn("git not found")
}
```

### File system (path utilities)

| Function                  | Returns | Description                              |
|---------------------------|---------|------------------------------------------|
| `os.join(...parts)`       | string  | Join path segments                       |
| `os.dirname(path)`        | string  | Parent directory of a path               |
| `os.basename(path)`       | string  | File name portion of a path              |
| `os.stat(path)`           | object  | `{ name, size, isDir, isFile, mode, modTime }` |
| `os.exists(path)`         | boolean | True if path exists                      |
| `os.mkdir(path)`          | —       | Create directory and all parents         |
| `os.remove(path)`         | —       | Delete a file or empty directory         |
| `os.rename(src, dst)`     | —       | Rename or move a path                    |
| `os.listDir(path)`        | array   | List directory entries                   |
| `os.glob(pattern)`        | array   | Expand a glob pattern                    |
| `os.tempDir()`            | string  | Path to a system temporary directory     |
| `os.tempFile()`           | string  | Path to a new temporary file             |

---

## `std.regex` — Regular expressions

```lx
val regex = @import("std.regex")
```

Uses Go's RE2 syntax (no lookaheads or backreferences).

### Testing

| Function                   | Returns | Description                               |
|----------------------------|---------|-------------------------------------------|
| `regex.test(s, pattern)`   | boolean | True if pattern matches anywhere in s     |
| `regex.isValid(pattern)`   | boolean | True if pattern is valid RE2 syntax       |

### Matching

| Function                     | Returns        | Description                                  |
|------------------------------|----------------|----------------------------------------------|
| `regex.match(s, pattern)`    | string \| null | First matching substring                     |
| `regex.matchAll(s, pattern)` | array          | All non-overlapping matches                  |
| `regex.index(s, pattern)`    | number         | Start index of first match (−1 if none)      |
| `regex.indices(s, pattern)`  | array          | Start indices of all matches                 |
| `regex.count(s, pattern)`    | number         | Number of non-overlapping matches            |

### Capture groups

| Function                         | Returns | Description                                   |
|----------------------------------|---------|-----------------------------------------------|
| `regex.groups(s, pattern)`       | array   | Capture groups from the first match           |
| `regex.groupsAll(s, pattern)`    | array   | Capture groups from every match               |
| `regex.namedGroups(s, pattern)`  | object  | Named capture groups as an object             |

### Replacement

| Function                              | Returns | Description                     |
|---------------------------------------|---------|---------------------------------|
| `regex.replace(s, pattern, repl)`     | string  | Replace first match             |
| `regex.replaceAll(s, pattern, repl)`  | string  | Replace all matches             |
| `regex.replaceFunc(s, pattern, fn)`   | string  | Replace with function output    |

### Splitting

| Function                  | Returns | Description          |
|---------------------------|---------|----------------------|
| `regex.split(s, pattern)` | array   | Split s on pattern   |

### Extraction helpers

| Function                   | Returns | Description                         |
|----------------------------|---------|-------------------------------------|
| `regex.extractNumbers(s)`  | array   | Extract all numeric substrings      |
| `regex.extractEmails(s)`   | array   | Extract all email addresses         |
| `regex.extractUrls(s)`     | array   | Extract all URLs                    |

### Escaping

| Function           | Returns | Description                            |
|--------------------|---------|----------------------------------------|
| `regex.escape(s)`  | string  | Escape all RE2 metacharacters in s     |

---

## `std.env` — Environment variables

```lx
val env = @import("std.env")
```

| Function               | Returns         | Description                             |
|------------------------|-----------------|-----------------------------------------|
| `env.get(key)`         | string \| null  | Read variable; null if not set          |
| `env.get(key, default)`| string          | Read with a fallback default            |
| `env.set(key, value)`  | —               | Write an environment variable           |
| `env.has(key)`         | boolean         | True if the variable is set             |
| `env.all()`            | object          | All environment variables as an object  |

---

## `runtime` — Runtime introspection

```lx
val runtime = @import("runtime")
```

| Function                      | Returns | Description                                |
|-------------------------------|---------|--------------------------------------------|
| `runtime.version()`           | string  | Lunex version string                       |
| `runtime.globals()`           | array   | Names of all globally visible bindings     |
| `runtime.getGlobal(name)`     | value   | Read a global by name                      |
| `runtime.setGlobal(name, v)`  | —       | Write a global by name                     |
| `runtime.hasGlobal(name)`     | boolean | True if global exists                      |
| `runtime.typeOf(v)`           | string  | Type name of a value                       |
| `runtime.gc()`                | —       | Request a garbage collection pass          |
