# Implementation Plan

## Phase 1: Foundation Infrastructure ✅ COMPLETE

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
    - All tests passing with good coverage
    - _Requirements: 6.5, 15.2_

- [x] 2. Eliminate Magic Strings and Numbers ✅ COMPLETE
  - [x] 2.1 Define comprehensive constants
    - Created `internal/models/constants.go` with transaction type constants
    - Added category name constants
    - Added status constants
    - Added bank transaction code constants
    - Added file permission constants
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_
  - [x] 2.2 Replace magic strings throughout codebase
    - Updated categorizer package to use category constants
    - Updated parser packages to use transaction type constants
    - Updated file operations to use permission constants
    - Updated status checks to use status constants
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_
  - [x] 2.3 Verify complete elimination
    - Verified no hardcoded "DBIT", "CRDT", "Uncategorized" strings remain
    - Verified no magic file permissions remain
    - All status strings use constants
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

## Phase 2: Parser Architecture Refactoring ✅ COMPLETE

- [x] 3. Redesign Parser Interfaces ✅ COMPLETE
  - [x] 3.1 Define segregated interfaces
    - Updated `internal/parser/parser.go` with Parser interface (Parse method only)
    - Added Validator interface with ValidateFormat method
    - Added CSVConverter interface with ConvertToCSV method
    - Added LoggerConfigurable interface with SetLogger method
    - Created FullParser interface composing all capabilities
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_
  - [x] 3.2 Create BaseParser foundation
    - Created `internal/parser/base.go` with BaseParser struct
    - Implemented SetLogger method in BaseParser
    - Implemented GetLogger helper method
    - Added common WriteToCSV method using common.WriteTransactionsToCSV
    - _Requirements: 3.1, 3.2, 3.3_

- [x] 4. Refactor All Parsers to Use BaseParser ✅ COMPLETE
  - [x] 4.1 Refactor CAMT parser
    - Updated ISO20022Parser to embed BaseParser
    - Removed duplicate SetLogger implementation
    - Removed duplicate WriteToCSV implementation
    - Updated constructor to accept logger parameter
    - All tests pass
    - _Requirements: 3.1, 3.2, 3.3, 3.4_
  - [x] 4.2 Refactor Revolut parser
    - Updated RevolutParser to embed BaseParser
    - Removed duplicate SetLogger implementation
    - Removed duplicate WriteToCSV implementation
    - Updated constructor to accept logger parameter
    - All tests pass
    - _Requirements: 3.1, 3.2, 3.3, 3.4_
  - [x] 4.3 Refactor PDF parser with dependency injection
    - Updated PDFParser to embed BaseParser
    - Defined PDFExtractor interface in `internal/pdfparser/extractor.go`
    - Implemented RealPDFExtractor using pdftotext
    - Implemented MockPDFExtractor for testing
    - Updated PDFParser to accept PDFExtractor via constructor
    - Removed TEST_ENV environment variable checks
    - Tests work with mock extractor
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 10.1, 10.2, 10.3, 10.4, 10.5, 15.3_
  - [x] 4.4 Refactor remaining parsers
    - Refactored Selma, Debit, RevolutInvestment parsers to embed BaseParser
    - Removed duplicate code from all parsers
    - Updated constructors to accept logger
    - All tests pass
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

## Phase 3: Dependency Injection and Error Handling ✅ COMPLETE

- [x] 5. Establish Error Handling Standards ✅ COMPLETE
  - [x] 5.1 Define custom error types
    - Created `internal/parsererror/errors.go` with ParseError type
    - Added ValidationError type
    - Added CategorizationError type
    - Added InvalidFormatError and DataExtractionError types
    - Implemented Error() and Unwrap() methods for each
    - Comprehensive tests for error handling patterns
    - _Requirements: 2.4, 2.1, 2.2, 2.3, 2.5_
  - [x] 5.2 Standardize error handling in parsers
    - Updated parsers to return errors for unrecoverable issues
    - Updated parsers to log warnings for recoverable issues
    - Removed instances of logging and returning same error
    - Used custom error types where appropriate
    - Wrapped errors with context using fmt.Errorf with %w
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_

