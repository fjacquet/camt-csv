# Requirements Document

## Introduction

This specification defines requirements for three focused enhancements to the camt-csv parser system: batch consolidation by bank account, and adding categorization support to PDF and Selma parsers. These enhancements will leverage the existing categorization system without reinventing functionality.

## Glossary

- **System**: The camt-csv application
- **Parser**: A component that converts financial data from a specific format to standardized Transaction models
- **Categorizer**: The existing transaction categorization system with direct mapping, keyword matching, and AI fallback
- **Batch_Processor**: The component that processes multiple files in a single operation
- **Bank_Account**: A unique identifier for a financial account (typically IBAN or account number)
- **Consolidated_Output**: A single CSV file containing transactions from multiple input files for the same bank account
- **PDF_Parser**: The existing parser for PDF bank statements (currently Viseca format)
- **Selma_Parser**: The existing parser for Selma investment CSV files

## Requirements

### Requirement 1: Enhance Batch Processing for Account Aggregation

**User Story:** As a user, I want the batch command to aggregate multiple files from the same account into consolidated output files, so that I get one file per account instead of one file per input file.

#### Acceptance Criteria

1. WHEN the Batch_Processor processes multiple CAMT files with the same account number (e.g., "CAMT.053_54293249_*"), THE System SHALL aggregate all transactions into a single output file per account
2. WHEN the System creates aggregated output files, THE System SHALL name them using the account number and overall date range (e.g., "54293249_2025-04-01_2025-06-30.csv")
3. WHEN transactions from multiple files are aggregated, THE System SHALL sort them chronologically by transaction date
4. WHEN the System detects potential duplicate transactions across files, THE System SHALL include all transactions but log a warning
5. WHEN the aggregated file is created, THE System SHALL include a header comment listing the source files that were merged

### Requirement 2: Add Categorization to PDF Parser

**User Story:** As a user, I want PDF-parsed transactions to be automatically categorized using the existing categorization system, so that PDF transactions have the same category information as other parser outputs.

#### Acceptance Criteria

1. WHEN the PDF_Parser processes transactions, THE System SHALL apply the existing Categorizer (direct mapping → keyword matching → AI fallback) to each transaction
2. WHEN the PDF_Parser creates Transaction objects, THE System SHALL populate category and subcategory fields using the existing categorization logic
3. WHEN the PDF_Parser outputs CSV, THE System SHALL include category and subcategory columns in the same format as CAMT and Revolut parsers
4. WHEN categorization fails for a PDF transaction, THE System SHALL assign "Uncategorized" as the category
5. WHEN the PDF_Parser completes processing, THE System SHALL log categorization statistics (successful/failed/uncategorized counts)

### Requirement 3: Add Categorization to Selma Parser

**User Story:** As a user, I want Selma investment transactions to be automatically categorized using the existing categorization system, so that investment transactions have the same category information as other parser outputs.

#### Acceptance Criteria

1. WHEN the Selma_Parser processes transactions, THE System SHALL apply the existing Categorizer (direct mapping → keyword matching → AI fallback) to each transaction
2. WHEN the Selma_Parser creates Transaction objects, THE System SHALL populate category and subcategory fields using the existing categorization logic
3. WHEN the Selma_Parser outputs CSV, THE System SHALL include category and subcategory columns in the same format as CAMT and Revolut parsers
4. WHEN categorization fails for a Selma transaction, THE System SHALL assign "Uncategorized" as the category
5. WHEN the Selma_Parser completes processing, THE System SHALL log categorization statistics (successful/failed/uncategorized counts)

### Requirement 4: Consistent Output Format

**User Story:** As a user, I want all parsers to produce consistent CSV output format, so that I can process files uniformly regardless of source format.

#### Acceptance Criteria

1. WHEN any parser outputs CSV, THE System SHALL include the same column headers (including category and subcategory columns)
2. WHEN the System processes files from different bank accounts, THE System SHALL create separate consolidated files for each account
3. WHEN the System processes single files, THE System SHALL still apply categorization and produce properly formatted output
4. WHEN batch processing is used, THE System SHALL consolidate by bank account as the default behavior
5. WHEN output files are created, THE System SHALL use consistent date formatting and field ordering across all parsers

### Requirement 5: Leverage Existing Categorization System

**User Story:** As a developer, I want to reuse the existing categorization infrastructure, so that categorization behavior is consistent and maintainable.

#### Acceptance Criteria

1. WHEN PDF and Selma parsers need categorization, THE parsers SHALL use the existing Categorizer interface without modification
2. WHEN the System initializes categorization for new parsers, THE System SHALL use the same configuration (YAML files, AI settings) as existing parsers
3. WHEN categorization strategies are applied, THE System SHALL follow the same three-tier approach (direct mapping, keyword matching, AI fallback)
4. WHEN categorization results are stored, THE System SHALL use the same auto-learning mechanism as existing parsers
5. WHEN categorization errors occur, THE System SHALL use the same error handling and logging patterns as existing parsers

### Requirement 6: Bank Account Identification

**User Story:** As a user, I want the system to automatically identify bank accounts from transaction data, so that consolidation happens correctly without manual configuration.

#### Acceptance Criteria

1. WHEN the System processes CAMT files, THE System SHALL extract the account number from the filename pattern (e.g., "CAMT.053_54293249_2025-04-01_2025-04-30_1.csv" → account "54293249")
2. WHEN the System processes PDF files, THE System SHALL extract the account identifier from the document metadata or transaction data
3. WHEN the System processes Selma files, THE System SHALL use a consistent account identifier (e.g., "SELMA" or account number if available)
4. WHEN the System processes Revolut files, THE System SHALL extract account identifier from filename or transaction data
5. WHEN the System cannot determine the bank account from filename pattern or content, THE System SHALL use the base filename as the account identifier

### Requirement 7: Output File Naming and Organization

**User Story:** As a user, I want consolidated output files to have clear, consistent naming, so that I can easily identify which files contain which account data.

#### Acceptance Criteria

1. WHEN the System creates consolidated files, THE System SHALL use the format "{account_identifier}_{start_date}_{end_date}.csv"
2. WHEN the System processes files with overlapping date ranges, THE System SHALL use the overall date range spanning all input files
3. WHEN the System creates output files, THE System SHALL sanitize account identifiers to be filesystem-safe
4. WHEN the batch output directory is specified, THE System SHALL organize consolidated files in the output directory
5. WHEN the System overwrites existing consolidated files, THE System SHALL overwrite without prompting (simpler behavior)
