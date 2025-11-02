# Implementation Plan

## Phase 1: Foundation Infrastructure

- [x] 1. Establish Logging Infrastructure ✅ COMPLETE
  - [x] 1.1 Create logging abstraction layer
    - Created `internal/logging/logger.go` with Logger interface and Field struct
    - Implemented LogrusAdapter that wraps logrus.Logger in `logrus_adapter.go`
    - Added constructor functions for creating loggers with different configurations
    - _Requirements: 6.1, 6.2, 6.3, 6.4_
  - [x] 1.2 Write comprehensive unit tests
    - Test LogrusAdapter implements Logger interface correctly
    - Test field conversion and structured logging
    - Test logger creation with various configurations
    - All tests passing with 100% coverage
    - _Requirements: 6.5, 15.2_

- [x] 2. Eliminate Magic Strings and Numbers
  - [x] 2.1 Define comprehensive constants
    - Create `internal/models/constants.go` with transaction type constants
    - Add category name constants
    - Add status constants
    - Add bank transaction code constants
    - Add file permission constants
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_
  - [x] 2.2 Replace magic strings throughout codebase
    - Update categorizer package to use category constants
    - Update parser packages to use transaction type constants
    - Update file operations to use permission constants
    - Update status checks to use status constants
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_
  - [x] 2.3 Verify complete elimination
    - Run grep search for hardcoded "DBIT", "CRDT", "Uncategorized"
    - Verify no magic file permissions remain
    - Check all status strings use constants
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

## Phase 2: Parser Architecture Refactoring

- [x] 3. Redesign Parser Interfaces
  - [x] 3.1 Define segregated interfaces
    - Update `internal/parser/parser.go` with Parser interface (Parse method only)
    - Add Validator interface with ValidateFormat method
    - Add CSVConverter interface with ConvertToCSV method
    - Add LoggerConfigurable interface with SetLogger method
    - Create FullParser interface composing all capabilities
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_
  - [x] 3.2 Create BaseParser foundation
    - Create `internal/parser/base.go` with BaseParser struct
    - Implement SetLogger method in BaseParser
    - Implement GetLogger helper method
    - Add common WriteToCSV method using common.WriteTransactionsToCSV
    - _Requirements: 3.1, 3.2, 3.3_

- [x] 4. Refactor All Parsers to Use BaseParser
  - [x] 4.1 Refactor CAMT parser
    - Update ISO20022Parser to embed BaseParser
    - Remove duplicate SetLogger implementation
    - Remove duplicate WriteToCSV implementation
    - Update constructor to accept logger parameter
    - Verify all tests pass
    - _Requirements: 3.1, 3.2, 3.3, 3.4_
  - [x] 4.2 Refactor Revolut parser
    - Update RevolutParser to embed BaseParser
    - Remove duplicate SetLogger implementation
    - Remove duplicate WriteToCSV implementation
    - Update constructor to accept logger parameter
    - Verify all tests pass
    - _Requirements: 3.1, 3.2, 3.3, 3.4_
  - [x] 4.3 Refactor PDF parser with dependency injection
    - Update PDFParser to embed BaseParser
    - Define PDFExtractor interface in `internal/pdfparser/extractor.go`
    - Implement RealPDFExtractor using pdftotext
    - Implement MockPDFExtractor for testing
    - Update PDFParser to accept PDFExtractor via constructor
    - Remove TEST_ENV environment variable checks
    - Write tests for PDF parser with mock extractor
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 10.1, 10.2, 10.3, 10.4, 10.5, 15.3_
  - [x] 4.4 Refactor remaining parsers
    - Refactor Selma, Debit, RevolutInvestment parsers to embed BaseParser
    - Remove duplicate code from all parsers
    - Update constructors to accept logger
    - Verify all tests pass
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

## Phase 3: Dependency Injection and Error Handling

