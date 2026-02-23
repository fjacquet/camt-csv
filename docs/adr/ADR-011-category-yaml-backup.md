# ADR-011: Category YAML Backup Before Auto-Learn Writes

## Status
Accepted

## Context

The AI categorization pipeline can write newly learned mappings directly back to `database/creditors.yaml` and `database/debtors.yaml` via the `--auto-learn` flag. These YAML files are the user's curated categorization database — built up over years of use.

A single bad AI suggestion, a bug in the write path, or an interrupted write could silently corrupt or destroy this data with no way to recover. The files are tracked in git, but users don't necessarily commit frequently.

## Decision

Before any auto-learn write to a category YAML file, create a timestamped backup copy in the same directory:

```
database/creditors.yaml        ← live file (always current)
database/creditors.yaml.backup ← copy taken before last write
```

**Rules:**
- Backup is created atomically before every write (not just first write)
- Backup overwrites the previous backup (one rolling backup per file)
- Backup is enabled by default — no flag required to opt in
- Backup files are listed in `.gitignore` (`*.backup` pattern)
- Staging mode (default, without `--auto-learn`) writes to `database/staging_creditors.yaml` instead — no backup needed there since staging files are throwaway

### Why one rolling backup instead of timestamped history

A full history of backups grows unboundedly and most users never need more than "undo the last write." One rolling backup covers the 99% case without disk management complexity.

## Consequences

**Positive:**
- User can always recover the previous state with a file copy
- Auto-learn is safe to use without git discipline
- Backup is silent/transparent — no user friction

**Negative:**
- One rolling backup means a second bad write destroys both versions — users with risky AI integrations should also rely on git
- Adds one file write per auto-learn operation (negligible overhead)

## Future Work

- Consider opt-in versioned backup (N copies) for power users
- Could expose `--no-backup` flag if performance becomes a concern at scale
