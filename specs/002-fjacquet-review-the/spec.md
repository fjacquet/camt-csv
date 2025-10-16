# Feature Specification: Codebase Improvements: Parser Factory, Custom Errors, Testable Categorization, Standardized Logging

**Feature Branch**: `002-fjacquet-review-the`
**Created**: mardi 14 octobre 2025
**Status**: Draft
**Input**: Uil s'agit d'un projet bien structuré et mature qui suit de nombreuses bonnes pratiques. L'architecture est propre, la documentation est complète et la stratégie de test est solide. Cependant, même les bons projets peuvent être améliorés. Voici quelques suggestions basées sur les fichiers et les modèles de votre espace de travail : ### 1. Centraliser la Logique de Création des Analyseurs (Parser Factory) Actuellement, différents analyseurs sont instanciés directement dans leurs commandes CLI respectives (par exemple, dans `cmd/revolut/convert.go`). Vous pourriez centraliser cette logique de création dans une "factory". Cela simplifierait l'ajout de nouveaux analyseurs et le rendrait plus maintenable. **Suggestion :** Créez une fonction `GetParser` dans un package commun qui renvoie l'implémentation de l'analyseur appropriée en fonction d'un type. `go package parser import ( "fmt" "fjacquet/camt-csv/internal/camtparser" "fjacquet/camt-csv/internal/debitparser" "fjacquet/camt-csv/internal/pdfparser" "fjacquet/camt-csv/internal/revolutinvestmentparser" "fjacquet/camt-csv/internal/revolutparser" "fjacquet/camt-csv/internal/selmaparser" ) // ParserType définit les types d'analyseurs disponibles. type ParserType string const ( CAMT ParserType = "camt" PDF ParserType = "pdf" Revolut ParserType = "revolut" RevolutInvestment ParserType = "revolut-investment" Selma ParserType = "selma" Debit ParserType = "debit" ) // GetParser renvoie une nouvelle instance de l'analyseur pour le type donné. func GetParser(parserType ParserType) (Parser, error) { switch parserType { case CAMT: return camtparser.NewAdapter(), nil case PDF: return pdfparser.NewAdapter(), nil case Revolut: return revolutparser.NewAdapter(), nil case RevolutInvestment: return revolutinvestmentparser.NewAdapter(), nil case Selma: return selmaparser.NewAdapter(), nil case Debit: return debitparser.NewAdapter(), nil default: return nil, fmt.Errorf("analyseur inconnu : %s", parserType) } }` ### 2. Améliorer la Gestion des Erreurs avec des Types d'Erreurs Personnalisés Le projet pourrait bénéficier de types d'erreurs plus spécifiques au lieu de s'appuyer principalement sur `fmt.Errorf` ou des erreurs génériques. Cela permet une gestion des erreurs plus programmatique et des messages plus clairs pour l'utilisateur. **Suggestion :** Définissez des erreurs personnalisées dans un package `parsererror`. `go package parsererror import "fmt" // InvalidFormatError est renvoyé lorsqu'un fichier ne correspond pas au format attendu. type InvalidFormatError struct { FilePath string ExpectedFormat string Err error } func (e *InvalidFormatError) Error() string { return fmt.Sprintf("le fichier '%s' n'est pas un format %s valide", e.FilePath, e.ExpectedFormat) } func (e *InvalidFormatError) Unwrap() error { return e.Err } // DataExtractionError est renvoyé lorsqu'une donnée spécifique ne peut pas être extraite. type DataExtractionError struct { FieldName string RawData string Err error } func (e *DataExtractionError) Error() string { return fmt.Sprintf("échec de l'extraction du champ '%s' à partir des données : %s", e.FieldName, e.RawData) } func (e *DataExtractionError) Unwrap() error { return e.Err }` ### 3. Refactoriser la Logique de Catégorisation pour une Meilleure Testabilité Le package categorizer.go mélange la logique de catégorisation pure (correspondance de mots-clés) avec des effets de bord (appels API Gemini, lecture/écriture de fichiers). En séparant ces préoccupations, vous pouvez rendre la logique de base plus facile à tester et à maintenir. **Suggestion :** Extrayez la logique de l'API Gemini dans une interface et une implémentation distinctes. `go package categorizer import "fjacquet/camt-csv/internal/models" // AIClient définit l'interface pour un client de catégorisation IA. type AIClient interface { CategorizeTransaction(tx models.Transaction, availableCategories []string) (string, error) // Potentiellement, une méthode pour vérifier la connectivité Ping() error } // GeminiClient est une implémentation de AIClient pour l'API Gemini. type GeminiClient struct { // ... champs pour la clé API, le modèle, le client http, le limiteur de débit } func NewGeminiClient(apiKey, model string, requestsPerMinute int) AIClient { // ... initialisation du client Gemini return &GeminiClient{} } func (c *GeminiClient) CategorizeTransaction(tx models.Transaction, availableCategories []string) (string, error) { // ... logique existante pour appeler l'API Gemini return "", nil } func (c *GeminiClient) Ping() error { // ... logique pour tester la connectivité à l'API Gemini return nil }` Vous pouvez ensuite injecter `AIClient` dans votre `Categorizer`, ce qui facilite l'utilisation d'un client factice dans les tests. ### 4. Standardiser la Journalisation (Logging) Le projet utilise `logrus` de manière cohérente, ce qui est excellent. Cependant, vous pourriez standardiser davantage les champs de journalisation pour améliorer l'observabilité et faciliter l'analyse des logs. **Suggestion :** Définissez des constantes pour les noms de champs de journalisation courants. `go package logging // Constantes pour les champs de journalisation structurés const ( FieldParser = "parser" FieldFile = "file_path" FieldOperation = "operation" FieldDurationMs = "duration_ms" FieldError = "error" FieldComponent = "component" FieldCount = "count" )` Utilisez ensuite ces constantes dans votre code : `log.WithField(logging.FieldParser, "camt")...` Ces améliorations s'appuient sur les excellentes bases que vous avez déjà mises en place et peuvent contribuer à rendre le projet encore plus robuste et maintenable à long terme."

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Centralized Parser Management (Priority: P1)