- [x] 5. Establish Error Handling Standards
  - [x] 5.1 Define custom error types
    - Create `internal/parsererror/errors.go` with ParseError type
    - Add ValidationError type
    - Add CategorizationError type
    - Implement Error() and Unwrap() methods for each
    - Write comprehensive tests for error handling patterns
    - _Requirements: 2.4, 2.1, 2.2, 2.3, 2.5_
  - [x] 5.2 Standardize error handling in parsers
    - Update parsers to return errors for unrecoverable issues
    - Update parsers to log warnings for recoverable issues
    - Remove instances of logging and returning same error
    - Use custom error types where appropriate
    - Wrap errors with context using fmt.Errorf with %w
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_

- [x] 6. Implement Dependency Injection Architecture
  - [x] 6.1 Create dependency container
    - Create `internal/container/container.go` with Container struct
    - Implement NewContainer function that wires all dependencies
    - Add GetParser method for retrieving parsers by type
    - Add helper methods for accessing common dependencies
    - _Requirements: 1.1, 1.2, 1.3_
  - [x] 6.2 Refactor Categorizer for dependency injection
    - Update Categorizer constructor to accept all dependencies
    - Remove global defaultCategorizer variable
    - Remove initCategorizer function
    - Update CategorizeTransaction to accept Categorizer instance
    - Add deprecation notice to old global functions
    - _Requirements: 1.1, 1.2, 1.3, 1.4_
  - [x] 6.3 Update CLI commands and config
    - Update all CLI commands (`cmd/camt/convert.go`, `cmd/pdf/convert.go`, etc.) to use container
    - Remove direct parser factory usage
    - Refactor config.Logger to be instance-based
    - Remove global configOnce and once variables
    - Update GetGlobalConfig to be deprecated
    - Provide migration path for existing code
    - _Requirements: 1.1, 1.2, 1.3_

## Phase 4: Transaction Model Modernization

- [-] 7. Redesign Transaction Data Model
  - [x] 7.1 Create decomposed transaction structure
    - Create Money value object in `internal/models/money.go`
    - Create Party struct in `internal/models/party.go`
    - Create TransactionCore struct in `internal/models/transaction_core.go`
    - Create TransactionWithParties struct
    - Create CategorizedTransaction struct
    - Define TransactionDirection type and constants
    - _Requirements: 4.1, 4.2, 4.3, 4.4_
  - [x] 7.2 Modernize date handling
    - Change Date field from string to time.Time
    - Change ValueDate field from string to time.Time
    - Update MarshalCSV to format time.Time as DD.MM.YYYY
    - Update UnmarshalCSV to parse strings to time.Time
    - Remove complex FormatDate string manipulation
    - Update dateutils package to focus on time.Time operations
    - _Requirements: 9.1, 9.2, 9.3, 9.5_

- [x] 8. Implement Transaction Builder Pattern
  - [x] 8.1 Create TransactionBuilder
    - Create `internal/models/builder.go` with TransactionBuilder struct
    - Implement fluent methods: WithDate, WithAmount, WithPayer, WithPayee, etc.
    - Implement AsDebit and AsCredit methods
    - Implement Build method with validation
    - Implement populateDerivedFields helper
    - Write comprehensive tests for TransactionBuilder
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 15.1_
  - [x] 8.2 Add backward compatibility
    - Add GetAmountAsFloat (deprecated) to Transaction
    - Add GetPayee helper method
    - Add GetPayer helper method
    - Add conversion methods between old and new formats
    - Add deprecation comments with migration guidance
    - _Requirements: 4.5, 14.4_
  - [x] 8.3 Migrate all parsers to use TransactionBuilder
    - Update CAMT parser entryToTransaction to use builder
    - Update Revolut parser convertRevolutRowToTransaction to use builder
    - Update PDF parser transaction construction
    - Update Selma parser transaction construction
    - Update other parsers as needed
    - Verify all tests pass
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 12.5_

## Phase 5: Categorization Strategy Pattern

