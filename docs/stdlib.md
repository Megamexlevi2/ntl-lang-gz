# Standard Library Reference

Complete API reference for all built-in modules in Lunex v0.9.2.

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
| `io.err(...args)`          | Print to stderr, colored red                       |
| `io.warn(...args)`         | Print to stderr, colored yellow                    |
| `io.info(...args)`         | Print to stdout, colored cyan                      |
| `io.success(...args)`      | Print to stdout with a green `✔` prefix            |
| `io.write(s)`              | Write a string to stdout with no trailing newline  |
| `io.newline(n?)`           | Print `n` blank lines (default 1)                  |
| `io.table(rows)`           | Render an array of structs/objects as a table      |
| `io.json(val)`             | Pretty-print any value as formatted JSON            |
| `io.hr(len?, char?)`       | Print a horizontal rule                            |
| `io.banner(text)`          | Print a highlighted banner                         |
| `io.clear()`               | Clear the terminal screen                          |

None of `io.err`/`io.warn`/`io.info` add a text prefix like `[ERROR]` —
they colorize the message only. Colors are skipped automatically when
stdout isn't an interactive terminal.

### Spinner

`io.spinner()` starts an animated spinner and returns a control object.
Each call to `tick(message?)` advances the animation by one frame and
redraws the line with the given message; `stop()` finishes the line with
a newline:

