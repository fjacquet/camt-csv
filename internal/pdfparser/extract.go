package pdfparser

import (
	"fjacquet/camt-csv/internal/logging"
	"fmt"
	"os"
	"os/exec"
)

// getDefaultLogger returns a default logger for backward compatibility
func getDefaultLogger() logging.Logger {
	return logging.NewLogrusAdapter("info", "text")
}

func extractTextFromPDFImpl(pdfFile string) (string, error) {
	// SECURITY: Create a temporary file with random unpredictable name to store the text output
	// This prevents attacks that rely on predictable temp file names
	tempFile, err := os.CreateTemp("", "pdftext-*.txt")
	if err != nil {
		return "", fmt.Errorf("error creating temporary file: %w", err)
	}
	tempFileName := tempFile.Name()

	// Close the file immediately - pdftotext will open and write to it
	if err := tempFile.Close(); err != nil {
		return "", fmt.Errorf("error closing temporary file: %w", err)
	}

	// Ensure cleanup on return
	defer func() {
		if err := os.Remove(tempFileName); err != nil {
			getDefaultLogger().WithError(err).Warn("Failed to remove temporary file")
		}
	}()

	// Use pdftotext command-line tool to extract text
	// Add the -raw option to preserve the original text layout
	cmd := exec.Command("pdftotext", "-layout", "-raw", pdfFile, tempFileName) // #nosec G204,G702 -- Expected subprocess for PDF text extraction
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running pdftotext: %w", err)
	}

	// Read the extracted text
	output, err := os.ReadFile(tempFileName) // #nosec G304,G703 -- reading from app-generated temp file
	if err != nil {
		return "", fmt.Errorf("error reading extracted text: %w", err)
	}

	return string(output), nil
}
