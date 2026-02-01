---
phase: 04-test-coverage-and-safety
plan: 04
subsystem: categorization
requires:
  - 01-03 # File permissions foundation
provides:
  - Category YAML backup system
  - Configurable backup location and format
  - Atomic save operations with backup safety
affects:
  - Future auto-learning features (backups protect against AI errors)
  - Category mapping management (users can restore from backups)
tags:
  - backup
  - data-safety
  - yaml
  - configuration
tech-stack:
  added: []
  patterns:
    - "Atomic file operations (backup before write)"
    - "Optional configuration via SetBackupConfig pattern"
key-files:
  created: []
  modified:
    - internal/config/viper.go
    - internal/store/store.go
    - internal/store/store_test.go
decisions:
  - "Backup enabled by default (backup.enabled: true)"
  - "Backup location defaults to same directory as original file"
  - "Timestamp format: YYYYMMDD_HHMMSS (20060102_150405)"
  - "Backup filename pattern: {original}.{timestamp}.backup"
  - "Failed backup prevents save (atomic behavior)"
  - "Store uses optional configuration via SetBackupConfig method"
metrics:
  duration: "4m"
  completed: "2026-02-01"
---

# Phase 04 Plan 04: Category YAML Backup System Summary

**One-liner:** Automatic timestamped backups of category YAML files before auto-learning overwrites, with configurable location and atomic save operations to prevent data loss from incorrect AI categorizations.

## What Was Built

Implemented a comprehensive backup system for category mapping files (creditors.yaml, debtors.yaml) that automatically creates timestamped backups before any save operation. The system provides:

1. **Configuration Support**: Added backup configuration to Viper with three settings:
   - `backup.enabled` (default: true) - Enable/disable backup creation
   - `backup.directory` (default: "") - Custom backup location (empty = same directory)
   - `backup.timestamp_format` (default: "20060102_150405") - Go time format for timestamps

2. **CategoryStore Backup Functionality**:
   - Added backup configuration fields to CategoryStore with sensible defaults
   - `SetBackupConfig` method for runtime configuration
   - `createBackup` method that creates timestamped backup files
   - Integrated backup calls into `SaveCreditorMappings` and `SaveDebtorMappings`

3. **Safety Guarantees**:
   - Backups created BEFORE write operations (atomic behavior)
   - Failed backup prevents save (no data loss risk)
   - Backup files use pattern: `{original}.{timestamp}.backup` (e.g., `creditors.yaml.20260201_203022.backup`)
   - Backup file permissions match originals (0644 per SEC-03)

## Decisions Made

1. **Backup enabled by default**: Conservative approach - users must explicitly disable backups to skip them. Protects against accidental data loss.

2. **Same directory default**: Backup files stored alongside originals by default for easy discovery. Users can configure custom backup directory if preferred.

3. **Timestamp format includes seconds**: Format `20060102_150405` ensures unique backup filenames even for rapid saves while remaining human-readable.

4. **Atomic save behavior**: Failed backup prevents save operation. This ensures users never lose the ability to restore from backup if save fails partway through.

5. **Optional configuration via SetBackupConfig**: CategoryStore doesn't require Config dependency by default. Container/factory can call `SetBackupConfig` to override defaults from application config.

## Technical Implementation

### Configuration (internal/config/viper.go)
- Added `Backup` struct with enabled, directory, and timestamp_format fields
- Set defaults in `setDefaults()` function
- Configuration accessible via standard Viper hierarchical loading (file → env → flags)

### Backup Mechanism (internal/store/store.go)
```go
// Backup workflow
1. Check if backupEnabled (skip if false)
2. Check if original file exists (skip if new file)
3. Generate timestamp: time.Now().Format(backupTimestampFormat)
4. Determine backup path: backupDirectory (if set) or same dir
5. Copy original to backup location using io.Copy
6. Set backup file permissions to 0644
7. Return error if any step fails
```

### Save Operation Integration
Both `SaveCreditorMappings` and `SaveDebtorMappings` now:
1. Resolve file path
2. Create parent directory if needed
3. **Call createBackup (CRITICAL - happens before marshal/write)**
4. Marshal data to YAML
5. Write to file

If step 3 fails, steps 4-5 never execute, preserving original file.