```lx
val sp = io.spinner()
sp.tick("Loading data...")
sp.tick("Loading data...")
sp.stop()
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
io.color("red", s)   // named color
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

| Name             | Value                |
|-------------------|-----------------------|
| `math.PI`         | 3.141592653589793     |
| `math.E`          | 2.718281828459045     |
| `math.LN2`        | ln(2)                  |
| `math.LN10`       | ln(10)                 |
| `math.LOG2E`      | log₂(e)                |
| `math.LOG10E`     | log₁₀(e)               |
| `math.SQRT2`      | √2                     |
| `math.SQRT1_2`    | √(1/2)                 |
| `math.PHI`        | golden ratio           |
| `math.Infinity`   | positive infinity      |
| `math.NaN`        | not-a-number           |

### Basic functions

| Function                  | Description                            |
|-----------------------------|------------------------------------------|
| `math.abs(x)`               | Absolute value                          |
| `math.ceil(x)`              | Round up to nearest integer             |
| `math.floor(x)`             | Round down to nearest integer           |
| `math.round(x)`             | Round to nearest integer                |
| `math.trunc(x)`             | Truncate fractional part                |
| `math.sign(x)`              | −1, 0, or 1                             |
| `math.sqrt(x)`              | Square root                             |
| `math.cbrt(x)`              | Cube root                               |
| `math.pow(x, y)`            | x to the power y                        |
| `math.hypot(a, b)`          | sqrt(a² + b²)                           |
| `math.min(a, b, ...)`       | Smallest of all arguments               |
| `math.max(a, b, ...)`       | Largest of all arguments                |
| `math.clamp(v, lo, hi)`     | Clamp v to [lo, hi]                     |
| `math.lerp(a, b, t)`        | Linear interpolation between a and b    |

### Exponentials and logarithms

| Function          | Description                                  |
|---------------------|-------------------------------------------------|
| `math.exp(x)`       | e^x                                              |
| `math.exp2(x)`      | 2^x                                              |
| `math.expm1(x)`     | e^x − 1, accurate for small x                    |
| `math.log(x)`       | Natural logarithm                                |
| `math.log2(x)`      | Base-2 logarithm                                 |
| `math.log10(x)`     | Base-10 logarithm                                |
| `math.log1p(x)`     | Natural logarithm of 1 + x, accurate for small x |

### Trigonometry

| Function            | Description                    |
|-----------------------|-----------------------------------|
| `math.sin(x)`         | Sine (radians)                    |
| `math.cos(x)`         | Cosine (radians)                  |
| `math.tan(x)`         | Tangent (radians)                 |
| `math.asin(x)`        | Arcsine                           |
| `math.acos(x)`        | Arccosine                         |
| `math.atan(x)`        | Arctangent                        |
| `math.atan2(y, x)`    | Two-argument arctangent           |
| `math.sinh(x)`        | Hyperbolic sine                   |
| `math.cosh(x)`        | Hyperbolic cosine                 |
| `math.tanh(x)`        | Hyperbolic tangent                |
| `math.asinh(x)`       | Inverse hyperbolic sine           |
| `math.acosh(x)`       | Inverse hyperbolic cosine         |
| `math.atanh(x)`       | Inverse hyperbolic tangent        |
| `math.degToRad(d)`    | Degrees to radians                |
| `math.radToDeg(r)`    | Radians to degrees                |

### Random

| Function                   | Description                          |
|-------------------------------|------------------------------------------|
| `math.random()`               | Uniform float in [0, 1)                  |
| `math.random(max)`            | Uniform float in [0, max)                |
| `math.random(min, max)`       | Uniform float in [min, max)              |
| `math.randomInt()`            | Random non-negative integer               |
| `math.randomInt(max)`         | Random integer in [0, max)                |
| `math.randomInt(min, max)`    | Random integer in **[min, max)** — max is exclusive, not inclusive |
| `math.seed(n)`                | Seed the random number generator         |

### Number theory

| Function                   | Description                           |
|-------------------------------|-------------------------------------------|
| `math.gcd(a, b)`              | Greatest common divisor                   |
| `math.lcm(a, b)`              | Least common multiple                     |
| `math.isPrime(n)`             | True if n is prime                        |
| `math.factorial(n)`           | n!                                        |
| `math.combinations(n, k)`     | Binomial coefficient C(n, k)              |
| `math.permutations(n, k)`     | P(n, k)                                   |
| `math.fib(n)`                 | The nth Fibonacci number                  |
| `math.primes(n)`              | Array of all primes up to and including n |

### Statistics

Each of these accepts either a single array argument or multiple numeric
arguments (`math.mean([1, 2, 3])` and `math.mean(1, 2, 3)` are equivalent).
`math.variance` and `math.stdDev` require an array argument.

| Function              | Description                              |
|--------------------------|-----------------------------------------------|
| `math.sum(...)`          | Sum of the values                              |
| `math.product(...)`      | Product of the values                          |
| `math.mean(...)`         | Arithmetic mean                                |
| `math.variance(arr)`     | Population variance                            |
| `math.stdDev(arr)`       | Population standard deviation                  |

### Base conversion

| Function                    | Description                                       |
|--------------------------------|--------------------------------------------------------|
| `math.toBinary(n)`             | Binary string representation                            |
| `math.toHex(n, prefix?)`       | Hex string representation; pass `true` to include `0x`  |
| `math.toOctal(n)`              | Octal string representation                             |

### Type checks

| Function              | Description                        |
|--------------------------|---------------------------------------|
| `math.isNaN(v)`          | True if the value is NaN              |
| `math.isFinite(v)`       | True if the value is finite           |
| `math.isInfinite(v)`     | True if the value is +/- infinity     |

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
| `utils.randInt(min?, max?)`| Random integer in **[min, max)** — max is exclusive, not inclusive |
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

`utils.template` also accepts `${key}` placeholders — both styles work in the
same call, e.g. `utils.template("Hi {{name}}, ${name}!", { name: "Ada" })`.

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

### Type helpers

| Function                | Description                                          |
|---------------------------|----------------------------------------------------------|
| `utils.type(v)`           | Runtime type name of a value, e.g. `"string"`, `"array"`  |
| `utils.isEmpty(v)`        | True for `""`, `[]`, `{}`, or a nullish value              |
| `utils.isNil(v)`          | True if the value is `null` or `undefined`                |
| `utils.isEmail(s)`        | True if the string looks like an email address             |
| `utils.isUrl(s)`          | True if the string looks like a URL                        |
| `utils.isNumeric(s)`      | True if the string parses as a number                      |
| `utils.toNumber(v)`       | Convert a value to a number (`NaN` on failure)              |
| `utils.toString(v)`       | Convert a value to its string representation                |
| `utils.clone(v)`          | Deep clone an array or object                               |
| `utils.equal(a, b)`       | Deep equality check                                          |

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
| `datetime.parse(s, format)`        | datetime  | Parse `s` using a custom layout (same tokens as `datetime.format`) |

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

**Layout tokens:**

| Token   | Meaning                          |
|---------|-----------------------------------|
| `YYYY`  | 4-digit year                      |
| `YY`    | 2-digit year                      |
| `MMMM`  | Full month name (`January`)        |
| `MMM`   | Short month name (`Jan`)           |
| `MM`    | 2-digit month                      |
| `M`     | Month, no leading zero              |
| `DD`    | 2-digit day                        |
| `D`     | Day, no leading zero                 |
| `dddd`  | Full weekday name (`Monday`)         |
| `ddd`   | Short weekday name (`Mon`)            |
| `HH`    | 2-digit hour, 24-hour                 |
| `hh`    | 2-digit hour, 12-hour                  |
| `h`     | Hour, 12-hour, no leading zero          |
| `mm`    | 2-digit minute                          |
| `ss`    | 2-digit second                           |
| `SSS`   | Milliseconds, zero-padded to 3            |
| `A`     | `AM`/`PM`                                  |
| `a`     | `am`/`pm`                                    |
| `Z`     | UTC offset, e.g. `+00:00`                     |
| `ZZ`    | UTC offset without colon, e.g. `+0000`          |

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

### Constructing from parts

| Function                 | Returns  | Description                                              |
|----------------------------|----------|--------------------------------------------------------------|
| `datetime.fromParts(parts)`| datetime | Build from an object: `{ year, month, day, hour?, minute?, second?, ms? }` (all default to the epoch/midnight if omitted) |

```lx
val dt = datetime.fromParts({ year: 2026, month: 6, day: 25, hour: 14 })
```

> Note the field names are `minute` and `second` (not `min`/`sec`).

### Other

| Function                       | Description                                                  |
|-----------------------------------|-------------------------------------------------------------------|
| `datetime.timezone(dt, tzName)`   | Convert dt to another IANA timezone (e.g. `"America/Sao_Paulo"`)  |
| `datetime.sleep(ms, unit?)`       | Pause execution for `ms` in the given unit (default: `"ms"`)      |

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
| `crypto.compare(a, b)` | Constant-time string comparison (safe for secrets/tokens) |

### Key derivation

| Function                                     | Description                                              |
|--------------------------------------------------|----------------------------------------------------------------|
| `crypto.pbkdf2(password, salt, iterations?, keyLen?)` | PBKDF2-HMAC-SHA256 key derivation, returns hex; defaults: 100000 iterations, 32-byte key |

### Password hashing

| Function                            | Description                         |
|-------------------------------------|-------------------------------------|
| `crypto.hashPassword(password)`     | Bcrypt hash at cost 10              |
| `crypto.verifyPassword(pwd, hash)`  | Verify a bcrypt hash                |

### JWT (embedded in crypto)

`std.crypto` also exposes a `jwt` sub-object:

```lx
val crypto = @import("std.crypto")

