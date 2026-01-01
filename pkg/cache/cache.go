package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CacheKey computes a deterministic SHA256 hash of build inputs
// Order is critical: job posting, cv, letter, schema, model
func CacheKey(jobPosting, cv, letter, schema, model string) string {
	h := sha256.New()
	h.Write([]byte(jobPosting))
	h.Write([]byte(cv))
	h.Write([]byte(letter))
	h.Write([]byte(schema))
	h.Write([]byte(model))
	return hex.EncodeToString(h.Sum(nil))
}

// CachePath returns the path to the cache file for a given key
func CachePath(key string) string {
	cacheDir := filepath.Join(os.ExpandEnv("$HOME"), ".cache", "cvx", "agent", "runs")
	return filepath.Join(cacheDir, key+".json")
}

// Read reads cached output for a given key
func Read(key string) (map[string]interface{}, error) {
	path := CachePath(key)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err // file not found is expected for cache miss
	}

	var output struct {
		CV     map[string]interface{} `json:"cv"`
		Letter map[string]interface{} `json:"letter"`
	}
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("failed to parse cache: %w", err)
	}

	return map[string]interface{}{
		"cv":     output.CV,
		"letter": output.Letter,
	}, nil
}

// Write writes output to cache for a given key
func Write(key string, cvOut, letterOut map[string]interface{}) error {
	path := CachePath(key)

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create cache dir: %w", err)
	}

	output := map[string]interface{}{
		"cv":     cvOut,
		"letter": letterOut,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}

// Exists checks if cache exists for a key
func Exists(key string) bool {
	_, err := os.Stat(CachePath(key))
	return err == nil
}