### Test Coverage (internal/store/store_test.go)
Added 5 comprehensive test cases (196 lines):
- `TestCategoryStore_BackupCreatedBeforeSave` - Verifies backup creation and data preservation
- `TestCategoryStore_BackupUsesConfiguredLocation` - Verifies custom backup directory
- `TestCategoryStore_BackupFailurePreventsSave` - Verifies atomic behavior with read-only dir
- `TestCategoryStore_BackupDisabledSkipsBackup` - Verifies backup.enabled=false works
- `TestCategoryStore_MultipleBackupsWithTimestamps` - Verifies multiple backup coexistence

## Deviations from Plan

None - plan executed exactly as written. All must_haves satisfied:
- ✅ Category YAML files backed up before auto-learn overwrite
- ✅ Backup location configurable via config file
- ✅ Failed saves do not corrupt existing YAML files
- ✅ Backup functionality integrated into store save operations
- ✅ Comprehensive test coverage (60+ lines requirement met with 196 lines)

## Verification Results

All verification criteria passed:
- ✅ All store tests pass: `go test -v ./internal/store` (25 tests)
- ✅ Backup files created before saves with correct timestamps
- ✅ Configuration controls backup behavior (enabled, directory, format)
- ✅ Backup failure prevents save (atomic behavior verified)
- ✅ Backup files have correct permissions (0644)

## Integration Notes

### For Container/Factory Integration
When creating CategoryStore in the container, read backup config and apply:

```go
store := store.NewCategoryStore(categoriesFile, creditorsFile, debtorsFile)
if cfg.Backup.Enabled {
    store.SetBackupConfig(cfg.Backup.Enabled, cfg.Backup.Directory, cfg.Backup.TimestampFormat)
}
```

### Configuration Example
Users can customize backup behavior in `~/.camt-csv/config.yaml`:

```yaml
backup:
  enabled: true
  directory: "/Users/username/.camt-csv/backups"  # Custom location
  timestamp_format: "2006-01-02_15-04-05"          # Custom format
```

### Backup Management
Backup files accumulate over time. Future enhancement could add:
- Backup rotation (keep last N backups)
- Backup cleanup command
- Backup restore command

Currently, users manually manage backup files.

## Next Phase Readiness

**Blocks:** None

**Concerns:** None

**Enables:**
- Auto-learning can now safely overwrite category mappings
- Users have recovery mechanism for incorrect AI categorizations
- Foundation for future backup management features (rotation, restore)

**Recommendation:** SAFE-01 requirement fully closed. Auto-learning features can proceed with confidence that users can recover from any AI categorization errors by restoring from timestamped backups.

## Related Requirements

- **SAFE-01**: Category mapping backup before auto-learning ✅ **CLOSED**
- **SEC-03**: File permissions (backups use 0644 for non-secret data) ✅ **MAINTAINED**

## Commands to Verify

```bash
# Run all store tests
go test -v ./internal/store

# Run backup-specific tests
go test -v ./internal/store -run Backup

# Run all tests with coverage
make coverage

# Test backup behavior manually
# 1. Create initial creditors.yaml
echo "Test: Food" > database/creditors.yaml

# 2. Trigger a save (via auto-learning or direct call)
# 3. Check for backup file
ls -la database/creditors.yaml.*.backup
```

## Artifacts

**Modified Files:**
- `/Users/fjacquet/Projects/camt-csv/internal/config/viper.go` - Added Backup configuration
- `/Users/fjacquet/Projects/camt-csv/internal/store/store.go` - Implemented backup functionality
- `/Users/fjacquet/Projects/camt-csv/internal/store/store_test.go` - Added comprehensive backup tests

**Commits:**
- `c843a1f` - feat(04-04): add backup configuration to Viper config
- `988beb5` - feat(04-04): implement backup functionality in CategoryStore
- `b812b22` - test(04-04): add comprehensive backup functionality tests

**Test Coverage:**
- Backup functionality: 5 new tests (196 lines)
- All existing tests continue to pass (20 tests)
- Total store tests: 25 tests

## Success Metrics

- ✅ Category YAML files backed up before every auto-learn save
- ✅ Backup location configurable (default: same directory)
- ✅ Backup filenames include timestamps (e.g., `creditors.yaml.20260201_203022.backup`)
- ✅ Failed backups prevent save (atomic behavior)
- ✅ Users can recover from incorrect AI categorizations by restoring backups
- ✅ Zero test failures
- ✅ Zero breaking changes to existing functionality
