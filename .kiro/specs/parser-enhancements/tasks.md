# Implementation Plan: Parser Enhancements

## Overview

This implementation plan breaks down the parser enhancements into discrete coding tasks that build incrementally. The approach focuses on adding account aggregation to batch processing and integrating categorization into PDF and Selma parsers while leveraging existing infrastructure.

## Tasks

- [x] 1. Create account identification utilities
  - Create `internal/common/account.go` with account extraction functions
  - Implement filename pattern parsing for CAMT files
  - Add account sanitization for filesystem safety
  - _Requirements: 6.1, 6.5, 7.3_

- [x] 1.1 Write property tests for account identification
  - **Property 13: Account identification from filenames**
  - **Validates: Requirements 6.1**

- [x] 2. Implement batch aggregation engine
  - Create `internal/batch/aggregator.go` with file grouping logic
  - Implement transaction aggregation and sorting
  - Add date range calculation and filename generation
  - _Requirements: 1.1, 1.2, 1.3, 7.1, 7.2_

- [x] 2.1 Write property tests for batch aggregation
  - **Property 1: Account-based file aggregation**
  - **Validates: Requirements 1.1**

- [x] 2.2 Write property tests for file naming
  - **Property 2: Consolidated file naming convention**
  - **Validates: Requirements 1.2, 7.1**

- [x] 2.3 Write property tests for transaction ordering
  - **Property 3: Chronological transaction ordering**
  - **Validates: Requirements 1.3**

- [x] 3. Enhance batch command with aggregation
  - Modify `cmd/batch/batch.go` to use new aggregation engine
  - Add account-based file grouping
  - Implement consolidated output file generation
  - _Requirements: 1.1, 1.4, 1.5, 4.2, 4.4_

- [x] 3.1 Write property tests for duplicate handling
  - **Property 4: Duplicate transaction preservation**
  - **Validates: Requirements 1.4**

- [x] 3.2 Write property tests for source file metadata
  - **Property 5: Source file metadata inclusion**
  - **Validates: Requirements 1.5**

- [x] 4. Checkpoint - Ensure batch aggregation tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Add categorization to PDF parser
  - Modify `internal/pdfparser/pdfparser.go` to implement `CategorizerConfigurable`
  - Add categorizer field and SetCategorizer method
  - Integrate categorization into Parse method
  - Update CSV output to include category columns
  - _Requirements: 2.1, 2.2, 2.3, 5.2, 5.3_

- [x] 5.1 Write property tests for PDF categorization
  - **Property 6: PDF categorization integration**
  - **Validates: Requirements 2.1, 5.3**

- [x] 5.2 Write property tests for PDF CSV format
  - **Property 8: Consistent CSV output format**
  - **Validates: Requirements 2.3, 4.1**
  - **Status: PASSED** - All 100 iterations completed successfully

- [x] 6. Add categorization to Selma parser
  - Modify `internal/selmaparser/selmaparser.go` to implement `CategorizerConfigurable`
  - Add categorizer field and SetCategorizer method
  - Integrate categorization into Parse method
  - Update CSV output to include category columns
  - _Requirements: 3.1, 3.2, 3.3, 5.2, 5.3_

- [x] 6.1 Write property tests for Selma categorization
  - **Property 7: Selma categorization integration**
  - **Validates: Requirements 3.1, 5.3**

- [x] 6.2 Write property tests for Selma CSV format
  - **Property 8: Consistent CSV output format**
  - **Validates: Requirements 3.3, 4.1**

- [x] 7. Implement categorization statistics and logging
  - Create categorization statistics tracking
  - Add logging for categorization results
  - Implement fallback behavior for failed categorization
  - _Requirements: 2.4, 2.5, 3.4, 3.5_

- [x] 7.1 Write property tests for categorization fallback
  - **Property 9: Categorization fallback behavior**
  - **Validates: Requirements 2.4, 3.4**

- [x] 7.2 Write property tests for statistics logging
  - **Property 10: Categorization statistics logging**
  - **Validates: Requirements 2.5, 3.5**

- [x] 8. Update container and dependency injection
  - Modify `internal/container/container.go` to configure categorizers for PDF and Selma parsers
  - Ensure consistent configuration across all parsers
  - Update parser factory to inject categorizers
  - _Requirements: 5.1, 5.2, 5.4_

- [x] 8.1 Write property tests for configuration consistency
  - **Property 11: Configuration consistency**
  - **Validates: Requirements 5.2**

- [x] 8.2 Write property tests for auto-learning consistency
  - **Property 12: Auto-learning mechanism consistency**
  - **Validates: Requirements 5.4**
  - **Status: PASSED** - Implemented in `internal/integration/cross_parser_test.go`

- [x] 9. Update CLI commands for enhanced parsers
  - Modify `cmd/pdf/convert.go` to use categorization
  - Modify `cmd/selma/convert.go` to use categorization
  - Ensure consistent command-line interface
  - _Requirements: 4.1, 4.3, 4.5_

- [x] 9.1 Write property tests for date range calculation
  - **Property 14: Date range calculation**
  - **Validates: Requirements 7.2**

- [x] 9.2 Write property tests for directory organization
  - **Property 15: Output directory organization**
  - **Validates: Requirements 7.4**

- [x] 10. Integration and end-to-end testing
  - Create integration tests for batch processing with multiple account files
  - Test PDF and Selma parsers with categorization enabled
  - Verify CSV output format consistency across all parsers
  - Test auto-learning mechanism with new parsers
  - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

- [x] 10.1 Write integration tests for cross-parser consistency
  - Test that all parsers produce identical CSV column structure
  - Verify categorization works consistently across parser types
  - Test batch processing with mixed file types

- [x] 11. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Each task references specific requirements for traceability
- Property tests validate universal correctness properties from the design document
- Integration tests ensure end-to-end functionality works correctly
- The implementation leverages existing categorization infrastructure without modification
- All tests are required for comprehensive quality assurance from the start
