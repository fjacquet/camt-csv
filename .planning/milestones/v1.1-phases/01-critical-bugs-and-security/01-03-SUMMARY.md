---
phase: 01-critical-bugs-and-security
plan: 03
subsystem: security
tags: [security, credentials, file-permissions, temp-files, gemini-api]

# Dependency graph
requires:
  - phase: 01-critical-bugs-and-security
    provides: codebase foundation
provides:
  - API credential security policy documented and verified
  - Random unpredictable temp file naming for PDF extraction
  - Standardized file permissions based on content type (0600 secrets, 0644 non-secrets, 0750 directories)
affects: [all future development - security foundation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Security comments documenting credential handling
    - File permission constants in models package
    - Random temp file naming with os.CreateTemp

key-files:
  created:
    - internal/models/constants.go: PermissionNonSecretFile constant added
  modified:
    - internal/categorizer/gemini_client.go: API credential security documentation
    - internal/pdfparser/pdfparser_helpers.go: Random temp file naming
    - internal/store/store.go: Corrected file permissions for non-secret files
    - internal/pdfparser/pdfparser.go: Use PermissionDirectory constant

key-decisions:
  - "API credentials must never appear in logs at any level"
  - "Temp files must use random unpredictable names (os.CreateTemp)"
  - "File permissions based on content: 0600 secrets, 0644 non-secrets, 0750 dirs"
  - "Creditor/debtor mappings are non-secret (just category mappings), use 0644"

patterns-established:
  - "SECURITY comments documenting credential handling policies"
  - "Permission constants in models package for consistency"
  - "os.CreateTemp with defer cleanup for temp files"

# Metrics
duration: 3min
completed: 2026-02-01
---

# Phase 1 Plan 3: Security Hardening Summary

**API credential protection, random temp file naming, and content-based file permissions (0600/0644/0750) standardized across codebase**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-01T17:42:18Z
- **Completed:** 2026-02-01T17:45:09Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments
- Documented and verified API credential security policy - no logging of apiKey or URLs containing credentials
- Replaced predictable temp file naming with os.CreateTemp for security
- Standardized file permissions with new PermissionNonSecretFile (0644) constant
- Corrected creditor/debtor mapping permissions from 0600 to 0644 (non-secret data)

## Task Commits

Each task was committed atomically:

1. **Task 1: Audit API credential logging** - `26fb19b` (fix)
   - Add comprehensive security documentation to GeminiClient struct
   - Document no-credential-logging policy for apiKey field
   - Add security warnings on URL construction

2. **Task 2: Random temp file naming** - `306f048` (fix)
   - Replace predictable pdfFile + ".txt" with os.CreateTemp
   - Add defer cleanup and close temp file before pdftotext writes

3. **Task 3: Standardize file permissions** - `51d819f` (fix)
   - Add PermissionNonSecretFile (0644) constant
   - Update creditor/debtor mappings to use 0644 instead of 0600
   - Update pdfparser.go to use PermissionDirectory constant

## Files Created/Modified
- `internal/categorizer/gemini_client.go` - Added SECURITY documentation for credential handling
- `internal/pdfparser/pdfparser_helpers.go` - Random temp file naming with os.CreateTemp
- `internal/models/constants.go` - Added PermissionNonSecretFile (0644) constant with security comments
- `internal/store/store.go` - Corrected creditor/debtor mapping permissions to 0644
- `internal/pdfparser/pdfparser.go` - Use PermissionDirectory constant instead of hardcoded 0750

## Decisions Made

**API Credential Security:**
- Documented explicit no-logging policy for apiKey field and URLs containing credentials
- Only response bodies (which don't contain credentials) may be logged for debugging
- Added SECURITY comments at struct level and URL construction points

**Temp File Security:**
- Use os.CreateTemp for unpredictable random naming to prevent attacks relying on predictable filenames
- Close file immediately after creation so pdftotext can write to it
- Defer cleanup ensures removal even on error paths

**File Permission Policy:**
- 0600 (PermissionConfigFile): Secret files containing credentials or API keys
- 0644 (PermissionNonSecretFile): Non-secret files like YAML categories, CSV exports, debug files
- 0750 (PermissionDirectory): Standard directory permissions
- Creditor/debtor mappings are non-secret (just merchant→category mappings), use 0644 not 0600

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all security improvements implemented cleanly without conflicts.

## Next Phase Readiness

Security foundation is now hardened:
- API credentials protected from logging at all levels
- Temp files use unpredictable names
- File permissions appropriate for content sensitivity

Ready for continued phase 1 work on remaining critical bugs and security issues.

---
*Phase: 01-critical-bugs-and-security*
*Completed: 2026-02-01*
