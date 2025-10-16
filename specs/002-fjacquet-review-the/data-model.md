# Data Model: Codebase Improvements

**Date**: mardi 14 octobre 2025

## Key Entities

### Parser
- **Description**: An interface (`internal/parser/Parser.go`) that defines the contract for converting various input data formats (e.g., CAMT, PDF, Revolut files) into a standardized `Transaction` format. Concrete implementations will adhere to this interface.
- **Fields**: N/A (interface)
- **Relationships**: Implemented by `camtparser.Adapter`, `pdfparser.Adapter`, `revolutparser.Adapter`, `revolutinvestmentparser.Adapter`, `selmaparser.Adapter`, `debitparser.Adapter`.
- **Validation Rules**: N/A (interface)

### ParserType
- **Description**: An enumeration (string type) used by the `ParserFactory` to identify and instantiate specific `Parser` implementations.
- **Values**: `CAMT`, `PDF`, `Revolut`, `RevolutInvestment`, `Selma`, `Debit`.
- **Relationships**: Used as input to the `ParserFactory.GetParser` function.

### Custom Errors
- **Description**: Specific error types to provide detailed context for parsing and data extraction failures, improving error handling and user feedback.
- **Types**:
    - `InvalidFormatError`: Returned when an input file does not conform to the expected format.
        - **Fields**: `FilePath` (string), `ExpectedFormat` (string), `ActualContentSnippet` (string, optional).
    - `DataExtractionError`: Returned when specific required data cannot be extracted from a valid file.
        - **Fields**: `FilePath` (string), `FieldName` (string), `RawDataSnippet` (string, optional), `Reason` (string).
- **Relationships**: Returned by `Parser` implementations and handled by calling code.
- **Validation Rules**: Error messages should be clear and actionable.

### AIClient
- **Description**: An interface (`internal/categorizer/AIClient.go`) defining the contract for AI-based categorization services, allowing for mock implementations in tests.
- **Fields**: N/A (interface)
- **Relationships**: Implemented by `GeminiClient`.
- **Validation Rules**: N/A (interface)

### GeminiClient
- **Description**: A concrete implementation of the `AIClient` interface that interacts with the Google Gemini API for transaction categorization.
- **Fields**: `APIKey` (string, configured via environment variables), `HTTPClient` (interface for making API calls).
- **Relationships**: Uses the Gemini API; consumed by the `Categorizer`.
- **Validation Rules**: Requires a valid API key; handles API errors gracefully.

### Transaction
- **Description**: A standardized data structure (`internal/models/transaction.go`) representing a financial transaction, used consistently across all parsers and the categorizer.
- **Fields**: (Existing fields, e.g., `Date`, `Description`, `Amount`, `Currency`, `Category`, `Type`, `Reference`)
- **Relationships**: Output of `Parser` implementations; input to `Categorizer`.
- **Validation Rules**: (Existing validation rules for transaction fields)

### Log Field Constants
- **Description**: Predefined string constants (`internal/logging/constants.go` or similar) for consistent naming of structured log fields, improving log parsing and analysis.
- **Examples**: `FieldParser = "parser"`, `FieldFile = "file_path"`, `FieldTransactionID = "transaction_id"`, `FieldCategory = "category"`.
- **Relationships**: Used by `logrus` logger throughout the application.
- **Validation Rules**: Constants should be unique and descriptive.
