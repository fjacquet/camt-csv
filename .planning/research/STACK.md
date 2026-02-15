# Technology Stack

**Project:** camt-csv — Go CLI financial converter with Revolut enhancements
**Researched:** 2026-02-15
**Recommendation:** Minimal new dependencies; use existing ecosystem strategically

---

## Recommended Stack

### Core Frameworks (Existing, Proven)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| **Go** | 1.21+ | CLI tool, financial calculations | Compiled binary; no runtime deps; excellent stdlib (encoding/csv, sync, errors) |
| **Cobra** | v1.7+ | CLI framework | Already integrated; structured subcommand handling |
| **Viper** | v1.16+ | Configuration management | Already integrated; hierarchical config (file → env → flags) |
| **decimal.Decimal** (shopspring) | v1.3+ | Financial precision | Already in use; avoids float precision loss; essential for rounding |
| **gocsv** (gocarina) | Latest | CSV unmarshaling | Already in use; struct-tag based parsing; handles Revolut format |
| **logrus** (sirupsen) | v1.9+ | Logging | Already integrated; structured logging with fields |

### New Dependencies (Minimal, Justified)

| Technology | Version | Purpose | When | Justification |
|------------|---------|---------|------|---|
| **fslock** (theckman) | v0.8.1+ | File locking | Phase 3 (Batch) | Prevents YAML corruption; no stdlib equivalent; portable |
| **errgroup** | stdlib | Concurrent error handling | Phase 3 (Batch) | Included in Go stdlib; handles worker pool + first error propagation |

### Supporting Libraries (No Additional Dependencies)

| Library | Purpose | Already Used? |
|---------|---------|---|
| `time` | Date parsing (DD.MM.YYYY format) | Yes |
| `math` | Rounding functions (banker's rounding) | Covered by decimal.Decimal |
| `sync` | Mutexes for concurrent YAML writes | Yes (patterns established) |
| `io/ioutil` | File I/O | Yes |
| `github.com/google/uuid` | Transaction IDs | Yes (compliance reports) |

---

## Dependency Justification Summary

| Dep | Why Add | Why Not Remove |
|-----|---------|---|
| decimal.Decimal | Essential for financial precision | Removes entire category of rounding errors |
| gocsv | Simplifies CSV struct mapping | Alternatives are less idiomatic Go |
| theckman/go-flock | Prevents YAML corruption under concurrency | No stdlib equivalent; critical for data integrity |
| errgroup | Built into Go; no added weight | Simplifies worker pool + error handling |

---

## File Locking Strategy (Phase 3)

**Recommendation:** Use `github.com/theckman/go-flock` for robust, portable file locking

**Why chosen:**
- File-level locking (better than directory tricks)
- Portable across POSIX/Windows
- Small, focused library
- Production-tested

---

## Installation

```bash
# New dependencies for Phase 3+
go get github.com/theckman/go-flock@v0.8.1
```

All other dependencies already present in go.mod.

---

## Sources

- [Go stdlib documentation](https://golang.org/pkg/)
- [shopspring/decimal — precision for Go](https://github.com/shopspring/decimal)
- [Cobra CLI framework](https://github.com/spf13/cobra)
- [Viper configuration](https://github.com/spf13/viper)
- [theckman/go-flock — portable file locking](https://github.com/theckman/go-flock)
