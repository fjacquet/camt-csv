# ADR-005: Revolut Investment Parser Implementation

## Status

Accepted

## Context

The CAMT-CSV project currently supports parsing various financial statement formats, including standard Revolut transaction CSV files.
However, Revolut also provides investment transaction data in a different CSV format that is not supported by the existing parser.
Users who invest through Revolut cannot process their investment transactions with the current implementation.

The existing Revolut parser expects a format with headers like: Type, Product, Started Date, Completed Date, Description, Amount, Fee, Currency, State, Balance.

The Revolut investment CSV format has different headers: Date, Ticker, Type, Quantity, Price per share, Total Amount, Currency, FX Rate.

## Decision

We will implement a new, separate parser package specifically for Revolut investment transactions rather than extending the existing Revolut parser. This approach provides several benefits:

1. **Clear Separation of Concerns**: Investment transactions have different semantics than regular banking transactions
2. **Maintainability**: Keeping the parsers separate reduces complexity in each implementation
3. **Extensibility**: Future enhancements to either parser can be made independently
4. **Interface Compliance**: Both parsers will implement the standard `parser.Parser` interface

The new parser will be implemented in `internal/revolutinvestmentparser/` with its own adapter and will be accessible through a new `revolut-investment` CLI subcommand.

## Consequences

### Positive

- Users can now process Revolut investment transactions
- Clean separation between regular and investment transaction parsing
- Consistent with existing architectural patterns (separate parser packages)
- Maintains backward compatibility with existing functionality
- Follows the established parser interface pattern

### Negative

- Increased codebase size with a new package
- Slight duplication of some infrastructure code (adapters, etc.)
- Need to maintain two Revolut-related parsers

### Neutral

- Additional documentation required for the new feature
- New test suite needed for the investment parser
- New CLI subcommand to expose the functionality

## Implementation Plan

1. Create `internal/revolutinvestmentparser/` package
2. Implement parser logic with proper data mapping
3. Create adapter for standard parser interface compliance
4. Add unit and integration tests
5. Add `revolut-investment` subcommand to CLI
6. Update documentation

## Alternatives Considered

### Extend Existing Revolut Parser

We considered extending the existing Revolut parser to handle both formats, but decided against it because:

- Would increase complexity of the existing parser
- Different transaction types require different handling logic
- Risk of introducing bugs in the existing functionality
- Less clear separation of concerns

### Format Detection

We considered implementing automatic format detection, but decided against it because:

- Reliability concerns with format detection
- Potential for misclassification
- Added complexity to both parsers
- Explicit subcommands provide clearer user experience

## Related ADRs

- ADR-001: Parser Interface Standardization
- ADR-002: Hybrid Categorization Approach

## Notes

This decision aligns with the project's design principles of interface-driven design, single responsibility principle, and maintainability.
