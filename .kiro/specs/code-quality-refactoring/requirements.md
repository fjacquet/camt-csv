# Requirements Document

## Introduction

This specification defines the requirements for refactoring the camt-csv codebase to address critical code quality issues, improve maintainability, and align with Go best practices and the project constitution. The refactoring will eliminate anti-patterns, reduce technical debt, and establish a more testable and maintainable architecture without changing external functionality.

## Glossary

- **System**: The camt-csv application
- **Parser**: A component that converts financial data from a specific format to the standardized Transaction model
- **Categorizer**: A component that assigns categories to transactions using multiple strategies
- **Transaction**: A financial transaction record with standardized fields
- **BaseParser**: A common struct providing shared functionality for all parser implementations
- **Container**: A dependency injection container that manages application dependencies
- **Strategy**: A categorization algorithm implementation following the Strategy pattern
- **Builder**: A pattern for constructing complex Transaction objects with validation
- **Logger**: An abstraction interface for structured logging functionality

## Requirements

### Requirement 1: Eliminate Global State and Singleton Anti-Patterns

**User Story:** As a developer, I want to eliminate global mutable state and singleton patterns, so that the codebase is easier to test and maintain.

#### Acceptance Criteria

1. WHEN the System initializes, THE System SHALL create all dependencies explicitly through constructors
2. WHEN a component requires a Logger, THE System SHALL inject the Logger through the constructor rather than using a global variable
3. WHEN a component requires configuration, THE System SHALL inject the configuration through the constructor rather than using a global singleton
4. WHEN the Categorizer is instantiated, THE System SHALL provide all dependencies through the constructor
5. WHEN tests are executed, THE System SHALL allow independent test instances without shared global state

### Requirement 2: Standardize Error Handling Patterns

**User Story:** As a developer, I want consistent error handling throughout the codebase, so that errors are predictable and easier to debug.

#### Acceptance Criteria

1. WHEN a Parser encounters an unrecoverable error, THE Parser SHALL return a wrapped error with context using fmt.Errorf with %w
2. WHEN a Parser encounters a recoverable error with degraded functionality, THE Parser SHALL log a warning and continue processing
3. WHEN a function logs an error, THE function SHALL NOT also return that same error to avoid duplicate logging
4. WHEN the System defines domain-specific errors, THE System SHALL create custom error types in the parsererror package
5. WHEN error handling code is written, THE code SHALL use errors.Is and errors.As for error inspection rather than string comparison

### Requirement 3: Extract Common Parser Functionality

**User Story:** As a developer, I want to eliminate duplicated code across parser implementations, so that changes to common functionality only need to be made once.

#### Acceptance Criteria

1. WHEN a new Parser is created, THE Parser SHALL embed a BaseParser struct that provides common functionality
2. WHEN any Parser needs to write CSV output, THE Parser SHALL use the common WriteTransactionsToCSV function
3. WHEN any Parser needs Logger configuration, THE Parser SHALL inherit SetLogger from BaseParser
4. WHEN validation logic is common across parsers, THE System SHALL provide shared validation utilities
5. WHEN file handling is required, THE System SHALL use common file handling utilities rather than duplicating code

### Requirement 4: Decompose Transaction God Object

**User Story:** As a developer, I want the Transaction model to follow Single Responsibility Principle, so that it's easier to understand and maintain.

#### Acceptance Criteria

1. WHEN the System defines transaction data structures, THE System SHALL separate core transaction data from party information
2. WHEN the System represents monetary values, THE System SHALL use a Money value object containing amount and currency
3. WHEN the System handles transaction parties, THE System SHALL use a Party struct containing name and IBAN
4. WHEN the System categorizes transactions, THE System SHALL use composition to add categorization data to base transactions
5. WHEN existing code accesses transaction fields, THE System SHALL maintain backward compatibility through accessor methods during migration

### Requirement 5: Implement Interface Segregation for Parsers

**User Story:** As a developer, I want parser interfaces to be focused and composable, so that implementations only need to provide the capabilities they support.

#### Acceptance Criteria

1. WHEN the System defines the Parser interface, THE interface SHALL contain only the Parse method
2. WHEN a Parser supports format validation, THE Parser SHALL implement a separate Validator interface
3. WHEN a Parser supports CSV conversion, THE Parser SHALL implement a separate CSVConverter interface
4. WHEN the System needs a full-featured parser, THE System SHALL compose multiple interfaces
5. WHEN factory methods create parsers, THE factory SHALL return the appropriate interface composition based on parser capabilities

### Requirement 6: Create Logging Abstraction Layer

**User Story:** As a developer, I want to decouple the codebase from specific logging frameworks, so that we can change logging implementations without modifying business logic.

#### Acceptance Criteria

1. WHEN the System defines logging capabilities, THE System SHALL create a Logger interface in the logging package
2. WHEN components need logging, THE components SHALL depend on the Logger interface rather than concrete implementations
3. WHEN the System initializes, THE System SHALL provide a LogrusAdapter that implements the Logger interface
4. WHEN structured logging is required, THE System SHALL use a Field struct for key-value pairs
5. WHEN tests require logging, THE tests SHALL use a mock Logger implementation

### Requirement 7: Replace Magic Strings with Constants

**User Story:** As a developer, I want magic strings replaced with named constants, so that the code is more maintainable and less error-prone.

#### Acceptance Criteria

1. WHEN the System defines transaction types, THE System SHALL use constants for "DBIT", "CRDT", and other transaction codes
2. WHEN the System defines category names, THE System SHALL use constants for "Uncategorized", "Salaire", and other categories
3. WHEN the System defines transaction statuses, THE System SHALL use constants for "COMPLETED" and other status values
4. WHEN the System defines file permissions, THE System SHALL use named constants rather than octal literals
5. WHEN code references these values, THE code SHALL use the named constants rather than string literals