val token   = crypto.jwt.sign({ userId: 42, role: "admin" }, "secret", 3600) // optional expiresIn, in seconds
val payload = crypto.jwt.verify(token, "secret")  // object or null if invalid/expired
val raw     = crypto.jwt.decode(token)            // payload object, without verifying the signature
```

> **`crypto.jwt` is a separate, simpler implementation from the dedicated
> `std.jwt` module below** — they don't share code and their return shapes
> differ. `crypto.jwt.verify` returns the payload object directly, or `null`
> if the token is invalid/expired. `std.jwt.verify` (below) instead always
> returns a `{ valid, payload }` / `{ valid: false, error }` wrapper object.
> Tokens signed with one are compatible with the other (both are standard
> HMAC-signed JWTs), but don't mix up which `verify`/`decode` shape to expect.

For the dedicated JWT module, see `std.jwt` below.

---

## `std.fs` — File system

```lx
val fs = @import("std.fs")
```

### Reading

| Function               | Returns | Description                        |
|-------------------------|---------|--------------------------------------|
| `fs.readFile(path)`     | string  | Read entire file as a UTF-8 string   |
| `fs.readLines(path)`    | array   | Read file and split by newline       |

### Writing

| Function                    | Description                   |
|-------------------------------|-----------------------------------|
| `fs.writeFile(path, data)`    | Write (overwrite) a file          |
| `fs.appendFile(path, data)`   | Append to a file                  |

### Data formats

| Function                            | Returns | Description                                          |
|----------------------------------------|---------|------------------------------------------------------------|
| `fs.readJSON(path)`                     | value   | Read a file and parse it as JSON                             |
| `fs.writeJSON(path, value, indent?)`    | boolean | Serialize a value to JSON and write it; `indent` is a space count |
| `fs.readCSV(path)`                      | array   | Read a CSV file into an array of rows                        |
| `fs.writeCSV(path, rows)`               | boolean | Write an array of objects (or arrays) as CSV                 |

### File operations

| Function              | Description                             |
|--------------------------|----------------------------------------------|
| `fs.delete(path)`        | Delete a file                                 |
| `fs.deleteAll(path)`     | Delete a file or directory recursively        |
| `fs.rename(src, dst)`    | Rename a file or directory                    |
| `fs.moveFile(src, dst)`  | Move a file to a new path                     |
| `fs.copy(src, dst)`      | Copy a file                                   |
| `fs.copyFile(src, dst)`  | Alias for `fs.copy`                           |

### Directory operations

| Function             | Returns | Description                         |
|------------------------|---------|-----------------------------------------|
| `fs.mkdir(path)`       | —       | Create directory and all parents        |
| `fs.ensureDir(path)`   | boolean | Alias for `fs.mkdir`, returns success   |
| `fs.rmdir(path)`       | —       | Remove an empty directory               |
| `fs.list(path)`        | array   | List directory entries                  |
| `fs.readDir(path)`     | array   | Alias for `fs.list`                     |

Each entry returned by `fs.list` / `fs.readDir` is an object:

```lx
{ name, path, isDir, isFile, size }
```

### Metadata

| Function            | Returns | Description                                           |
|-----------------------|---------|-------------------------------------------------------------|
| `fs.exists(path)`     | boolean | True if path exists                                          |
| `fs.stat(path)`       | object  | `{ name, size, isDir, isFile, mode, modTime }`                |
| `fs.isDir(path)`      | boolean | True if path is a directory                                  |
| `fs.isFile(path)`     | boolean | True if path is a regular file                               |
| `fs.size(path)`       | number  | File size in bytes                                           |

### Path helpers

| Function                     | Returns | Description                                     |
|---------------------------------|---------|-------------------------------------------------------|
| `fs.abs(path)`                  | string  | Absolute path                                          |
| `fs.join(...parts)`             | string  | Join path segments using the OS separator              |
| `fs.dirname(path)`              | string  | Parent directory                                        |
| `fs.basename(path, suffix?)`    | string  | Final path segment, with an optional suffix stripped    |
| `fs.extname(path)`              | string  | File extension, including the leading `.`               |
| `fs.glob(pattern)`              | array   | Paths matching a glob pattern                            |
| `fs.cwd()`                      | string  | Current working directory                                |
| `fs.home()`                     | string  | Current user's home directory                            |

### Temp files

| Function                | Returns | Description                                    |
|---------------------------|---------|------------------------------------------------------|
| `fs.tempFile(prefix?)`    | string  | Create and return the path to a new temp file          |
| `fs.tempDir(prefix?)`     | string  | Create and return the path to a new temp directory      |

---

## `std.http` — HTTP client and server

```lx
val http = @import("std.http")
```

### Client

| Function                          | Returns  | Description        |
|--------------------------------------|----------|---------------------------|
| `http.request(method, url, opts?)`   | response | Request with an arbitrary method |
| `http.get(url, opts?)`               | response | GET request                |
| `http.post(url, opts?)`              | response | POST request                |
| `http.put(url, opts?)`               | response | PUT request                 |
| `http.patch(url, opts?)`             | response | PATCH request                |
| `http.delete(url, opts?)`            | response | DELETE request                |
| `http.head(url)`                     | response | HEAD request                   |

`opts` is `{ body, headers, timeout }`. `body` is JSON-encoded
automatically (and `Content-Type` set) if it's an object or array.

Response object: `{ ok, status, body, headers, text }` — or `{ ok: false,
error }` if the request itself failed (e.g. couldn't connect). `body` is
JSON-decoded automatically when the response looks like JSON; `text` always
holds the raw response text regardless of content type. The response object
also has a `.json()` method that parses `text` as JSON on demand.

### URL helpers

| Function                        | Returns | Description                                             |
|-------------------------------------|---------|----------------------------------------------------------------|
| `http.parseURL(url)`                | object  | `{ protocol, host, path, query, search }`                        |
| `http.buildURL(base, params?)`      | string  | Append a query-string object to a base URL                       |
| `http.encode(s)`                    | string  | URL-encode a string                                               |
| `http.decode(s)`                    | string  | URL-decode a string                                               |
| `http.statusText(code)`             | string  | Standard reason phrase for an HTTP status code                    |

### Server

`http.createServer(handler?)` creates a server. `handler`, if given, is a
catch-all `fn(req, res)` called for any request that doesn't match a
registered route. Start it with `http.listen(server, port, host?, onReady?)`
— `host` defaults to listening on all interfaces and may be omitted in
favor of passing `onReady` as the third argument directly.

```lx
val server = http.createServer()

server
  .get("/", fn(req, res) { res.text("Hello from Lunex!") })
  .get("/users/:id", fn(req, res) { res.json({ id: req.params.id }) })
  .post("/users", fn(req, res) { res.json(req.body, 201) })
  .use(fn(req, res, next) { io.log(req.method, req.path); next() })