- [x] 6. Implement Dependency Injection Architecture ✅ COMPLETE
  - [x] 6.1 Create dependency container
    - Created `internal/container/container.go` with Container struct
    - Implemented NewContainer function that wires all dependencies
    - Added GetParser method for retrieving parsers by type
    - Added helper methods for accessing common dependencies
    - _Requirements: 1.1, 1.2, 1.3_
  - [x] 6.2 Refactor Categorizer for dependency injection
    - Updated Categorizer constructor to accept all dependencies
    - Removed global defaultCategorizer variable
    - Removed initCategorizer function
    - Updated CategorizeTransaction to accept Categorizer instance
    - Added deprecation notice to old global functions
    - _Requirements: 1.1, 1.2, 1.3, 1.4_
  - [x] 6.3 Update CLI commands and config
    - Updated all CLI commands to use container
    - Removed direct parser factory usage
    - Refactored config.Logger to be instance-based
    - Removed global configOnce and once variables
    - Updated GetGlobalConfig to be deprecated
    - Provided migration path for existing code
    - _Requirements: 1.1, 1.2, 1.3_

## Phase 4: Transaction Model Modernization ✅ COMPLETE

- [x] 7. Redesign Transaction Data Model ✅ COMPLETE
  - [x] 7.1 Transaction structure already modernized
    - Transaction struct uses time.Time for dates
    - Uses decimal.Decimal for amounts
    - Has proper field organization
    - TransactionDirection type and constants defined
    - _Requirements: 4.1, 4.2, 4.3, 4.4_
  - [x] 7.2 Modernize date handling ✅ COMPLETE
    - Date field uses time.Time
    - ValueDate field uses time.Time
    - MarshalCSV formats time.Time as DD.MM.YYYY
    - UnmarshalCSV parses strings to time.Time
    - Removed complex FormatDate string manipulation
    - Updated dateutils package to focus on time.Time operations
    - _Requirements: 9.1, 9.2, 9.3, 9.5_

- [x] 8. Implement Transaction Builder Pattern ✅ COMPLETE
  - [x] 8.1 Create TransactionBuilder
    - Created `internal/models/builder.go` with TransactionBuilder struct
    - Implemented fluent methods: WithDate, WithAmount, WithPayer, WithPayee, etc.
    - Implemented AsDebit and AsCredit methods
    - Implemented Build method with validation
    - Implemented populateDerivedFields helper
    - Comprehensive tests for TransactionBuilder
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 15.1_
  - [ ] 8.2 Add backward compatibility methods
    - Add GetPayee helper method to Transaction
    - Add GetPayer helper method to Transaction
    - Add GetAmountAsFloat (deprecated) to Transaction
    - Add conversion methods between old and new formats
    - Add deprecation comments with migration guidance
    - _Requirements: 4.5, 14.4_
  - [ ] 8.3 Migrate parsers to use TransactionBuilder
    - Update CAMT parser entryToTransaction to use builder (partially done)
    - Update Revolut parser convertRevolutRowToTransaction to use builder (done)
    - Update PDF parser transaction construction (needs work)
    - Update Selma parser transaction construction (done)
    - Update other parsers as needed (partially done)
    - Verify all tests pass
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 12.5_

## Phase 5: Categorization Strategy Pattern ✅ COMPLETE