### Requirement 8: Standardize Naming Conventions

**User Story:** As a developer, I want consistent naming conventions throughout the codebase, so that the code is easier to read and understand.

#### Acceptance Criteria

1. WHEN the System uses the term for someone who owes money, THE System SHALL use "debtor" rather than "debitor"
2. WHEN package names are defined, THE packages SHALL use lowercase names without underscores
3. WHEN exported functions are defined, THE functions SHALL use PascalCase naming
4. WHEN private functions are defined, THE functions SHALL use camelCase naming
5. WHEN struct field names are defined, THE fields SHALL use consistent capitalization patterns

### Requirement 9: Refactor Date Handling to Use time.Time

**User Story:** As a developer, I want date handling to use Go's time.Time type internally, so that date operations are type-safe and less error-prone.

#### Acceptance Criteria

1. WHEN the Transaction struct stores dates, THE struct SHALL use time.Time fields rather than string fields
2. WHEN transactions are marshaled to CSV, THE System SHALL convert time.Time to the required string format
3. WHEN transactions are unmarshaled from CSV, THE System SHALL parse strings into time.Time values
4. WHEN date comparisons are needed, THE System SHALL use time.Time methods rather than string comparison
5. WHEN the migration is complete, THE System SHALL remove the complex FormatDate string manipulation function

### Requirement 10: Remove Test Environment Detection from Production Code

**User Story:** As a developer, I want production code to be free of test-specific logic, so that the code is cleaner and more maintainable.

#### Acceptance Criteria

1. WHEN the PDFParser is instantiated, THE Parser SHALL accept a PDFExtractor interface through dependency injection
2. WHEN production code runs, THE System SHALL provide a RealPDFExtractor implementation
3. WHEN tests run, THE tests SHALL provide a MockPDFExtractor implementation
4. WHEN the Parse method executes, THE method SHALL NOT check for TEST_ENV environment variable
5. WHEN the refactoring is complete, THE System SHALL remove all environment variable checks from production code

### Requirement 11: Implement Strategy Pattern for Categorization

**User Story:** As a developer, I want categorization logic to use the Strategy pattern, so that categorization methods are modular and testable.

#### Acceptance Criteria

1. WHEN the System defines categorization strategies, THE System SHALL create a CategorizationStrategy interface
2. WHEN direct mapping categorization is needed, THE System SHALL use a DirectMappingStrategy implementation
3. WHEN keyword-based categorization is needed, THE System SHALL use a KeywordStrategy implementation
4. WHEN AI-based categorization is needed, THE System SHALL use an AIStrategy implementation
5. WHEN the Categorizer processes a Transaction, THE Categorizer SHALL iterate through strategies in priority order until one succeeds

### Requirement 12: Implement Builder Pattern for Transaction Construction

**User Story:** As a developer, I want to use the Builder pattern for constructing transactions, so that complex transaction creation is more readable and maintainable.

#### Acceptance Criteria

1. WHEN the System provides Transaction construction, THE System SHALL offer a TransactionBuilder type
2. WHEN a TransactionBuilder is created, THE Builder SHALL initialize with sensible defaults
3. WHEN Transaction fields are set, THE Builder SHALL provide fluent methods that return the Builder for chaining
4. WHEN the Build method is called, THE Builder SHALL validate and populate derived fields automatically
5. WHEN complex transactions are created in parsers, THE parsers SHALL use TransactionBuilder for improved readability

### Requirement 13: Optimize Performance in Hot Paths

**User Story:** As a developer, I want performance optimizations in frequently-executed code paths, so that the application runs efficiently.

#### Acceptance Criteria

1. WHEN string operations are performed repeatedly in categorization, THE System SHALL minimize allocations using strings.Builder
2. WHEN expensive resources like AI clients are initialized, THE System SHALL use lazy initialization with sync.Once
3. WHEN maps are created with known sizes, THE System SHALL pre-allocate capacity using make with size hints
4. WHEN slices grow dynamically, THE System SHALL pre-allocate capacity when the final size is known
5. WHEN the optimizations are complete, THE System SHALL maintain identical functionality with improved performance

### Requirement 14: Maintain Backward Compatibility During Migration

**User Story:** As a user, I want the refactoring to maintain all existing functionality, so that my workflows are not disrupted.

#### Acceptance Criteria

1. WHEN the refactoring is in progress, THE System SHALL maintain all existing CLI commands and flags
2. WHEN the refactoring is in progress, THE System SHALL produce identical CSV output for the same input files
3. WHEN the refactoring introduces new interfaces, THE System SHALL provide adapter layers for existing code
4. WHEN deprecated methods exist, THE methods SHALL include deprecation comments with migration guidance
5. WHEN the refactoring is complete, THE System SHALL pass all existing tests without modification

### Requirement 15: Improve Test Coverage and Testability

**User Story:** As a developer, I want improved test coverage and testability, so that the codebase is more reliable and easier to maintain.

#### Acceptance Criteria

1. WHEN components use dependency injection, THE tests SHALL easily mock dependencies
2. WHEN the Categorizer is tested, THE tests SHALL inject mock AIClient and CategoryStore implementations
3. WHEN parsers are tested, THE tests SHALL not require file system access for unit tests
4. WHEN the refactoring is complete, THE System SHALL achieve comprehensive test coverage with 100% coverage for critical paths (parsing, categorization, data validation) and good coverage for remaining functionality
5. WHEN test coverage is evaluated, THE System SHALL prioritize risk-based testing over arbitrary percentage targets


