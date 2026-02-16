---
phase: 08-ai-safety-controls
plan: 03
subsystem: categorizer
tags:
  - ai
  - safety
  - user-control
  - auto-learn
  - configuration
dependency_graph:
  requires:
    - 08-01-confidence-metadata
  provides:
    - auto-learn-flag-control
    - safe-default-off
  affects:
    - cmd/root/root.go
    - internal/config/viper.go
    - internal/categorizer/categorizer.go
    - internal/container/container.go
tech_stack:
  added: []
  patterns:
    - feature-flag-gating
    - safe-defaults
    - hierarchical-configuration
key_files:
  created: []
  modified:
    - cmd/root/root.go
    - internal/config/viper.go
    - internal/config/viper_test.go
    - internal/categorizer/categorizer.go
    - internal/categorizer/*_test.go
    - internal/container/container.go
    - cmd/common/process.go
    - internal/integration/cross_parser_test.go
decisions:
  - "Auto-learn defaults to OFF (false) per v1.2 decision D-11"
  - "CLI flag provides explicit user control: --auto-learn"
  - "Configuration hierarchy: defaults → config file → env → CLI flag"
  - "Tests use auto-learn=true to maintain existing behavior"
  - "SaveMappings helper uses auto-learn=false (deprecated path)"
  - "DEBUG-level logging when auto-learning disabled but categorization succeeds"
metrics:
  duration: 480
  completed: "2026-02-16T08:15:31Z"
  tasks: 3
  files: 10
  commits: 3
---

# Phase 8 Plan 03: Auto-Learn Flag Control Summary

**One-liner:** --auto-learn flag implementation with safe default (OFF) provides user control over AI categorization auto-save behavior.

## Objectives Achieved

Implemented complete --auto-learn flag wiring from CLI through config to categorizer with safe default (OFF). This is the final safety gate preventing silent AI miscategorization, completing Phase 8 requirement AI-01 (user control over auto-learning).

## Implementation Details

### Task 1: Change auto-learn default to OFF and add CLI flag

**Modified files:**
- `internal/config/viper.go` - Changed default from `true` to `false` with comment "Default: OFF per v1.2 D-11"
- `cmd/root/root.go` - Added `--auto-learn` CLI flag with Viper binding to `categorization.auto_learn`
- `internal/config/viper_test.go` - Updated test assertion from `assert.True` to `assert.False` for default case

**Key changes:**
```go
// internal/config/viper.go
v.SetDefault("categorization.auto_learn", false) // Default: OFF per v1.2 D-11

// cmd/root/root.go
Cmd.PersistentFlags().Bool("auto-learn", false, "Enable AI auto-learning of categorizations (default: false)")
viper.BindPFlag("categorization.auto_learn", Cmd.PersistentFlags().Lookup("auto-learn"))
```

**Verification:**
- Config tests pass with new default
- Help text shows `--auto-learn` flag
- Grep confirms default changed to false

### Task 2: Add auto-learn control to Categorizer

**Modified files:**
- `internal/categorizer/categorizer.go` - Core gating logic
- Test files (7 files) - Updated all NewCategorizer() callsites

**Key changes:**

1. Added field to Categorizer struct:
```go
isAutoLearnEnabled bool // Controls whether AI categorizations are saved to YAML
```

2. Updated NewCategorizer signature:
```go
func NewCategorizer(aiClient AIClient, store CategoryStoreInterface, logger logging.Logger, autoLearnEnabled bool) *Categorizer
```

3. Gated save operations in both `Categorize()` and `CategorizeTransactionWithCategorizer()`:
```go
if err == nil && c.isAutoLearnEnabled && category.Name != "" && category.Name != models.CategoryUncategorized {
    // Auto-learning enabled: save mapping
    c.logger.Info("Auto-learning debitor/creditor mapping")
    c.updateDebitorCategory(partyName, category.Name)
    // ... save to YAML
} else if err == nil && !c.isAutoLearnEnabled && category.Name != "" && category.Name != models.CategoryUncategorized {
    // Auto-learning disabled: log but don't save
    c.logger.Debug("Categorization found but auto-learning disabled")
}
```

4. Updated all test callsites:
   - Most tests: `NewCategorizer(..., true)` - maintain existing auto-learn behavior
   - `cmd/common/process.go`: `NewCategorizer(..., false)` - deprecated helper function

**Verification:**
- All categorizer tests pass
- Signature includes autoLearnEnabled parameter
- Gating logic present in both methods

### Task 3: Wire auto-learn flag through container

**Modified file:**
- `internal/container/container.go`

**Key change:**
```go
cat := categorizer.NewCategorizer(aiClient, categoryStore, logger, cfg.Categorization.AutoLearn)
```

**Verification:**
- Container tests pass
- Grep confirms wiring with AutoLearn parameter
- End-to-end flow complete: CLI → Viper → Container → Categorizer

## Deviations from Plan

None - plan executed exactly as written.

## Testing

**Unit tests:**
- `internal/config/` - PASS (all tests pass with new default)
- `internal/categorizer/` - PASS (0.926s, all tests pass with updated signatures)
- `internal/container/` - PASS (all tests pass with new wiring)

**Full test suite:**
- `make test` - PASS (all packages pass)

**Verification criteria (from plan):**
- ✅ Auto-learn default changed from true to false in setDefaults()
- ✅ --auto-learn flag defined in cmd/root/root.go
- ✅ Viper binding connects flag to categorization.auto_learn config
- ✅ Categorizer has isAutoLearnEnabled field
- ✅ NewCategorizer accepts autoLearnEnabled parameter
- ✅ Categorize() only saves when isAutoLearnEnabled && category is valid
- ✅ Categorize() logs "auto_learn_disabled" when flag is off but categorization succeeds
- ✅ Container passes cfg.Categorization.AutoLearn to NewCategorizer
- ✅ All tests pass (including updated test expectations)

## Self-Check: PASSED

**Modified files exist:**
```
FOUND: cmd/root/root.go
FOUND: internal/config/viper.go
FOUND: internal/config/viper_test.go
FOUND: internal/categorizer/categorizer.go
FOUND: internal/container/container.go
```

**Commits exist:**
```
FOUND: c84fb18 (Task 1: --auto-learn flag with safe default)
FOUND: 1519a1a (Task 2: auto-learn control to Categorizer)
FOUND: f628394 (Task 3: wire flag through container)
```

**Must-have verification:**
- ✅ --auto-learn CLI flag exists and binds to categorization.auto_learn
- ✅ Default changed to false in viper.go with comment
- ✅ Categorizer struct has isAutoLearnEnabled field
- ✅ NewCategorizer signature includes autoLearnEnabled parameter
- ✅ Save operations gated by isAutoLearnEnabled check
- ✅ DEBUG logging when disabled
- ✅ Container wires cfg.Categorization.AutoLearn
- ✅ All tests pass with updated callsites

## Configuration Hierarchy Verification

The --auto-learn flag follows the standard hierarchical configuration pattern:

1. **Default (lowest priority):** `false` - set in `setDefaults()`
2. **Config file:** User can set `categorization.auto_learn: true` in config.yaml
3. **Environment variable:** `CAMT_CATEGORIZATION_AUTO_LEARN=true`
4. **CLI flag (highest priority):** `--auto-learn`

This ensures users have multiple ways to control the feature while maintaining a safe default.

## Impact

**Safety:** Auto-learning is now OFF by default, preventing silent AI miscategorizations from being saved automatically. Users must explicitly enable via flag, config, or env var.

**User experience:** Clear flag name (`--auto-learn`) and help text make the feature discoverable and understandable.

**Backward compatibility:** Tests maintain auto-learn=true to preserve existing test behavior. Production usage requires explicit opt-in.

**Logging:** DEBUG-level logs provide visibility when categorization succeeds but auto-learning is disabled, helpful for troubleshooting.

## Next Steps

Phase 8 Plan 03 completes the auto-learn control implementation. This was the final plan in Phase 8 (AI Safety Controls). The phase is now complete with all three plans:

- Plan 01: Confidence metadata infrastructure ✅
- Plan 02: Rate limiting and retry logic ✅
- Plan 03: Auto-learn flag control ✅

Next phase would be Phase 9 (if planned) or completion of v1.2 milestone.

## Key Learnings

1. **Safe defaults matter:** Starting with OFF ensures users won't accidentally pollute their category databases with AI miscategorizations.

2. **Complete wiring:** Following the full chain (CLI → Viper → Container → Component) ensures the flag works correctly across all configuration sources.

3. **Test isolation:** Using `true` in tests maintains existing test behavior while production gets the safe default.

4. **Debug logging strategy:** Logging at DEBUG level when disabled (vs INFO when enabled) provides visibility without noise in production logs.

5. **Signature evolution:** Adding a parameter to widely-used constructors requires careful coordination across test files but is straightforward with replace-all operations.
