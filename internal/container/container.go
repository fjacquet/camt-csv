// Package container provides dependency injection for the camt-csv application.
// It centralizes the creation and wiring of all application dependencies,
// making them explicit and testable.
package container

import (
	"fmt"

	"fjacquet/camt-csv/internal/camtparser"
	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/config"
	"fjacquet/camt-csv/internal/debitparser"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/parser"
	"fjacquet/camt-csv/internal/pdfparser"
	"fjacquet/camt-csv/internal/revolutinvestmentparser"
	"fjacquet/camt-csv/internal/revolutparser"
	"fjacquet/camt-csv/internal/selmaparser"
	"fjacquet/camt-csv/internal/store"
)

// ParserType defines the types of parsers available.
type ParserType string

const (
	CAMT              ParserType = "camt"
	PDF               ParserType = "pdf"
	Revolut           ParserType = "revolut"
	RevolutInvestment ParserType = "revolut-investment"
	Selma             ParserType = "selma"
	Debit             ParserType = "debit"
)

// Container holds all application dependencies and provides methods to access them.
// It acts as the central registry for dependency injection, ensuring that all
// components receive their required dependencies through constructors.
//
// Container is immutable after creation - all fields are private and can only
// be accessed through getter methods. This prevents accidental modification
// of dependencies after initialization.
type Container struct {
	// Core dependencies (private for immutability)
	logger      logging.Logger
	config      *config.Config
	store       *store.CategoryStore
	aiClient    categorizer.AIClient
	categorizer *categorizer.Categorizer

	// Parser registry (private for immutability)
	parsers map[ParserType]parser.FullParser
}

// NewContainer creates and wires all application dependencies.
// This is the main entry point for dependency injection in the application.
//
// Parameters:
//   - cfg: Application configuration
//
// Returns:
//   - *Container: Fully wired container with all dependencies
//   - error: Any error encountered during dependency creation
func NewContainer(cfg *config.Config) (*Container, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	// Create logger first as it's needed by other components
	logger := logging.NewLogrusAdapter(cfg.Log.Level, cfg.Log.Format)

	// Create category store
	categoryStore := store.NewCategoryStore(
		cfg.Categories.File,
		cfg.Categories.CreditorsFile,
		cfg.Categories.DebtorsFile,
	)

	// Create AI client (if enabled)
	var aiClient categorizer.AIClient
	if cfg.AI.Enabled && cfg.AI.APIKey != "" {
		aiClient = categorizer.NewGeminiClient(logger)
		logger.Info("AI categorization enabled")
	} else {
		logger.Info("AI categorization disabled")
	}

	// Create categorizer with all dependencies
	cat := categorizer.NewCategorizer(aiClient, categoryStore, logger)

	// Create parsers with dependency injection
	parsers := make(map[ParserType]parser.FullParser)

	// CAMT parser
	camtParser := camtparser.NewAdapter(logger)
	camtParser.SetCategorizer(cat)
	parsers[CAMT] = camtParser

	// PDF parser - needs special handling for extractor
	pdfParser := pdfparser.NewAdapter(logger, nil) // nil for real extractor
	pdfParser.SetCategorizer(cat)
	parsers[PDF] = pdfParser

	// Revolut parser
	revolutParser := revolutparser.NewAdapter(logger)
	revolutParser.SetCategorizer(cat)
	parsers[Revolut] = revolutParser

	// Revolut Investment parser
	revolutInvestmentParser := revolutinvestmentparser.NewAdapter(logger)
	revolutInvestmentParser.SetCategorizer(cat)
	parsers[RevolutInvestment] = revolutInvestmentParser

	// Selma parser
	selmaParser := selmaparser.NewAdapter(logger)
	selmaParser.SetCategorizer(cat)
	parsers[Selma] = selmaParser

	// Debit parser
	debitParser := debitparser.NewAdapter(logger)
	debitParser.SetCategorizer(cat)
	parsers[Debit] = debitParser

	logger.Info("Container initialized successfully",
		logging.Field{Key: "parsers_count", Value: len(parsers)},
		logging.Field{Key: "ai_enabled", Value: cfg.AI.Enabled})

	return &Container{
		logger:      logger,
		config:      cfg,
		store:       categoryStore,
		aiClient:    aiClient,
		categorizer: cat,
		parsers:     parsers,
	}, nil
}

// GetParser returns a parser for the given type.
// This method provides type-safe access to parser instances.
//
// Parameters:
//   - pt: The type of parser to retrieve
//
// Returns:
//   - parser.FullParser: The requested parser instance
//   - error: Error if parser type is unknown
func (c *Container) GetParser(pt ParserType) (parser.FullParser, error) {
	p, ok := c.parsers[pt]
	if !ok {
		return nil, fmt.Errorf("unknown parser type: %s", pt)
	}
	return p, nil
}

// GetParsers returns a copy of the parser registry.
// This prevents external modification of the internal parser map.
func (c *Container) GetParsers() map[ParserType]parser.FullParser {
	// Return a copy to maintain immutability
	result := make(map[ParserType]parser.FullParser, len(c.parsers))
	for k, v := range c.parsers {
		result[k] = v
	}
	return result
}

// GetLogger returns the container's logger instance.
// This is a convenience method for accessing the logger.
func (c *Container) GetLogger() logging.Logger {
	return c.logger
}

// GetConfig returns the container's configuration instance.
// This is a convenience method for accessing the configuration.
func (c *Container) GetConfig() *config.Config {
	return c.config
}

// GetCategorizer returns the container's categorizer instance.
// This is a convenience method for accessing the categorizer.
func (c *Container) GetCategorizer() *categorizer.Categorizer {
	return c.categorizer
}

// GetStore returns the container's category store instance.
// This is a convenience method for accessing the store.
func (c *Container) GetStore() *store.CategoryStore {
	return c.store
}

// GetAIClient returns the container's AI client instance.
// Returns nil if AI is not enabled.
func (c *Container) GetAIClient() categorizer.AIClient {
	return c.aiClient
}

// Close performs cleanup of container resources.
// This method should be called when the container is no longer needed.
func (c *Container) Close() error {
	// Currently no resources need explicit cleanup
	// This method is provided for future extensibility
	c.logger.Info("Container closed")
	return nil
}
