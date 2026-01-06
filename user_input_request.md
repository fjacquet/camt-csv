# Property 8 Test Failure - Need User Direction

## Current Status

Task 5.2 (Property 8: Consistent CSV output format) is **FAILING**.

The test expects CSV headers that are not present in the current PDF parser output:
- Missing: `DebitAmount`, `CreditAmount`, `SubCategory`, `Payee`, `PayeeIBAN`, `Payer`, `PayerIBAN`, `RecipientIBAN`
- Present: Standard headers from `internal/common/csv.go`

## Options

**Option 1: Fix the failure**
- Update the PDF parser CSV format to include the missing headers
- Modify the common CSV format or PDF-specific format
- Ensure consistency across all parsers

**Option 2: I'll come back to this later**
- Mark the test as failing for now
- Continue with other tasks
- Address the format inconsistency in a future iteration

Which option would you prefer?