As a developer, I want to easily add new parsers or modify existing parser instantiation logic in a single, centralized location, so that the codebase is more maintainable and extensible.

**Why this priority**: This improvement directly addresses maintainability and extensibility, which are foundational for future development and reducing technical debt.

**Independent Test**: Can be fully tested by adding a new dummy parser type and verifying that it can be instantiated via the factory without changes to existing CLI commands, and delivers a more organized and extensible parser management system.

**Acceptance Scenarios**:

1. **Given** a new parser type is introduced, **When** a developer adds it to the `GetParser` function, **Then** the new parser can be instantiated without modifying individual CLI commands.
2. **Given** an existing parser's instantiation logic needs modification, **When** a developer updates the `GetParser` function, **Then** all consumers of the parser factory automatically use the updated logic.

---

### User Story 2 - Clearer Error Handling (Priority: P1)

As a developer, I want to receive specific and programmatic error types when parsing or data extraction fails, so that I can implement more robust error handling and provide clearer feedback to users.

**Why this priority**: Improved error handling leads to more stable applications and better user experience by providing actionable error messages, reducing debugging time.

**Independent Test**: Can be fully tested by providing malformed input files or files with missing data and asserting that the correct custom error types (`InvalidFormatError`, `DataExtractionError`) are returned with appropriate details, and delivers more precise error reporting.

**Acceptance Scenarios**:

1. **Given** a file with an invalid format is processed, **When** the parser encounters the invalid format, **Then** an `InvalidFormatError` is returned with details about the file path and expected format.
2. **Given** a valid file where specific data cannot be extracted, **When** the parser attempts to extract the data, **Then** a `DataExtractionError` is returned with details about the missing field and raw data.

---

### User Story 3 - Testable AI Categorization Logic (Priority: P2)

As a developer, I want to easily test the core categorization logic without making actual calls to the Gemini API, so that I can write faster, more reliable unit tests for the categorizer.

**Why this priority**: Enhances testability and allows for isolated testing of business logic, which is crucial for code quality and faster development cycles.

**Independent Test**: Can be fully tested by refactoring the `Categorizer` to accept an `AIClient` interface and then providing a mock implementation of `AIClient` in unit tests to verify categorization logic without external API calls, and delivers a more robust and testable categorization module.

**Acceptance Scenarios**:

1. **Given** the `Categorizer` depends on an `AIClient` interface, **When** a developer provides a mock `AIClient` implementation during testing, **Then** the `Categorizer` can be tested in isolation without external API calls.
2. **Given** the `GeminiClient` implements the `AIClient` interface, **When** the `Categorizer` is used in production, **Then** it correctly interacts with the Gemini API.

