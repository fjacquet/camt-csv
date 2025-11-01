# Product Overview

CAMT-CSV is a command-line tool that converts various financial statement formats into standardized CSV files with intelligent transaction categorization.

## Core Purpose

Convert financial data from multiple sources (CAMT.053 XML, PDF bank statements, Revolut CSV, Selma investment CSV, and generic debit CSV) into a unified CSV format for analysis and accounting.

## Key Features

- **Multi-format Support**: Extensible parser architecture supporting CAMT.053 XML, PDF (Viseca), Revolut, Revolut Investment, Selma, and generic debit CSV formats
- **Smart Categorization**: Three-tier hybrid approach:
  1. Direct mapping via creditor/debitor YAML files
  2. Local keyword matching from categories.yaml
  3. AI fallback using Google Gemini (optional, with auto-learning)
- **Batch Processing**: Process multiple files in a single operation
- **Hierarchical Configuration**: Viper-based system supporting config files, environment variables, and CLI flags

## Target Users

- Individuals managing personal finances across multiple banks
- Small businesses consolidating financial data
- Accountants processing client statements
- Developers building financial data pipelines

## Design Philosophy

- **Local-first**: Prioritize local processing with optional cloud AI
- **Extensibility**: Easy to add new parsers via standardized interface
- **Reliability**: Comprehensive error handling and validation
- **Privacy**: Sensitive data processed locally, AI is optional
