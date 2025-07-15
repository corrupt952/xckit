package test

import (
	"os"
	"path/filepath"
	"testing"
)

// TempFile creates a temporary file with the given content for testing
func TempFile(t *testing.T, name, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, name)

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	return filePath
}

// FixturePath returns the path to a test fixture file
func FixturePath(filename string) string {
	return filepath.Join("../fixtures", filename)
}

// AssertNoError is a helper to check that no error occurred
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError is a helper to check that an error occurred
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

// AssertEqual is a helper to check equality
func AssertEqual(t *testing.T, actual, expected interface{}) {
	t.Helper()
	if actual != expected {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

// AssertSliceEqual is a helper to check slice equality
func AssertSliceEqual(t *testing.T, actual, expected []string) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Errorf("expected slice length %d, got %d", len(expected), len(actual))
		return
	}
	for i := range actual {
		if actual[i] != expected[i] {
			t.Errorf("at index %d: expected %v, got %v", i, expected[i], actual[i])
		}
	}
}
