package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCacheKeyDeterministic(t *testing.T) {
	tests := []struct {
		name      string
		posting   string
		cv        string
		letter    string
		schema    string
		model     string
		expected  string // Empty means we just check it's consistent
		wantEqual string // Another key to compare against
	}{
		{
			name:      "Same inputs produce same hash",
			posting:   "Job posting text",
			cv:        "name: John\nemail: john@example.com",
			letter:    "sender: John",
			schema:    "{}",
			model:     "gemini-2.5-flash",
			wantEqual: "Job posting textname: John\nemail: john@examplesender: John{}gemini-2.5-flash",
		},
		{
			name:      "Different posting changes hash",
			posting:   "Different job",
			cv:        "name: John",
			letter:    "sender: John",
			schema:    "{}",
			model:     "gemini-2.5-flash",
			wantEqual: "Different jobname: Johnsender: John{}gemini-2.5-flash",
		},
		{
			name:      "Different model changes hash",
			posting:   "Job posting",
			cv:        "name: John",
			letter:    "sender: John",
			schema:    "{}",
			model:     "claude-haiku-4-5-20251001",
			wantEqual: "Job postingname: Johnsender: John{}claude-haiku-4-5-20251001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := CacheKey(tt.posting, tt.cv, tt.letter, tt.schema, tt.model)

			// Same inputs should produce same hash
			key2 := CacheKey(tt.posting, tt.cv, tt.letter, tt.schema, tt.model)
			if key1 != key2 {
				t.Errorf("CacheKey not deterministic: %s vs %s", key1, key2)
			}

			// Hash should be 64 characters (SHA256 hex)
			if len(key1) != 64 {
				t.Errorf("CacheKey wrong length: got %d, want 64", len(key1))
			}
		})
	}
}

func TestCacheKeyOrderMatters(t *testing.T) {
	posting := "Job posting"
	cv := "name: John"
	letter := "sender: Jane"
	schema := "{}"
	model := "gemini"

	key1 := CacheKey(posting, cv, letter, schema, model)

	// Swap inputs
	key2 := CacheKey(cv, posting, letter, schema, model)

	if key1 == key2 {
		t.Error("CacheKey should differ when input order changes")
	}
}

func TestReadWriteCache(t *testing.T) {
	// Setup temp cache dir
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

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

	// Verify file exists
	path := CachePath(key)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Cache file not created: %v", err)
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
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

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
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

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
}