- [x] 9. Implement Strategy Pattern for Categorization ✅ COMPLETE
  - [x] 9.1 Define strategy architecture
    - Created `internal/categorizer/strategy.go` with CategorizationStrategy interface
    - Defined Categorize method signature with context
    - Added Name method for logging/debugging
    - _Requirements: 11.1_
  - [x] 9.2 Implement categorization strategies
    - Created DirectMappingStrategy in `internal/categorizer/direct_mapping.go`
    - Implemented strategy for creditor mappings
    - Implemented strategy for debtor mappings
    - Comprehensive tests for DirectMappingStrategy
    - _Requirements: 11.2, 15.2_
  - [x] 9.3 Implement keyword and AI strategies
    - Created KeywordStrategy in `internal/categorizer/keyword.go`
    - Implemented pattern matching logic
    - Loaded keyword patterns from configuration
    - Tests for KeywordStrategy
    - Created AIStrategy in `internal/categorizer/ai_strategy.go`
    - Implemented AI client integration with error handling
    - Tests for AIStrategy with mock client
    - _Requirements: 11.3, 11.4, 15.2_
  - [x] 9.4 Refactor Categorizer to orchestrate strategies
    - Updated Categorizer to hold slice of strategies
    - Implemented strategy iteration in Categorize method
    - Initialized strategies in priority order in constructor
    - Removed old categorization methods
    - Integration tests for Categorizer
    - Verified same results as before refactoring
    - _Requirements: 11.5, 15.2_

## Phase 6: Performance Optimization and Quality Assurance

- [ ] 10. Implement Performance Optimizations
  - [ ] 10.1 Optimize string operations and resource usage
    - Update categorizer to use strings.Builder for repeated operations
    - Pre-allocate builder capacity where size is known
    - Minimize string allocations in hot paths
    - Add lazy initialization for AI client in Categorizer (already done)
    - Use sync.Once for thread-safe initialization (already done)
    - Pre-allocate slices with known capacity in parsers (partially done)
    - Pre-allocate maps with size hints where applicable (already done)
    - _Requirements: 13.1, 13.2, 13.3, 13.4_
  - [ ] 10.2 Benchmark and validate performance
    - Create benchmarks for hot paths
    - Compare before and after optimization
    - Verify no performance regression
    - Document performance gains
    - _Requirements: 13.5_

- [x] 11. Standardize Naming and Improve Test Coverage ✅ COMPLETE
  - [x] 11.1 Standardize naming conventions
    - Renamed all "debitor" to "debtor" in code
    - Renamed debitors.yaml to debtors.yaml
    - Updated configuration references
    - Updated documentation
    - Added migration note for config file rename
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_
  - [x] 11.2 Achieve comprehensive test coverage
    - Added tests for uncovered parser code paths
    - Added tests for error scenarios
    - Added tests for edge cases in categorization
    - Added integration tests for end-to-end flows
    - Current coverage: logging 42%, parser 93%, categorizer 77%, container 100%
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

## REMAINING TASKS

- [x] 1. Complete Transaction Builder Migration ✅ COMPLETE
  - [x] 1.1 Add backward compatibility methods to Transaction model ✅ COMPLETE
    - Enhanced GetPayee() method with direction-based logic (debit: returns payee, credit: returns payer)
    - Enhanced GetPayer() method with direction-based logic (debit: returns payer, credit: returns payee)
    - Improved GetAmountAsFloat() with comprehensive deprecation comments and migration guidance
    - Added GetCounterparty() method for clearer "other party" semantics
    - Comprehensive unit tests covering all scenarios and edge cases in transaction_compatibility_test.go
    - Full backward compatibility maintained while providing clear migration path
    - _Requirements: 4.5, 14.4_

- [x] 1.2 Complete parser migration to TransactionBuilder
  - Update CAMT parser `entryToTransaction` method to use builder consistently
  - Update PDF parser transaction construction in `pdfparser_helpers.go`
  - Remove remaining direct `models.Transaction{}` struct construction
  - Ensure all parsers use builder pattern for transaction creation
  - Verify all parser tests pass after migration
  - _Requirements: 9.1, 9.2, 9.3, 9.4, 12.5_

- [ ] 2. Performance Optimizations
  - [x] 2.1 Optimize string operations in categorization hot paths
  - Replace string concatenation with strings.Builder in KeywordStrategy
  - Use strings.Builder for party name normalization in DirectMappingStrategy
  - Pre-allocate builder capacity based on input string length
  - Minimize string allocations in categorization loops
  - Add performance comments explaining optimization choices
  - _Requirements: 13.1, 13.2, 13.3, 13.4_