- [x] 9. Implement Strategy Pattern for Categorization
  - [x] 9.1 Define strategy architecture
    - Create `internal/categorizer/strategy.go` with CategorizationStrategy interface
    - Define Categorize method signature with context
    - Add Name method for logging/debugging
    - _Requirements: 11.1_
  - [x] 9.2 Implement categorization strategies
    - Create DirectMappingStrategy in `internal/categorizer/direct_mapping.go`
    - Implement strategy for creditor mappings
    - Implement strategy for debitor mappings
    - Write comprehensive tests for DirectMappingStrategy
    - _Requirements: 11.2, 15.2_
  - [x] 9.3 Implement keyword and AI strategies
    - Create KeywordStrategy in `internal/categorizer/keyword.go`
    - Implement pattern matching logic
    - Load keyword patterns from configuration
    - Write tests for KeywordStrategy
    - Create AIStrategy in `internal/categorizer/ai_strategy.go`
    - Implement AI client integration with error handling
    - Write tests for AIStrategy with mock client
    - _Requirements: 11.3, 11.4, 15.2_
  - [x] 9.4 Refactor Categorizer to orchestrate strategies
    - Update Categorizer to hold slice of strategies
    - Implement strategy iteration in Categorize method
    - Initialize strategies in priority order in constructor
    - Remove old categorization methods
    - Write integration tests for Categorizer
    - Verify same results as before refactoring
    - _Requirements: 11.5, 15.2_

## Phase 6: Performance Optimization and Quality Assurance

- [ ] 10. Implement Performance Optimizations
  - [ ] 10.1 Optimize string operations and resource usage
    - Update categorizer to use strings.Builder for repeated operations
    - Pre-allocate builder capacity where size is known
    - Minimize string allocations in hot paths
    - Add lazy initialization for AI client in Categorizer
    - Use sync.Once for thread-safe initialization
    - Pre-allocate slices with known capacity in parsers
    - Pre-allocate maps with size hints where applicable
    - _Requirements: 13.1, 13.2, 13.3, 13.4_
  - [ ] 10.2 Benchmark and validate performance
    - Create benchmarks for hot paths
    - Compare before and after optimization
    - Verify no performance regression
    - Document performance gains
    - _Requirements: 13.5_

- [ ] 11. Standardize Naming and Improve Test Coverage
  - [ ] 11.1 Standardize naming conventions
    - Rename all "debitor" to "debtor" in code
    - Rename debitors.yaml to debtors.yaml
    - Update configuration references
    - Update documentation
    - Add migration note for config file rename
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_
  - [ ] 11.2 Achieve comprehensive test coverage
    - Add tests for uncovered parser code paths
    - Add tests for error scenarios
    - Add tests for edge cases in categorization
    - Add integration tests for end-to-end flows
    - Verify 80%+ overall coverage
    - Verify 100% coverage for critical paths
    - _Requirements: 15.4, 15.5_

- [ ] 12. Documentation and Migration Cleanup
  - [ ] 12.1 Update all documentation
    - Update package documentation with new patterns
    - Add godoc comments for all public APIs
    - Create architecture documentation
    - Update developer guide with new patterns
    - Create migration guide for users
    - Verify documentation completeness
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.5_
  - [ ] 12.2 Remove deprecated code
    - Remove old global singleton functions (with major version bump)
    - Remove deprecated accessor methods
    - Create migration guide document
    - Update CHANGELOG with breaking changes
    - _Requirements: 14.4_

## Final Verification and Quality Gates

- [ ] 13. Comprehensive Testing and Validation
  - [ ] 13.1 Run full test suite and verify coverage
    - Run all unit tests
    - Run all integration tests
    - Generate coverage report
    - Verify 80%+ overall coverage
    - Verify 100% coverage for critical paths
    - _Requirements: 15.4, 15.5_
  - [ ] 13.2 Verify backward compatibility
    - Test all CLI commands with sample files
    - Compare CSV output with previous version
    - Verify existing config files work
    - Test deprecated APIs still function
    - _Requirements: 14.1, 14.2, 14.3, 14.5_
  - [ ] 13.3 Performance and quality validation
    - Run performance benchmarks
    - Compare with baseline performance
    - Verify no significant regressions
    - Document performance improvements
    - Run golangci-lint and fix any issues
    - Run go vet and gosec for security issues
    - Verify all checks pass
    - _Requirements: 13.5, 1.1, 2.1, 2.2, 2.3, 2.4, 2.5_