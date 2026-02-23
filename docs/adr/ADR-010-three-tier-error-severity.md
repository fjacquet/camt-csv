# ADR-010: Three-Tier Error Severity Pattern

## Status
Accepted

## Context

As the codebase grew across multiple parsers and subsystems, error handling was inconsistent. Some errors caused immediate `os.Exit`, others were silently swallowed, and callers had no reliable way to distinguish transient failures (worth retrying) from permanent ones (not worth retrying) from data errors (worth skipping and continuing).

This made batch processing unreliable: one bad file could crash the entire run, or conversely, a real infrastructure problem could be silently ignored.

## Decision

Adopt a documented three-tier error severity model applied consistently across all parsers and subsystems:

| Tier | Type | Meaning | Behaviour |
|------|------|---------|-----------|
| 1 | **Fatal** | Unrecoverable — cannot proceed | Log and exit; no retry |
| 2 | **Retryable** | Transient — may succeed on next attempt | Log warning; retry with backoff |
| 3 | **Recoverable** | Bad input — skip this item, continue others | Log warning; skip file/transaction |

Custom error types (`ParseError`, `ValidationError`, etc.) carry a `Severity` field so callers can switch on tier without string matching.

### Implementation

- `internal/models/errors.go` — error types with severity constants
- `cmd/common/convert.go` — central error handling applying the tier logic
- `internal/categorizer/` — AI errors classified as Retryable; YAML errors as Recoverable
- All parsers return typed errors instead of raw `fmt.Errorf`

## Consequences

**Positive:**
- Batch processing continues past recoverable per-file errors
- Transient API errors trigger automatic retry instead of permanent failure
- Fatal misconfiguration surfaces immediately with a clear message
- Error handling is auditable — each error has an explicit tier assignment

**Negative:**
- All new code must classify errors at the point of creation
- Misclassification (e.g., marking a logic bug as Retryable) can cause infinite retry loops

## Future Work

- Add tier-aware metrics/telemetry once observability tooling is added
- Consider per-tier configurable retry counts (currently hardcoded backoff for Retryable)
