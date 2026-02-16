---
phase: 06-revolut-parsers-overhaul
plan: 01
subsystem: models
tags: [transaction-model, builder-pattern, csv-export, revolut]
dependency_graph:
  requires: [phase-05-output-framework]
  provides: [product-field-support]
  affects: [revolut-parsers, csv-output]
tech_stack:
  added: []
  patterns: [builder-pattern]
key_files:
  created: []
  modified:
    - internal/models/transaction.go
    - internal/models/builder.go
    - internal/common/csv.go
    - internal/formatter/standard.go
    - internal/camtparser/camtparser_test.go
    - internal/formatter/formatter_test.go
    - internal/models/transaction_test.go
decisions:
  - "Product field positioned after Currency in Transaction struct for logical grouping"
  - "CSV format expanded from 34 to 35 columns to include Product field"
  - "No validation on Product field values - accepts any string from source data"
metrics:
  duration: 496
  completed: 2026-02-16T05:46:13Z
---

# Phase 06 Plan 01: Add Product Field to Transaction Model Summary

**One-liner:** Add Product field to Transaction model and TransactionBuilder for Revolut account routing (Current/Savings)

## Overview

Successfully added Product field support to the Transaction model and builder pattern, enabling Revolut account routing based on product type (Current vs Savings). Updated all related CSV output infrastructure and tests to support the new 35-column format.

## What Was Delivered

### Task 1: Add Product field to Transaction model
- ✅ Added `Product string` field with `csv:"Product"` tag after Currency field (line 30)
- ✅ Updated `MarshalCSV()` to include Product in CSV output at position 15
- ✅ Updated `UnmarshalCSV()` to parse Product from CSV input at position 15
- ✅ Positioned after Currency for logical grouping of account-level fields

**Commit:** `172b591`

### Task 2: Add WithProduct() method to TransactionBuilder
- ✅ Added `WithProduct(product string)` method to TransactionBuilder (line 302 in builder.go)
- ✅ Follows fluent API pattern consistent with other builder methods
- ✅ No validation - accepts any string value from source data
- ✅ Updated CSV header in `WriteTransactionsToCSVWithLogger` to include Product
- ✅ Updated `StandardFormatter.Header()` to return 35 columns
- ✅ Fixed all test expectations for 35-column format

**Commit:** `9891337`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated test CSV record format**
- **Found during:** Task 2 verification
- **Issue:** `TestTransaction_DateFormattingIndirect/CSV_unmarshaling_with_dates` failed because test CSV record had 34 fields but Product field added 35th field
- **Fix:** Inserted empty Product field at position 15 in test CSV data, adjusted field order to match struct
- **Files modified:** `internal/models/transaction_test.go`
- **Commit:** `9891337`

**2. [Rule 3 - Blocking] Updated camtparser test expectations**
- **Found during:** Task 2 verification
- **Issue:** `TestConvertToCSV` failed because expected CSV lacked Product column in header
- **Fix:** Added "Product" to expected CSV header string at position 15
- **Files modified:** `internal/camtparser/camtparser_test.go`
- **Commit:** `9891337`

**3. [Rule 3 - Blocking] Updated formatter tests and StandardFormatter**
- **Found during:** Task 2 verification
- **Issue:** StandardFormatter tests failed because Header() returned 34 columns instead of 35, and Category field index shifted
- **Fix:** Updated StandardFormatter.Header() to include Product at position 15, updated test assertions for 35 columns, adjusted Category index from 21 to 22
- **Files modified:** `internal/formatter/standard.go`, `internal/formatter/formatter_test.go`
- **Commit:** `9891337`

**4. [Rule 3 - Blocking] Updated CSV header in WriteTransactionsToCSVWithLogger**
- **Found during:** Task 2 verification
- **Issue:** `WriteTransactionsToCSVWithLogger` had hardcoded header array missing Product field
- **Fix:** Added "Product" to hardcoded header array at position 15 (after Currency)
- **Files modified:** `internal/common/csv.go`
- **Commit:** `9891337`

## Verification Results

✅ Product field exists in Transaction struct with proper csv tag (line 30)
✅ WithProduct() method exists in TransactionBuilder (line 302)
✅ All tests pass: `make test` - 100% pass rate
✅ Build succeeds: `make build` - no errors
✅ CSV format updated from 34 to 35 columns across entire codebase

## Technical Notes

**CSV Column Order:**
The Product field was inserted at position 15 (0-indexed 14) in the CSV output:
- Position 14: Currency
- **Position 15: Product (NEW)**
- Position 16: AmountExclTax

This required updating:
1. Transaction.MarshalCSV() method (manual string array construction)
2. Transaction.UnmarshalCSV() method (manual parsing with shifted indices)
3. WriteTransactionsToCSVWithLogger() hardcoded header array
4. StandardFormatter.Header() method
5. All test expectations referencing field positions

**No Breaking Changes:**
- Existing parsers will write empty Product field values
- UnmarshalCSV handles empty Product fields gracefully
- Product field is optional - no validation or required constraints

## Impact Assessment

**Affected Components:**
- Transaction model (core data structure)
- TransactionBuilder (fluent API)
- CSV output infrastructure (common package)
- Standard formatter (output formatting)
- All tests validating CSV structure

**Downstream Dependencies:**
- Phase 06-02: Revolut current account parser can now use WithProduct("Current")
- Phase 06-03: Revolut savings parser can now use WithProduct("Savings")
- iCompta import: Account routing logic will use Product field for Revolut transactions

## Self-Check: PASSED

**Created files:** None (only modifications)

**Modified files verified:**
- ✅ FOUND: internal/models/transaction.go (Product field at line 30)
- ✅ FOUND: internal/models/builder.go (WithProduct() at line 302)
- ✅ FOUND: internal/common/csv.go (header updated)
- ✅ FOUND: internal/formatter/standard.go (header updated)

**Commits verified:**
- ✅ FOUND: 172b591 (Task 1: Product field)
- ✅ FOUND: 9891337 (Task 2: WithProduct() + test fixes)

All planned deliverables exist. All commits found. Build succeeds. Tests pass.

---

**Plan completed:** 2026-02-16T05:46:13Z
**Execution time:** 496 seconds (~8.3 minutes)
**Task completion:** 2/2 (100%)
