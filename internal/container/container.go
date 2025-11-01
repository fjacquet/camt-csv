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
type Container struct {
	// Core dependencies
	Logger      logging.Logger
	Config      *config.Config
	Store       *store.CategoryStore
	AIClient    categorizer.AIClient
	Categorizer *categorizer.Categorizer
	
	// Parser registry
	Parsers map[ParserType]parser.FullParser
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
		cfg.Categories.DebitorsFile,
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
	parsers[CAMT] = camtparser.NewAdapter(logger)
	
	// PDF parser - needs special handling for extractor
	parsers[PDF] = pdfparser.NewAdapter(logger, nil) // nil for real extractor
	
	// Revolut parser
	parsers[Revolut] = revolutparser.NewAdapter(logger)
	
	// Revolut Investment parser
	parsers[RevolutInvestment] = revolutinvestmentparser.NewAdapter(logger)
	
	// Selma parser
	parsers[Selma] = selmaparser.NewAdapter(logger)
	
	// Debit parser
	parsers[Debit] = debitparser.NewAdapter(logger)
	
	logger.Info("Container initialized successfully",
		logging.Field{Key: "parsers_count", Value: len(parsers)},
		logging.Field{Key: "ai_enabled", Value: cfg.AI.Enabled})
	
	return &Container{
		Logger:      logger,
		Config:      cfg,
		Store:       categoryStore,
		AIClient:    aiClient,
		Categorizer: cat,
		Parsers:     parsers,
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
	p, ok := c.Parsers[pt]
	if !ok {
		return nil, fmt.Errorf("unknown parser type: %s", pt)
	}
	return p, nil
}

// GetLogger returns the container's logger instance.
// This is a convenience method for accessing the logger.
func (c *Container) GetLogger() logging.Logger {
	return c.Logger
}

// GetConfig returns the container's configuration instance.
// This is a convenience method for accessing the configuration.
func (c *Container) GetConfig() *config.Config {
	return c.Config
}

// GetCategorizer returns the container's categorizer instance.
// This is a convenience method for accessing the categorizer.
func (c *Container) GetCategorizer() *categorizer.Categorizer {
	return c.Categorizer
}

// GetStore returns the container's category store instance.
// This is a convenience method for accessing the store.
func (c *Container) GetStore() *store.CategoryStore {
	return c.Store
}

// GetAIClient returns the container's AI client instance.
// Returns nil if AI is not enabled.
func (c *Container) GetAIClient() categorizer.AIClient {
	return c.AIClient
}

// Close performs cleanup of container resources.
// This method should be called when the container is no longer needed.
func (c *Container) Close() error {
	// Currently no resources need explicit cleanup
	// This method is provided for future extensibility
	c.Logger.Info("Container closed")
	return nil
}