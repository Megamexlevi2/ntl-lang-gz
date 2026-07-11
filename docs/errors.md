# Error Reference

Lunex assigns a code to every diagnostic it can detect. This document lists
all codes, their meaning, and typical fixes.

Error codes follow the pattern `E####` for hard errors.

---

## How to read an error

```
✗ error[E0021] UndefinedVariable: 'usr' is not defined
  ──▶ main.lx:12:3

 10 │
 11 │  fn greet(user) {
 12 │    io.log(usr.name)
    │           ^^^
    │           ╰── did you mean 'user'?
```

| Field         | Meaning                                        |
|---------------|------------------------------------------------|
| `E0021`       | Error code                                     |
| `UndefinedVariable` | Error class                              |
| `main.lx:12:3`| File, line, column                             |
| Source window | Up to 5 lines of context                       |
| `^^^`         | Underline pointing at the problem              |
| Suggestion    | Automatic fix hint (where available)           |

---

## Syntax errors (E0001–E0020)

| Code  | Name                  | Common cause                                                       |
|-------|-----------------------|--------------------------------------------------------------------|
| E0001 | UnexpectedToken       | A token appeared where it was not expected                         |
| E0002 | UnexpectedEOF         | File ended before a statement was complete                         |
| E0003 | MissingCloseParen     | `)` not found after argument list                                  |
| E0004 | MissingCloseBrace     | `}` not found after block                                          |
| E0005 | MissingCloseBracket   | `]` not found after array or index                                 |
| E0006 | InvalidAssignment     | Left-hand side of `=` is not assignable                            |
| E0007 | InvalidOperator       | Unrecognized operator                                              |
| E0008 | DuplicateKey          | Object literal has duplicate keys                                  |
| E0009 | MissingArrow          | `=>` expected in match arm                                         |
| E0010 | InvalidDestructure    | Destructuring pattern is not an object or array literal            |
| E0011 | MissingElse           | `guard` statement is missing its `else` block                      |
| E0012 | InvalidImport         | `@import` or `@fimport` argument is not a string literal           |
| E0013 | InvalidSpawn          | `spawn` must be followed by a function call expression             |
| E0014 | InvalidDefer          | `defer` must be followed by a block `{ }`                          |
| E0015 | InvalidMatch          | `match` expression has no arms                                     |
| E0016 | MissingIn             | `each` is missing the `in` keyword                                 |
| E0017 | InvalidTemplateString | Unclosed template literal                                          |
| E0018 | InvalidEscape         | Unknown escape sequence in string literal                          |
| E0019 | InvalidNumber         | Number literal is malformed                                        |
| E0020 | NoReturnKeyword       | `return` is not a keyword in Lunex — the last expression is the result |

---

## Name resolution errors (E0021–E0035)

| Code  | Name                 | Common cause                                                |
|-------|----------------------|-------------------------------------------------------------|
| E0021 | UndefinedVariable    | Variable used before declaration                            |
| E0022 | UndefinedField       | Field access on an object that does not have that field     |
| E0023 | UndefinedMethod      | Method call on a struct that does not have that method      |
| E0024 | UndefinedModule      | `@import` target does not exist                             |
| E0025 | UndefinedPackage     | Installed package not found                                 |
| E0026 | ImmutableAssignment  | Attempting to reassign a `val` binding                      |
| E0027 | ShadowedBinding      | New binding shadows an existing one in the same scope       |
| E0028 | CircularImport       | `@fimport` introduces a circular dependency                 |
| E0029 | MissingMain          | File has no `fn main()` declaration                         |
| E0030 | DuplicateDeclaration | Two `fn` or `val`/`var` bindings share a name in same scope |

---

## Runtime errors (E0036–E0060)

| Code  | Name              | Common cause                                                  |
|-------|-------------------|---------------------------------------------------------------|
| E0036 | TypeError         | Operation applied to the wrong type (e.g. arithmetic on a string) |
| E0037 | NullAccess        | Field or method access on `null`                              |
| E0038 | IndexOutOfRange   | Array index outside the valid range                           |
| E0039 | DivisionByZero    | Integer division or modulo by zero                            |
| E0040 | StackOverflow     | Call depth exceeded (likely infinite recursion)               |
| E0041 | ChannelClosed     | `recv` or `send` on a closed channel                          |
| E0042 | FileNotFound      | `fs.readFile` path does not exist                             |
| E0043 | PermissionDenied  | File operation not permitted by the OS                        |
| E0044 | NetworkError      | HTTP or WebSocket connection failed                           |
| E0045 | JSONParseError    | Invalid JSON passed to a parse function                       |
| E0046 | RegexError        | Invalid regular expression pattern                            |
| E0047 | CryptoError       | Encryption/decryption failed (bad key length, etc.)           |
| E0048 | DBError           | Database operation failed                                     |
| E0049 | JWTError          | JWT signing or verification failed                            |
| E0050 | ExecError         | `os.exec` command exited with non-zero status                 |
| E0051 | ImportError       | Module failed to load or initialize                           |
| E0052 | CastError         | Explicit type conversion failed                               |
| E0053 | KeyError          | Object key not found                                          |
| E0054 | TimeoutError      | Operation exceeded its time limit                             |
| E0055 | AssertionError    | An explicit assertion failed                                  |
| E0056 | MemoryError       | Program ran out of memory                                     |
| E0057 | RecursionError    | Maximum call depth exceeded                                   |
| E0058 | RuntimePanic      | An internal error in the Lunex runtime                        |
| E0059 | InvalidArgument   | Function received an argument outside its expected range      |
| E0060 | UnreachableCode   | Execution reached code that should be impossible              |

---

## Compiler errors (E0061–E0071)

| Code  | Name                    | Common cause                                                    |
|-------|-------------------------|-----------------------------------------------------------------|
| E0061 | BuildError              | `lunex build` failed                                            |
| E0062 | ArchiveVersionMismatch  | `.nax` archive compiled with an incompatible Lunex version     |
| E0063 | ArchiveCorrupt          | `.nax` file is malformed or incomplete                          |
| E0064 | CacheCorrupt            | Bytecode cache entry is corrupt (`lunex memcache clear` to fix) |
| E0065 | TimeoutError            | Operation exceeded its deadline                                 |
| E0066 | NativeLoadError         | Native extension failed to load                                 |
| E0067 | PackageInstallError     | Package download or compilation failed                          |
| E0068 | ManifestError           | `config.lx` is malformed                                        |
| E0069 | EmbedError              | Embedded resource failed to initialize                          |
| E0070 | PlatformUnsupported     | Feature not supported on this OS/arch combination               |
| E0071 | InternalError           | Unexpected compiler error — please file a bug report            |

---

## Filing a bug

If you hit `E0058 RuntimePanic` or `E0071 InternalError`, please open an issue at:

```
https://github.com/Megamexlevi2/lunex-language/issues
```

Include:
- Lunex version (`lunex version`)
- Platform info (`lunex platform`)
- A minimal `.lx` file that reproduces the issue
- The complete error output