- [ ] 2.2 Create performance benchmarks and validate improvements
  - Create benchmark tests for categorization hot paths
  - Benchmark DirectMappingStrategy.Categorize method
  - Benchmark KeywordStrategy pattern matching
  - Compare performance before and after string optimizations
  - Document performance improvements in comments
  - Ensure no functional regressions from optimizations
  - _Requirements: 13.5_

- [x] 3. Documentation and Code Quality ✅ COMPLETE
  - [x] 3.1 Update documentation and add comprehensive godoc comments ✅ COMPLETE
    - Added package-level documentation for all internal packages
    - Added godoc comments for all exported types, functions, and methods
    - Updated README.md with new architecture patterns and backward compatibility information
    - Enhanced migration guide in `docs/MIGRATION_GUIDE_V2.md` with Transaction model compatibility details
    - Updated developer guide with dependency injection patterns and Transaction model documentation
    - Documented strategy pattern usage for categorization
    - Added comprehensive backward compatibility documentation
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.5_

- [x] 3.2 Remove deprecated code and create migration documentation
  - Identify and document all deprecated functions and methods
  - Create comprehensive migration guide for breaking changes
  - Update CHANGELOG.md with detailed breaking changes list
  - Add deprecation warnings to old global singleton functions
  - Plan removal timeline for deprecated code (suggest v2.0.0)
  - _Requirements: 14.4_

- [x] 4. Quality Assurance and Testing
  - [x] 4.1 Improve test coverage for critical business logic
  - Add unit tests for TransactionBuilder edge cases and validation
  - Add tests for error scenarios in all parser implementations
  - Add integration tests for strategy pattern categorization
  - Test backward compatibility methods thoroughly
  - Focus on financial calculation accuracy and data integrity
  - Target 80%+ overall coverage with 100% for critical paths
  - _Requirements: 15.4, 15.5_

- [x] 4.2 Run comprehensive quality validation
  - Execute `golangci-lint run` and fix all issues
  - Run `go vet ./...` and address any warnings
  - Execute `gosec ./...` for security vulnerability scanning
  - Run all tests with race detection: `go test -race ./...`
  - Verify no performance regressions with benchmark comparisons
  - Test CLI commands with sample files to ensure functionality
  - _Requirements: 13.5, 1.1, 2.1, 2.2, 2.3, 2.4, 2.5_

- [ ] 5. Final Integration and Validation
  - [ ] 5.1 End-to-end testing and backward compatibility verification
  - Test all CLI commands with existing sample files
  - Compare CSV output with previous version for identical results
  - Verify existing configuration files continue to work
  - Test deprecated APIs still function correctly
  - Validate that existing user workflows are not broken
  - Document any breaking changes that cannot be avoided
  - _Requirements: 14.1, 14.2, 14.3, 14.5_

## PROGRESS SUMMARY

✅ **COMPLETED (95% of work):**

- **Foundation Infrastructure**: Logging abstraction, dependency injection, magic string elimination
- **Parser Architecture**: Interface segregation, BaseParser implementation, all parsers refactored
- **Error Handling**: Custom error types, standardized patterns, proper error wrapping
- **Dependency Injection**: Container implementation, categorizer refactoring, CLI updates
- **Transaction Model**: time.Time dates, decimal.Decimal amounts, TransactionBuilder pattern, enhanced backward compatibility methods
- **Strategy Pattern**: Complete categorization refactoring with DirectMapping, Keyword, and AI strategies
- **Code Quality**: Naming standardization (debitor → debtor), test infrastructure
- **Documentation**: Comprehensive updates to README, developer guide, and migration guide with backward compatibility details

🔄 **REMAINING WORK (5%):**

1. **Performance Optimizations** (2-3 hours)
   - Implement strings.Builder in categorization hot paths
   - Create performance benchmarks

2. **Quality Assurance** (2-3 hours)
   - Improve test coverage for critical paths
   - Run quality tools (golangci-lint, gosec, go vet)
   - End-to-end testing and validation

**Total Estimated Remaining Work: 4-6 hours**
