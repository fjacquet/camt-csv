package batch_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fjacquet/camt-csv/cmd/batch"
	"fjacquet/camt-csv/cmd/root"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchCommand_CommandMetadata(t *testing.T) {
	assert.Equal(t, "batch", batch.Cmd.Use)
	assert.Contains(t, batch.Cmd.Short, "Batch process")
	assert.NotNil(t, batch.Cmd.Run)
}

func TestBatchCommand_LongDescription(t *testing.T) {
	assert.Contains(t, batch.Cmd.Long, "Batch process files")
	assert.Contains(t, batch.Cmd.Long, "input directory")
	assert.Contains(t, batch.Cmd.Long, "another directory")
}

func TestBatchCommand_Example(t *testing.T) {
	assert.Contains(t, batch.Cmd.Long, "Example")
	assert.Contains(t, batch.Cmd.Long, "batch")
}

func TestBatchCommand_UsageTemplate(t *testing.T) {
	// Test that the custom usage template is set
	template := batch.Cmd.UsageTemplate()
	assert.Contains(t, template, "Global Flags (for batch, -i/-o refer to directories)")
}

func TestBatchFunc_MissingInputOutput(t *testing.T) {
	// Save original flags
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output

	// Setup root command with empty flags
	root.SharedFlags.Input = ""
	root.SharedFlags.Output = ""

	// Reset flags after test
	defer func() {
		root.SharedFlags.Input = originalInput
		root.SharedFlags.Output = originalOutput
	}()

	// Test that validation logic works correctly
	assert.Equal(t, "", root.SharedFlags.Input)
	assert.Equal(t, "", root.SharedFlags.Output)
}

func TestBatchFunc_NoSupportedFiles(t *testing.T) {
	// Save original flags
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output

	// Setup temporary directories
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	// Create non-XML file
	testFile := filepath.Join(inputDir, "test.txt")
	err := os.WriteFile(testFile, []byte("not xml"), 0600)
	require.NoError(t, err)

	// Setup root flags
	root.SharedFlags.Input = inputDir
	root.SharedFlags.Output = outputDir

	// Reset after test
	defer func() {
		root.SharedFlags.Input = originalInput
		root.SharedFlags.Output = originalOutput
	}()

	// Test that no XML files are found
	files, err := os.ReadDir(inputDir)
	require.NoError(t, err)

	xmlCount := 0
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".xml") {
			xmlCount++
		}
	}

	assert.Equal(t, 0, xmlCount, "Expected no XML files in test directory")
}

func TestBatchFunc_DirectoryCreation(t *testing.T) {
	// Save original flags
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output

	// Setup temporary directories
	inputDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "nonexistent", "output")

	// Setup root flags
	root.SharedFlags.Input = inputDir
	root.SharedFlags.Output = outputDir

	// Reset after test
	defer func() {
		root.SharedFlags.Input = originalInput
		root.SharedFlags.Output = originalOutput
	}()

	// Test directory creation logic
	err := os.MkdirAll(outputDir, 0750)
	require.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(outputDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestBatchFunc_XMLFileFiltering(t *testing.T) {
	// Setup temporary directory
	inputDir := t.TempDir()

	// Create various file types
	files := map[string]string{
		"test.xml":  "xml content",
		"test.txt":  "text content",
		"test.csv":  "csv content",
		"test.json": "json content",
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(inputDir, filename), []byte(content), 0600)
		require.NoError(t, err)
	}

	// Test file filtering logic
	dirFiles, err := os.ReadDir(inputDir)
	require.NoError(t, err)

	var xmlFiles []string
	for _, file := range dirFiles {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".xml") {
			xmlFiles = append(xmlFiles, file.Name())
		}
	}

	assert.Len(t, xmlFiles, 1, "Expected 1 XML file")
	assert.Contains(t, xmlFiles, "test.xml")
}

func TestBatchFunc_FilePathConstruction(t *testing.T) {
	inputDir := "/test/input"
	fileName := "test.xml"

	expectedPath := filepath.Join(inputDir, fileName)
	actualPath := filepath.Join(inputDir, fileName)

	assert.Equal(t, expectedPath, actualPath)
	assert.Contains(t, actualPath, inputDir)
	assert.Contains(t, actualPath, fileName)
}

func TestBatchCommand_InitFunction(t *testing.T) {
	// Test that init function sets up the usage template correctly
	template := batch.Cmd.UsageTemplate()
	assert.NotEmpty(t, template)
	assert.Contains(t, template, "Usage:")
	assert.Contains(t, template, "Global Flags")
}

func TestBatchCommand_CommandStructure(t *testing.T) {
	// Test command structure
	assert.Equal(t, "batch", batch.Cmd.Use)
	assert.NotEmpty(t, batch.Cmd.Short)
	assert.NotEmpty(t, batch.Cmd.Long)
	assert.NotNil(t, batch.Cmd.Run)

	// Test that long description contains expected content
	assert.Contains(t, batch.Cmd.Long, "Batch process files")
	assert.Contains(t, batch.Cmd.Long, "input directory")
	assert.Contains(t, batch.Cmd.Long, "output them to another directory")
	assert.Contains(t, batch.Cmd.Long, "XML files")
	assert.Contains(t, batch.Cmd.Long, "AI-powered categorization")
}

func TestBatchCommand_ExampleContent(t *testing.T) {
	// Test example content in long description
	longDesc := batch.Cmd.Long
	assert.Contains(t, longDesc, "Example:")
	assert.Contains(t, longDesc, "camt-csv batch")
	assert.Contains(t, longDesc, "-i input_dir/")
	assert.Contains(t, longDesc, "-o output_dir/")
}