---

### User Story 4 - Standardized Logging (Priority: P3)

As a developer, I want to use consistent field names for structured logging across the application, so that logs are easier to parse, filter, and analyze for debugging and monitoring.

**Why this priority**: Improves observability and simplifies log analysis, which is important for troubleshooting and understanding application behavior in production.

**Independent Test**: Can be fully tested by modifying existing log statements to use the new constants and verifying that log outputs consistently use the standardized field names, and delivers more consistent and analyzable logs.

**Acceptance Scenarios**:

1. **Given** a log message needs to include a parser name, **When** the `logging.FieldParser` constant is used, **Then** the log entry consistently uses "parser" as the field name.
2. **Given** a log message needs to include a file path, **When** the `logging.FieldFile` constant is used, **Then** the log entry consistently uses "file_path" as the field name.

---

### Edge Cases

- What happens when an unknown `ParserType` is requested from the factory? The factory should return an error indicating the parser is unknown.
- How does the system handle partial data extraction failures within a valid file? A `DataExtractionError` should be returned for the specific field, allowing other valid data to be processed if possible.
- What if the Gemini API is unavailable or returns an error during categorization? The `GeminiClient` should handle this gracefully, potentially returning an error or a default category.
- How does the system behave if a required logging field constant is not used? The log will still be generated, but the field name will not be standardized, potentially impacting log analysis.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The system MUST provide a centralized mechanism for instantiating different parser implementations based on a specified type.
- **FR-002**: The system MUST define custom error types for common parsing and data extraction failures.
- **FR-003**: The system MUST use the custom error types (`InvalidFormatError`, `DataExtractionError`) when relevant parsing or data extraction issues occur.
- **FR-004**: The categorization logic MUST be refactored to depend on an `AIClient` interface for AI-based categorization.
- **FR-005**: An implementation of `AIClient` for the Gemini API (`GeminiClient`) MUST be provided.
- **FR-006**: The system MUST define a set of standardized constants for common structured logging field names.
- **FR-007**: The logging mechanism MUST encourage the use of these standardized logging field constants.

### Non-Functional Requirements

- **NFR-001**: Performance: Default Go application performance is assumed to be sufficient for the current scope.

### Key Entities _(include if feature involves data)_

- **Parser**: Represents a component responsible for converting input data (e.g., CAMT, PDF, Revolut files) into a standardized transaction format.
- **ParserType**: An enumeration or string type representing the different kinds of parsers available.
- **Error**: Custom error types (`InvalidFormatError`, `DataExtractionError`) providing specific context about parsing failures.
- **AIClient**: An interface defining the contract for AI-based categorization services.
- **GeminiClient**: A concrete implementation of the `AIClient` interface that interacts with the Gemini API.
- **Transaction**: A standardized data structure representing a financial transaction.
- **Log Field Constants**: Predefined string constants for consistent naming of structured log fields.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Developers can integrate a new parser into the system by modifying only the parser factory and the new parser's implementation, reducing changes in CLI commands by 100%.
- **SC-002**: Error messages related to parsing and data extraction provide specific context (file path, expected format, field name, raw data) in 100% of cases where custom errors are applicable.
- **SC-003**: Unit tests for the core categorization logic can be executed without external network calls, improving test execution speed by at least 50% for relevant tests.
- **SC-004**: All new structured log entries adhere to the standardized field names, achieving 100% consistency for defined fields.
- **SC-005**: The codebase's maintainability index (as measured by a suitable metric, e.g., cyclomatic complexity of parser instantiation logic) improves by at least 15% after implementing the parser factory.

## Clarifications

### Session 2025-10-14
- Q: Should specific performance targets be defined for this feature? → A: Assume default Go application performance is sufficient.
- Q: For managing the Gemini API key, which security mechanism should be implemented? → A: Load the API key from an environment variable (e.g., GEMINI_API_KEY).

## Assumptions

- The existing parsers (CAMT, PDF, Revolut, etc.) can be adapted to a common `Parser` interface.
- The `Categorizer` currently uses a direct call to Gemini API or similar AI service.
- The `logrus` logging library is used consistently across the project.
- The project has a defined `Parser` interface that all concrete parsers implement.
- The Gemini API key is securely managed and made available as an environment variable.