http.listen(server, 3000, fn() {
  io.log("Listening on http://localhost:3000")
})
```

**Request object:** `{ method, url, path, query, params, headers, body, ip, host }`

- `path` is the URL without the query string; `query` is the parsed query
  object; `params` holds named route params (e.g. `:id`) matched by the
  current route, and is empty for requests that don't go through a
  pattern-matched route.

**Router methods on a server**, each returning the server so calls can be
chained:

| Method                              | Description                                          |
|--------------------------------------|-------------------------------------------------------|
| `server.get(pattern, handler)`        | Register a GET route                                   |
| `server.post(pattern, handler)`       | Register a POST route                                  |
| `server.put(pattern, handler)`        | Register a PUT route                                   |
| `server.patch(pattern, handler)`      | Register a PATCH route                                 |
| `server.delete(pattern, handler)`     | Register a DELETE route                                |
| `server.all(pattern, handler)`        | Register a route matching any method                     |
| `server.use(handler)`                 | Register middleware run before route handlers, `fn(req, res, next)` |
| `server.use(pattern, handler)`        | Middleware scoped to paths under `pattern`                |
| `server.close()`                      | Stop the server                                            |
| `server.port`                         | The port the server is listening on                         |

Route patterns support `:name` params (e.g. `/users/:id`).

**Response object methods** (the `res` passed into every handler):

| Method                              | Description                                          |
|--------------------------------------|---------------------------------------------------------|
| `res.json(value, status?)`            | Send a JSON response (default status 200)                |
| `res.send(value, status?)`            | Send a response; JSON-encodes objects/arrays, sends strings as text |
| `res.text(text, status?)`             | Send a plain-text response                                |
| `res.html(html, status?)`             | Send an HTML response                                     |
| `res.redirect(url, status?)`          | Send a redirect (default status 302)                        |
| `res.status(code)`                    | Set the status code for a subsequent `.send`/`.end`; returns `res` |
| `res.setHeader(name, value)`          | Set a response header; returns `res`                        |
| `res.getHeader(name)`                 | Read a header already set on this response                    |
| `res.removeHeader(name)`              | Remove a previously set header                                 |
| `res.cookie(name, value, opts?)`      | Set a `Set-Cookie` header                                        |
| `res.clearCookie(name)`               | Clear a cookie                                                    |
| `res.end(body?)`                      | End the response, optionally with a raw body                       |

The global convenience functions `http.text(res, text, status)`,
`http.json(res, value, status)`, `http.html(res, html, status)`, and
`http.redirect(res, url, status?)` are equivalent to calling the
corresponding method on `res` directly — both styles work.

| Function                              | Description                                          |
|--------------------------------------------|-------------------------------------------------------------|
| `http.parseBody(req)`                        | Parse `req.body`: JSON-decodes it if it looks like JSON, otherwise returns it as-is |
| `http.serveStatic(dir)`                      | Returns a `{ staticDir }` descriptor for serving static files from `dir`; pass it to `server.use(...)` |

---

## `std.ws` — WebSockets

```lx
val ws = @import("std.ws")
```

A minimal WebSocket server and client. Text frames only.

### Server

`ws.createServer(port, connHandler?)` starts listening immediately and
returns a server handle. `connHandler(client)` is called once per new
connection with a client object.

```lx
val server = ws.createServer(8081, fn(client) {
  io.log("client connected")
  client.onMessage(fn(msg) { client.send("echo: " + msg) })
  client.onClose(fn() { io.log("client left") })
})

io.log("listening on port", server.port)
```

| Function                     | Description                                        |
|-------------------------------|----------------------------------------------------|
| `ws.createServer(port, connHandler?)` | Start a server; returns `{ port, broadcast(msg), clientCount(), close() }` |
| `ws.send(client, msg)`         | Shorthand for `client.send(msg)`                    |
| `ws.onMessage(client, fn)`     | Shorthand for `client.onMessage(fn)`                |
| `ws.onClose(client, fn)`       | Shorthand for `client.onClose(fn)`                  |
| `ws.closeServer(server)`       | Shorthand for `server.close()`                      |

A **client object** (passed into `connHandler`, or returned by `ws.connect`)
has these methods:

| Method                  | Description                                        |
|--------------------------|----------------------------------------------------|
| `client.send(msg)`       | Send a text frame (objects/arrays are JSON-encoded) |
| `client.close()`         | Close the connection                                |
| `client.onMessage(fn)`   | Register a handler for `fn(message)`                |
| `client.onClose(fn)`     | Register a handler called when the connection closes |
| `client.isClosed()`      | True if the connection has been closed              |

A **server object** (returned by `ws.createServer`) has:

| Method                     | Description                            |
|------------------------------|-----------------------------------------|
| `server.port`                | The port the server is listening on      |
| `server.broadcast(msg)`      | Send a message to every connected client |
| `server.clientCount()`       | Number of currently connected clients    |
| `server.close()`             | Stop accepting new connections           |

### Client

```lx
val client = ws.connect("ws://localhost:8081")
client.onMessage(fn(msg) { io.log("server:", msg) })
client.send("hello")
client.close()
```

| Function              | Description                                  |
|------------------------|-----------------------------------------------|
| `ws.connect(url)`      | Connect to a server; returns a client object    |
| `ws.closeClient(client)` | Shorthand for `client.close()`                |

> **Known limitation:** only the `ws://` scheme is supported. `ws.connect`
> with a `wss://` URL always fails with "wss:// (TLS) not supported; use
> ws://" — there is no TLS/secure WebSocket support in this build.

---

## `std.db` — SQLite-backed document database

```lx
val db = @import("std.db")
```

Each named database is a SQLite file stored on disk under
`.lunex/data/<n>.db` (created on first use). Tables are plain SQLite tables
holding JSON documents, so data survives process restarts; it is not an
in-memory store.

### Opening a database

| Function              | Description                                                |
|------------------------|--------------------------------------------------------------|
| `db.open(name?)`       | Open (or create) a named database; default name `"default"`    |
| `db.create(name?)`     | Alias for `db.open`                                            |
| `db.connect(name?)`    | Alias for `db.open`                                            |
| `db.drop(name)`        | Delete a database file entirely                                |
| `db.list()`            | Array of names of databases that exist on disk                 |
| `db.table(name)`       | Shorthand: get a table on the **default** database, no explicit `open()` needed |
| `db.collection(name)`  | Alias for `db.table`                                            |

`db.open()` / `db.create()` / `db.connect()` return a **database object**:

| Method                          | Description                                          |
|----------------------------------|--------------------------------------------------------|
| `database.table(name)`           | Get (or create) a table object                          |
| `database.collection(name)`      | Alias for `.table`                                       |
| `database.tables()`              | Array of table names in this database                    |
| `database.drop(name)`            | Drop one table from this database                         |
| `database.transaction(fn)`       | Run `fn(database)`; the return value is passed through     |
| `database.dump()`                | Object of `{ tableName: [rows...] }` for every table        |
| `database.load(data)`            | Bulk-insert from a `{ tableName: [rows...] }` object         |
| `database.close()`               | Close the underlying SQLite connection                       |
| `database.name` / `database.path`| The database's name and file path                            |

```lx
val users = db.table("users")   // shorthand for db.open().table("users")
```

### Schema definition (optional)

```lx
users.schema({
  id:    { type: "string", default: "$uuid" }
  name:  { type: "string", required: true }
  email: { type: "string", required: true, unique: true }
  age:   { type: "number", default: 0, min: 0 }
})
```

`table.define(...)` is an alias for `table.schema(...)`. Each field
definition may include:

| Key          | Effect                                                         |
|--------------|------------------------------------------------------------------|
| `type`        | Informational; not strictly enforced on every write               |
| `required`    | Insert fails if the field is missing                               |
| `unique`      | Creates a unique index; insert/update fails on duplicates          |
| `index`       | Creates a (non-unique) index on this field                         |
| `primary`     | Marks the field as a primary key                                   |
| `min` / `max` | Numeric bounds                                                    |
| `minLength` / `maxLength` | String length bounds                                   |
| `ref`         | Informational reference to another table (not enforced/joined automatically) |
| `enum`        | Array of allowed values                                            |
| `default`     | Static default value, or `"$uuid"` / `"$now"` / `"$seq"` for generated defaults |
| `onUpdate`    | e.g. `"now"` — refresh the field to the current time on every update |

`table.index(field(s), { unique? })` creates an index outside of `schema()`;
`field(s)` can be a single field name or an array for a compound index.

### Table methods — writing

| Method                          | Description                                          |
|-----------------------------------|--------------------------------------------------------|
| `table.insert(record)`             | Insert one record                                      |
| `table.insertMany(records)`        | Insert an array of records                              |
| `table.upsert(query, patch)`       | Update the first match, or insert `patch` if none match  |
| `table.update(query, patch)`       | Update all matching records; `patch` may be a plain object of fields, or `{ $set: {...} }` |
| `table.updateOne(query, patch)`    | Update only the first matching record                    |
| `table.delete(query)`              | Delete all matching records                              |
| `table.deleteOne(query)`           | Delete only the first matching record                     |
| `table.deleteById(id)`             | Delete the record with the given `_id`                    |
| `table.clear()`                    | Remove all records (keeps schema/indexes)                 |
| `table.drop()`                     | Drop the table entirely, including schema/indexes         |

### Table methods — reading

| Method                            | Description                                          |
|-------------------------------------|--------------------------------------------------------|
| `table.find(query?, options?)`       | Find matching records. `options`: `{ select, sort, orderBy, limit, offset, skip }` |
| `table.findOne(query?)`              | First matching record, or `null`                        |
| `table.findById(id)`                 | Record with the given `_id`, or `null`                   |
| `table.count(query?)`                | Count matching records                                   |
| `table.exists(query?)`               | True if at least one record matches                       |
| `table.distinct(field, query?)`      | Array of distinct values of `field` among matches          |
| `table.search(text, fields?)`        | Simple substring search across `fields` (all fields if omitted) |
| `table.dump()`                       | Return every record, ignoring any filter                   |
| `table.indexes()`                    | Array of `{ name, fields, unique }` for defined indexes     |

There is no `table.all()` — use `table.find()` with no arguments, or
`table.dump()`, to get every record.

### Query builder

`table.where(query)` returns a chainable query builder (as do `.select()`,
`.orderBy()`, and `.limit()` called directly on the table, though those three
start a fresh builder rather than adding to one):

```lx
val topActive = users
  .where({ active: true })
  .and({ age: { $gte: 18 } })
  .orderBy("age", "desc")
  .limit(10)
  .exec()
```

| Method                     | Description                                    |
|------------------------------|---------------------------------------------------|
| `.where(query)`               | Set (or replace) the filter                         |
| `.and(query)` / `.or(query)`  | Combine with the current filter using `$and`/`$or`   |
| `.select(fields)`             | Project only the given fields                        |
| `.orderBy(field, "asc"\|"desc"?)` | Add a sort key (can be called multiple times)   |
| `.limit(n)`                   | Limit the number of results                           |
| `.offset(n)` / `.skip(n)`     | Skip the first n results                               |
| `.page(n, size)`              | Shorthand for `offset((n-1)*size).limit(size)`         |
| `.exec()`                     | Run the query, return an array                         |
| `.first()`                    | Run the query, return the first result or `null`        |
| `.last()`                     | Run the query, return the last result or `null`          |
| `.count()`                    | Count matches (ignores limit/offset)                     |
| `.exists()`                   | True if any record matches                                |
| `.update(patch)`              | Update all matching records                               |
| `.delete()`                   | Delete all matching records                                |

### Query filter operators

Filters are plain objects. A bare value means equality; nested operator
objects support:

`$eq`, `$ne`, `$gt`, `$gte`, `$lt`, `$lte`, `$in`, `$nin`, `$like`, `$ilike`,
`$regex`, `$exists`, `$between`, `$contains`, `$size`, `$type`,
`$startsWith`, `$endsWith` — plus the boolean combinators `$and`, `$or`,
`$not`, `$nor` at the top level of a filter.

```lx
users.find({ age: { $gte: 18, $lt: 65 }, name: { $startsWith: "A" } })
users.find({ $or: [{ role: "admin" }, { role: "owner" }] })
```

### Aggregation

`table.aggregate(pipeline)` runs a MongoDB-style aggregation pipeline (an
array of stage objects) and returns an array of results. Supported stages:
`$match`, `$sort`, `$limit`, `$skip`, `$project`, `$group`, `$unwind`,
`$count`. Inside `$group`, accumulators `$count`, `$sum`, `$avg`, `$min`,
`$max`, `$first`, `$last`, `$push`, and `$addToSet` are supported.

