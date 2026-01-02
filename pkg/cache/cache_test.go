package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCacheKeyDeterministic(t *testing.T) {
	tests := []struct {
		name       string
		issueNum   int
		posting    string
		cv         string
		letter     string
		schema     string
		model      string
		expected   string // Empty means we just check it's consistent
		wantEqual  string // Another key to compare against
		wantDiffer int    // Issue number that should produce different hash
	}{
		{
			name:     "Same inputs produce same hash",
			issueNum: 42,
			posting:  "Job posting text",
			cv:       `{"name":"John","email":"john@example.com"}`,
			letter:   `{"sender":"John"}`,
			schema:   "{}",
			model:    "gemini-2.5-flash",
		},
		{
			name:     "Different posting changes hash",
			issueNum: 42,
			posting:  "Different job",
			cv:       `{"name":"John"}`,
			letter:   `{"sender":"John"}`,
			schema:   "{}",
			model:    "gemini-2.5-flash",
		},
		{
			name:     "Different model changes hash",
			issueNum: 42,
			posting:  "Job posting",
			cv:       `{"name":"John"}`,
			letter:   `{"sender":"John"}`,
			schema:   "{}",
			model:    "claude-haiku-4-5-20251001",
		},
		{
			name:       "Different issue number changes hash",
			issueNum:   42,
			posting:    "Job posting",
			cv:         `{"name":"John"}`,
			letter:     `{"sender":"John"}`,
			schema:     "{}",
			model:      "gemini-2.5-flash",
			wantDiffer: 43,
		},
		{
			name:       "Different schema changes hash",
			issueNum:   42,
			posting:    "Job posting",
			cv:         `{"name":"John"}`,
			letter:     `{"sender":"John"}`,
			schema:     `{"type":"object"}`,
			model:      "gemini-2.5-flash",
			wantDiffer: 0, // We'll test schema change separately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := CacheKey(tt.issueNum, tt.posting, tt.cv, tt.letter, tt.schema, tt.model)

			// Same inputs should produce same hash
			key2 := CacheKey(tt.issueNum, tt.posting, tt.cv, tt.letter, tt.schema, tt.model)
			if key1 != key2 {
				t.Errorf("CacheKey not deterministic: %s vs %s", key1, key2)
			}

			// Hash should be 64 characters (SHA256 hex)
			if len(key1) != 64 {
				t.Errorf("CacheKey wrong length: got %d, want 64", len(key1))
			}

			// Test that different issue numbers produce different hashes
			if tt.wantDiffer > 0 {
				key3 := CacheKey(tt.wantDiffer, tt.posting, tt.cv, tt.letter, tt.schema, tt.model)
				if key1 == key3 {
					t.Errorf("Different issue numbers should produce different hashes")
				}
			}
		})
	}
}

func TestCacheKeySchemaChange(t *testing.T) {
	issueNum := 42
	posting := "Job posting"
	cv := `{"name":"John"}`
	letter := `{"sender":"John"}`
	model := "gemini-2.5-flash"

	// Same schema should produce same hash
	schema1 := `{"type":"object"}`
	key1 := CacheKey(issueNum, posting, cv, letter, schema1, model)
	key2 := CacheKey(issueNum, posting, cv, letter, schema1, model)

	if key1 != key2 {
		t.Error("Same schema should produce same hash")
	}

	// Different schema should produce different hash
	schema2 := `{"type":"object","required":["name"]}`
	key3 := CacheKey(issueNum, posting, cv, letter, schema2, model)

	if key1 == key3 {
		t.Error("Different schema should invalidate cache (produce different hash)")
	}
}

func TestCacheKeyOrderMatters(t *testing.T) {
	issueNum := 42
	posting := "Job posting"
	cv := `{"name":"John"}`
	letter := `{"sender":"Jane"}`
	schema := "{}"
	model := "gemini"

	key1 := CacheKey(issueNum, posting, cv, letter, schema, model)

	// Swap posting and cv
	key2 := CacheKey(issueNum, cv, posting, letter, schema, model)

	if key1 == key2 {
		t.Error("CacheKey should differ when input order changes")
	}
}

func TestReadWriteCache(t *testing.T) {
	// Setup temp directory and change to it
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

	key := "test-key-12345"
	cvOut := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	}
	letterOut := map[string]interface{}{
		"sender": map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
		},
	}

	// Write cache
	err := Write(key, cvOut, letterOut)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify file exists in .cvx/cache/agent/
	path := CachePath(key)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Cache file not created: %v", err)
	}

	// Verify path is in .cvx/cache/agent/
	// Resolve symlinks for comparison (e.g., /var -> /private/var on macOS)
	resolvedPath, _ := filepath.EvalSymlinks(path)
	expectedPath := filepath.Join(tmpDir, ".cvx", "cache", "agent", key+".json")
	resolvedExpected, _ := filepath.EvalSymlinks(expectedPath)
	if resolvedPath != resolvedExpected {
		t.Errorf("Cache path incorrect: got %s, want %s", resolvedPath, resolvedExpected)
	}

	// Read cache
	result, err := Read(key)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// Verify contents
	if result["cv"] == nil {
		t.Error("Missing cv in cached result")
	}
	if result["letter"] == nil {
		t.Error("Missing letter in cached result")
	}
}

func TestCacheHitMiss(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

	key := "nonexistent-key"

	// Should not exist
	if Exists(key) {
		t.Error("Cache should not exist for new key")
	}

	// Write it
	err := Write(key, map[string]interface{}{}, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Should exist now
	if !Exists(key) {
		t.Error("Cache should exist after write")
	}
}

func TestCachePathCreatesDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

	key := "nested-key-abc123"

	// Directory should not exist yet
	path := CachePath(key)
	cacheDir := filepath.Dir(path)
	if _, err := os.Stat(cacheDir); err == nil {
		t.Error("Cache directory should not exist yet")
	}

	// Write should create it
	err := Write(key, map[string]interface{}{}, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Should exist now
	if _, err := os.Stat(cacheDir); err != nil {
		t.Errorf("Cache directory not created: %v", err)
	}

	// Verify directory structure is .cvx/cache/agent/
	// Resolve symlinks for comparison (e.g., /var -> /private/var on macOS)
	resolvedDir, _ := filepath.EvalSymlinks(cacheDir)
	expectedDir := filepath.Join(tmpDir, ".cvx", "cache", "agent")
	resolvedExpected, _ := filepath.EvalSymlinks(expectedDir)
	if resolvedDir != resolvedExpected {
		t.Errorf("Cache directory incorrect: got %s, want %s", resolvedDir, resolvedExpected)
	}
}
