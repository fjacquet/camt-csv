package logging

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMockLogger(t *testing.T) {
	logger := NewMockLogger()
	require.NotNil(t, logger)
	assert.NotNil(t, logger.mu)
	assert.NotNil(t, logger.entries)
	assert.Empty(t, *logger.entries)
}

func TestMockLogger_ImplementsInterface(t *testing.T) {
	var _ Logger = (*MockLogger)(nil)
}

func TestMockLogger_Debug(t *testing.T) {
	logger := NewMockLogger()
	logger.Debug("debug msg", Field{Key: "k", Value: "v"})

	entries := logger.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "DEBUG", entries[0].Level)
	assert.Equal(t, "debug msg", entries[0].Message)
	assert.Len(t, entries[0].Fields, 1)
	assert.Equal(t, "k", entries[0].Fields[0].Key)
}

func TestMockLogger_Info(t *testing.T) {
	logger := NewMockLogger()
	logger.Info("info msg")

	entries := logger.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "INFO", entries[0].Level)
	assert.Equal(t, "info msg", entries[0].Message)
}

func TestMockLogger_Warn(t *testing.T) {
	logger := NewMockLogger()
	logger.Warn("warn msg", Field{Key: "a", Value: 1})

	entries := logger.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "WARN", entries[0].Level)
	assert.Equal(t, "warn msg", entries[0].Message)
}

func TestMockLogger_Error(t *testing.T) {
	logger := NewMockLogger()
	logger.Error("error msg")

	entries := logger.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "ERROR", entries[0].Level)
}

func TestMockLogger_Fatal(t *testing.T) {
	logger := NewMockLogger()
	logger.Fatal("fatal msg", Field{Key: "code", Value: 1})

	entries := logger.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "FATAL", entries[0].Level)
	assert.Equal(t, "fatal msg", entries[0].Message)
}

func TestMockLogger_Fatalf(t *testing.T) {
	logger := NewMockLogger()
	logger.Fatalf("fatal %s %d", "msg", 42)

	entries := logger.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "FATAL", entries[0].Level)
	assert.Equal(t, "fatal msg 42", entries[0].Message)
}

func TestMockLogger_WithError(t *testing.T) {
	logger := NewMockLogger()
	testErr := errors.New("test error")

	child := logger.WithError(testErr)
	child.Error("something failed")

	entries := logger.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, testErr, entries[0].Error)
	assert.Equal(t, "something failed", entries[0].Message)
}

func TestMockLogger_WithField(t *testing.T) {
	logger := NewMockLogger()

	child := logger.WithField("user", "alice")
	child.Info("user action")

	entries := logger.GetEntries()
	require.Len(t, entries, 1)
	assert.Len(t, entries[0].Fields, 1)
	assert.Equal(t, "user", entries[0].Fields[0].Key)
	assert.Equal(t, "alice", entries[0].Fields[0].Value)
}

func TestMockLogger_WithFields(t *testing.T) {
	logger := NewMockLogger()

	child := logger.WithFields(
		Field{Key: "a", Value: 1},
		Field{Key: "b", Value: 2},
	)
	child.Info("multi fields")

	entries := logger.GetEntries()
	require.Len(t, entries, 1)
	assert.Len(t, entries[0].Fields, 2)
}

func TestMockLogger_WithErrorAndFields(t *testing.T) {
	logger := NewMockLogger()
	testErr := errors.New("err")

	child := logger.WithError(testErr).WithFields(Field{Key: "k", Value: "v"})
	child.Warn("combined")

	entries := logger.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, testErr, entries[0].Error)
	assert.Equal(t, "WARN", entries[0].Level)
}

func TestMockLogger_SharedEntries(t *testing.T) {
	logger := NewMockLogger()
	child := logger.WithField("src", "child")

	logger.Info("parent msg")
	child.Info("child msg")

	// Both entries should be visible from either logger
	entries := logger.GetEntries()
	require.Len(t, entries, 2)
}

func TestMockLogger_GetEntriesByLevel(t *testing.T) {
	logger := NewMockLogger()
	logger.Debug("d1")
	logger.Info("i1")
	logger.Info("i2")
	logger.Warn("w1")
	logger.Error("e1")

	infos := logger.GetEntriesByLevel("INFO")
	assert.Len(t, infos, 2)
	assert.Equal(t, "i1", infos[0].Message)
	assert.Equal(t, "i2", infos[1].Message)

	debugs := logger.GetEntriesByLevel("DEBUG")
	assert.Len(t, debugs, 1)

	fatals := logger.GetEntriesByLevel("FATAL")
	assert.Empty(t, fatals)
}

func TestMockLogger_Clear(t *testing.T) {
	logger := NewMockLogger()
	logger.Info("msg1")
	logger.Info("msg2")
	require.Len(t, logger.GetEntries(), 2)

	logger.Clear()
	assert.Empty(t, logger.GetEntries())
}

func TestMockLogger_HasEntry(t *testing.T) {
	logger := NewMockLogger()
	logger.Info("found msg")
	logger.Warn("warn msg")

	assert.True(t, logger.HasEntry("INFO", "found msg"))
	assert.True(t, logger.HasEntry("WARN", "warn msg"))
	assert.False(t, logger.HasEntry("INFO", "not found"))
	assert.False(t, logger.HasEntry("ERROR", "found msg"))
}

func TestMockLogger_VerifyFatalLog(t *testing.T) {
	logger := NewMockLogger()
	logger.Fatal("critical failure occurred")

	assert.True(t, logger.VerifyFatalLog("critical failure"))
	assert.True(t, logger.VerifyFatalLog("failure occurred"))
	assert.False(t, logger.VerifyFatalLog("not present"))
}

func TestMockLogger_VerifyFatalLogWithDebug(t *testing.T) {
	logger := NewMockLogger()
	testErr := errors.New("debug err")
	logger.WithError(testErr).Fatal("fatal with error", Field{Key: "ctx", Value: "test"})

	assert.True(t, logger.VerifyFatalLogWithDebug("fatal with error"))
	// When not found, prints debug output (just verify it doesn't panic)
	assert.False(t, logger.VerifyFatalLogWithDebug("nonexistent"))
}

func TestMockLogger_StructLiteralInit(t *testing.T) {
	// Test that a MockLogger created via struct literal (not NewMockLogger) works
	logger := &MockLogger{}
	logger.Info("test msg")

	entries := logger.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "test msg", entries[0].Message)
}

func TestMockLogger_NilEntries_GetMethods(t *testing.T) {
	// Test GetEntries/GetEntriesByLevel/HasEntry/VerifyFatalLog with empty entries
	logger := &MockLogger{}
	// Before any log call, entries is empty
	assert.Empty(t, logger.GetEntries())
	assert.Empty(t, logger.GetEntriesByLevel("INFO"))
	assert.False(t, logger.HasEntry("INFO", "x"))
	assert.False(t, logger.VerifyFatalLog("x"))
}