```lx
val byRole = users.aggregate([
  { $match: { active: true } }
  { $group: { _id: "$role", total: { $count: {} } } }
])
```

Simple aggregates also have dedicated shortcuts that don't require a
pipeline: `table.sum(field, query?)`, `table.avg(field, query?)`,
`table.min(field, query?)`, `table.max(field, query?)`.

> **Known limitation:** `table.join(...)` is currently a stub — it validates
> its arguments but always returns an empty array. There is no cross-table
> join support yet.

### Change notifications

`table.watch(query?, fn)` calls `fn(event, record)` whenever a matching
record is inserted, updated, or deleted, and returns an `unwatch()`
function to stop listening.

### Transactions

`database.transaction(fn)` calls `fn(database)` and returns its result.
There is no automatic rollback on error — it is a convenience wrapper, not
an atomic transaction guarantee.

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
io.table(users.find())
```

---

## `std.buffer` — Byte buffers

```lx
val buffer = @import("std.buffer")
```

A `buffer` is a mutable, addressable block of bytes, separate from `string`
and `array`. It exists for binary formats, network protocols, and file
parsing — anywhere fixed-width integers and raw byte layout matter.

### Creating buffers

| Function                       | Description                                       |
|---------------------------------|----------------------------------------------------|
| `buffer.alloc(size)`            | New zero-filled buffer of `size` bytes             |
| `buffer.from(string)`           | Buffer from a UTF-8 string                         |
| `buffer.from(string, "hex")`    | Buffer decoded from a hex string                   |
| `buffer.from(string, "base64")` | Buffer decoded from a base64 string                |
| `buffer.from(array)`            | Buffer from an array of byte values (0-255)        |
| `buffer.concat(buf1, buf2, …)`  | New buffer with all inputs concatenated            |

### Reading and writing

Every `readX`/`writeX` method takes a byte offset first. Multi-byte methods
take an optional byte order argument last: `"le"` (default) or `"be"`.

| Method                                | Description                             |
|-----------------------------------------|--------------------------------------------|
| `buf.readU8(offset)` / `buf.writeU8(offset, v)`   | Unsigned 8-bit integer          |
| `buf.readI8(offset)` / `buf.writeI8(offset, v)`   | Signed 8-bit integer            |
| `buf.readU16(offset, order?)` / `buf.writeU16(offset, v, order?)` | Unsigned 16-bit integer |
| `buf.readI16(offset, order?)` / `buf.writeI16(offset, v, order?)` | Signed 16-bit integer   |
| `buf.readU32(offset, order?)` / `buf.writeU32(offset, v, order?)` | Unsigned 32-bit integer |
| `buf.readI32(offset, order?)` / `buf.writeI32(offset, v, order?)` | Signed 32-bit integer   |
| `buf.readU64(offset, order?)` / `buf.writeU64(offset, v, order?)` | Unsigned 64-bit integer |
| `buf.readI64(offset, order?)` / `buf.writeI64(offset, v, order?)` | Signed 64-bit integer   |
| `buf.readF32(offset, order?)` / `buf.writeF32(offset, v, order?)` | 32-bit float             |
| `buf.readF64(offset, order?)` / `buf.writeF64(offset, v, order?)` | 64-bit float             |

Reads and writes outside the buffer's bounds are no-ops rather than errors —
`read*` returns `0`.

### Other operations

| Method                     | Description                                       |
|------------------------------|-----------------------------------------------------|
| `buf.length`                | Number of bytes                                    |
| `buf.slice(start, end)`     | New buffer copied from a byte range                |
| `buf.copyFrom(src, offset?)`| Copy another buffer's bytes into this one           |
| `buf.fill(byteValue)`       | Fill every byte with a value                        |
| `buf.resize(newSize)`       | Grow (zero-padded) or shrink in place                |
| `buf.toHex()`               | Encode as a hex string                              |
| `buf.toBase64()`            | Encode as a base64 string                           |
| `buf.toString()`            | Decode as a UTF-8 string                            |
| `buf.toArray()`             | Convert to an array of byte values                   |

```lx
val buffer = @import("std.buffer")

val buf = buffer.alloc(8)
buf.writeU32(0, 4021, "be")
buf.writeI16(4, -12, "be")
io.log(buf.readU32(0, "be"))  // 4021
io.log(buf.toHex())
```

---

## `std.ints` — Fixed-width integer arithmetic

```lx
val ints = @import("std.ints")
```

Lunex numbers are 64-bit floats, so they don't overflow or wrap the way C's
fixed-width integers do. `std.ints` reproduces that wraparound behavior
explicitly, which matters when porting code that depends on exact overflow
semantics (checksums, hashing, binary protocol fields).

### Casts

Each cast truncates the input to the target width and reinterprets the bit
pattern, matching a C-style `(uint32_t)x` cast.

| Function     | Range                                    |
|---------------|--------------------------------------------|
| `ints.u8(x)`  | 0 to 255                                   |
| `ints.i8(x)`  | −128 to 127                                |
| `ints.u16(x)` | 0 to 65535                                 |
| `ints.i16(x)` | −32768 to 32767                            |
| `ints.u32(x)` | 0 to 4294967295                            |
| `ints.i32(x)` | −2147483648 to 2147483647                  |
| `ints.u64(x)` | 0 to 18446744073709551615                  |
| `ints.i64(x)` | −9223372036854775808 to 9223372036854775807 |

### Wrapping arithmetic

`add`, `sub`, and `mul` are provided for every width, each suffixed with the
target type (`U8`, `I8`, `U16`, `I16`, `U32`, `I32`, `U64`, `I64`). All wrap
silently on overflow instead of losing precision as plain `+`/`-`/`*` would
past 2^53.

```lx
ints.addU8(250, 10)   // 4   (250 + 10 wraps at 256)
ints.subI8(-128, 1)   // 127 (wraps the other way)
ints.mulU32(200000, 200000)
```

### Shifts and rotation

| Function                 | Description                          |
|----------------------------|------------------------------------------|
| `ints.shlU32(x, n)`        | Logical left shift, 32-bit               |
| `ints.shrU32(x, n)`        | Logical right shift, 32-bit              |
| `ints.shlU64(x, n)`        | Logical left shift, 64-bit               |
| `ints.shrU64(x, n)`        | Logical right shift, 64-bit              |
| `ints.rotlU32(x, n)`       | Rotate left, 32-bit                      |
| `ints.rotrU32(x, n)`       | Rotate right, 32-bit                     |

Bitwise `&`, `|`, `^`, `~`, `<<`, `>>`, and `>>>` are also available directly
as operators anywhere in Lunex — no import needed.

### Overflow checks

| Function                  | Description                             |
|-----------------------------|--------------------------------------------|
| `ints.isU8Overflow(x)`      | True if x falls outside 0-255              |
| `ints.isI32Overflow(x)`     | True if x falls outside the i32 range      |

### Constants

`ints.U8_MAX`, `ints.I8_MAX`, `ints.I8_MIN`, `ints.U16_MAX`, `ints.I16_MAX`,
`ints.I16_MIN`, `ints.U32_MAX`, `ints.I32_MAX`, `ints.I32_MIN`,
`ints.U64_MAX`, `ints.I64_MAX`, `ints.I64_MIN`.

---

## `std.jwt` — JSON Web Tokens

```lx
val jwt = @import("std.jwt")
```

| Function                                | Returns  | Description                                   |
|-------------------------------------------|----------|-----------------------------------------------|
| `jwt.sign(payload, secret, options?)`      | string   | Sign a payload; returns a JWT string           |
| `jwt.verify(token, secret)`                | object   | Verify a token — see return shape below         |
| `jwt.decode(token)`                        | object   | `{ header, payload }` — decoded without verifying the signature |
| `jwt.isExpired(token)`                     | boolean  | True if the token's `exp` claim is in the past   |
| `jwt.refresh(token, secret, expiresIn?)`   | string   | Verify `token`, then issue a new one with a fresh `iat`/`exp` |

`jwt.sign` automatically adds `iat` (issued-at) to the payload, and adds
`exp` if `expiresIn` is set. `options` is an object supporting:

| Key          | Description                                        |
|--------------|-------------------------------------------------------|
| `algorithm`   | Signing algorithm (default `"HS256"`)                  |
| `expiresIn`   | Lifetime in seconds; sets the `exp` claim                |
| `issuer`      | Sets the `iss` claim                                       |
| `audience`    | Sets the `aud` claim                                          |
| `subject`     | Sets the `sub` claim                                            |

`jwt.verify` **never returns `null`** — it always returns an object:

```lx
val result = jwt.verify(token, "secret")
if result.valid {
  io.log(result.payload)
} else {
  io.log("invalid:", result.error)
}
```

> **Note:** `std.jwt` is a distinct implementation from the `jwt` sub-object
> exposed by `std.crypto` (`crypto.jwt`) — see the note in the `std.crypto`
> section above. In particular, `crypto.jwt.verify` returns the payload
> directly (or `null`), while `jwt.verify` here always returns a
> `{ valid, payload }` / `{ valid: false, error }` wrapper.

---

## `std.os` — Operating system

```lx
val os = @import("std.os")
```

### Process

| Function         | Returns | Description                  |
|--------------------|---------|----------------------------------|
| `os.getpid()`      | number  | Current process ID               |
| `os.pid()`         | number  | Alias for `os.getpid()`          |
| `os.getppid()`     | number  | Parent process ID                |
| `os.ppid()`        | number  | Alias for `os.getppid()`         |
| `os.exit(code?)`   | —       | Exit the process                 |
| `os.args()`        | array   | Command-line arguments           |

### Platform info

| Function         | Returns | Description                                              |
|--------------------|---------|----------------------------------------------------------------|
| `os.platform()`    | string  | `"linux"`, `"darwin"`, `"windows"`, `"android"`                 |
| `os.arch()`        | string  | `"amd64"`, `"arm64"`, etc.                                      |
| `os.hostname()`    | string  | Machine hostname                                                |
| `os.cpus()`        | number  | Number of logical CPUs                                         |
| `os.sep`           | string  | OS path separator (`"/"` or `"\"`)                              |
| `os.pathSep`       | string  | OS path-list separator (`:` or `;`)                             |
| `os.eol`           | string  | Line ending Lunex uses (`"\n"`)                                 |
| `os.homeDir`       | string  | Current user's home directory                                   |

`sep`, `pathSep`, `eol`, and `homeDir` are plain values, not functions.

### Working directory

| Function         | Returns | Description                        |
|--------------------|---------|------------------------------------------|
| `os.cwd()`         | string  | Alias for `os.getcwd()`                   |
| `os.getcwd()`      | string  | Current working directory                 |
| `os.chdir(path)`   | —       | Change working directory                  |

### Environment variables

| Function                    | Returns         | Description                                    |
|-------------------------------|------------------|------------------------------------------------------|
| `os.getenv(key)`               | string \| null   | Read environment variable                              |
| `os.setenv(key, value)`        | —                | Write an environment variable                          |
| `os.unsetenv(key)`             | —                | Remove an environment variable                         |
| `os.environ()`                 | object           | All environment variables as an object                 |
| `os.expandEnv(s)`               | string           | Expand `$VAR` and `${VAR}` in a string                  |

### Shell execution

| Function                  | Returns | Description                          |
|------------------------------|---------|--------------------------------------------|
| `os.exec(cmd, opts?)`         | object  | Run a command synchronously                 |
| `os.execSync(cmd, opts?)`     | object  | Alias for `os.exec`                         |
| `os.spawn(cmd, opts?)`        | object  | Run a command in the background             |

`os.exec` returns `{ stdout, stderr, code, ok }`.
`os.spawn` returns `{ pid, wait(), kill() }` — `wait()` blocks until the
process exits and returns its exit code (a number), and does not capture
`stdout`/`stderr`.

Optional opts object: `{ cwd, env, timeout }`.

> **Known limitation:** `cmd` is split into arguments by whitespace only
> (there's no shell involved). Quoted arguments containing spaces, `&&`,
> pipes, globbing, and other shell syntax are **not** interpreted — they're
> passed through literally as part of the split tokens. For anything beyond
> a simple `program arg1 arg2` command, invoke a shell explicitly, e.g.
> `os.exec("sh -c \"...\"")` on Unix.

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
|------------------------------|---------|-------------------------------------------------|
| `os.join(...parts)`           | string  | Join path segments                                |
| `os.dirname(path)`            | string  | Parent directory of a path                        |
| `os.basename(path)`           | string  | File name portion of a path                       |
| `os.extname(path)`            | string  | File extension, including the leading `.`         |
| `os.abs(path)`                | string  | Absolute path                                     |
| `os.stat(path)`               | object  | `{ name, size, isDir, isFile, mode, modTime }`    |
| `os.exists(path)`             | boolean | True if path exists                               |
| `os.mkdir(path)`              | —       | Create directory and all parents                  |
| `os.remove(path)`             | —       | Delete a file or empty directory                  |
| `os.rename(src, dst)`         | —       | Rename or move a path                             |
| `os.listDir(path)`            | array   | List directory entries                            |
| `os.glob(pattern)`            | array   | Expand a glob pattern                             |
| `os.tempDir()`                | string  | Path to a system temporary directory              |
| `os.tempFile()`               | string  | Path to a new temporary file                      |

### Timing

| Function           | Returns | Description                                        |
|-----------------------|---------|------------------------------------------------------------|
| `os.time()`           | number  | Current Unix time in milliseconds                             |
| `os.hrtime()`         | number  | High-resolution monotonic-ish timestamp in milliseconds       |
| `os.sleep(ms)`        | —       | Pause execution for `ms` milliseconds                         |

---

## `std.regex` — Regular expressions

```lx
val regex = @import("std.regex")
```

Uses Go's RE2 syntax (no lookaheads or backreferences).

### Compiling

| Function                        | Returns | Description                                        |
|-------------------------------------|---------|---------------------------------------------------------|
| `regex.compile(pattern, flags?)`    | regex   | Precompile a pattern into a reusable regex value        |

```lx
val re = regex.compile("\\d+", "i")
```

### Flags support

Flags (e.g. `"i"` for case-insensitive) are only accepted by `regex.test`,
`regex.match`, `regex.matchAll`, `regex.groups`, `regex.groupsAll`, and
`regex.replace`. **`regex.replaceAll`, `regex.replaceFunc`, `regex.split`,
`regex.namedGroups`, `regex.isValid`, `regex.count`, `regex.index`, and
`regex.indices` do not take a flags parameter at all** — inline flags
(e.g. `(?i)` at the start of the pattern) are the only way to affect
case-sensitivity for those functions. A `regex.compile(pattern, flags)`
value carries its flags with it and works consistently everywhere it's
accepted.

### Testing

| Function                   | Returns | Description                               |
|----------------------------|---------|-------------------------------------------|
| `regex.test(s, pattern, flags?)` | boolean | True if pattern matches anywhere in s |
| `regex.isValid(pattern)`   | boolean | True if pattern is valid RE2 syntax       |

### Matching

| Function                     | Returns        | Description                                  |
|------------------------------|----------------|----------------------------------------------|
| `regex.match(s, pattern, flags?)`    | string \| null | First matching substring              |
| `regex.matchAll(s, pattern, flags?)` | array          | All non-overlapping matches           |
| `regex.index(s, pattern)`    | number         | Start index of first match (−1 if none)      |
| `regex.indices(s, pattern)`  | array          | Start indices of all matches                 |
| `regex.count(s, pattern)`    | number         | Number of non-overlapping matches            |

### Capture groups

| Function                         | Returns | Description                                   |
|----------------------------------|---------|-----------------------------------------------|
| `regex.groups(s, pattern, flags?)`    | array   | Capture groups from the first match      |
| `regex.groupsAll(s, pattern, flags?)` | array   | Capture groups from every match          |
| `regex.namedGroups(s, pattern)`  | object  | Named capture groups as an object             |

### Replacement

| Function                              | Returns | Description                     |
|---------------------------------------|---------|---------------------------------|
| `regex.replace(s, pattern, repl, flags?)` | string | Replace **all** matches      |
| `regex.replaceAll(s, pattern, repl)`  | string  | Replace all matches — identical behavior to `regex.replace` |
| `regex.replaceFunc(s, pattern, fn)`   | string  | Replace every match with the output of `fn(match)` |

> **Known behavior:** despite the name, `regex.replace` does **not** stop
> after the first match — it replaces every match in the string, exactly
> like `regex.replaceAll`. There is currently no built-in way to replace
> only the first occurrence; work around it with `regex.replaceFunc` and a
> counter, or `regex.index` plus manual string slicing.

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

| Function                   | Returns              | Description                                                        |
|-----------------------------|-----------------------|---------------------------------------------------------------------|
| `env.get(key)`               | string \| undefined  | Read variable; undefined if not set                                  |
| `env.get(key, default)`      | string                | Read with a fallback default                                        |
| `env.set(key, value)`        | —                     | Write an environment variable                                       |
| `env.has(key)`                | boolean               | True if the variable is set                                         |
| `env.delete(key)`             | —                     | Unset an environment variable                                       |
| `env.all()`                    | object                | All environment variables as an object                              |
| `env.load(path?)`              | boolean               | Load a `.env` file into the process environment (default: `.env`); `false` if the file can't be read |
| `env.require(key)`             | string \| undefined  | Like `env.get`, but treats an empty string the same as unset         |
| `env.int(key, default?)`       | number                | Parse as a number; `default` (or `0`) if unset or unparsable        |
| `env.bool(key, default?)`      | boolean               | Parse as a boolean (`"true"`, `"1"`, `"yes"`, `"on"` are true); `default` (or `false`) if unset |

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

There is no `runtime.typeOf()` or `runtime.gc()` in the `runtime` module.
To get the type name of a value, use the global `typeof(v)` keyword
described in `language-reference.md` — it's a language construct, not a
`runtime` module function. There is no way to force a garbage-collection
pass from Lunex code.
