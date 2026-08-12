package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Manifest represents an ollama model manifest file.
type Manifest struct {
	SchemaVersion int     `json:"schemaVersion"`
	MediaType     string  `json:"mediaType"`
	Config        Layer   `json:"config"`
	Layers        []Layer `json:"layers"`
}

// Layer represents a single layer (blob reference) in a manifest.
type Layer struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
}

// ModelInfo holds information about a model in the vault or ollama.
type ModelInfo struct {
	Name string // Model name e.g. "qwen3-vl:latest"
	Tag  string // Tag portion e.g. "latest"
	Size int64  // Total size of all blobs
}

// VacuumResult holds stats from a vacuum operation.
type VacuumResult struct {
	TotalBlobs    int
	OrphanedBlobs int
	RemovedBlobs  int
	BytesFreed    int64
	TotalModels   int
	UniqueBlobs   int
	DedupRatio    float64 // Ratio of shared blobs (1.0 = all unique, >1.0 = dedup happening)
}

// DefaultOllamaModelsPath returns the default path to ollama's models directory.
func DefaultOllamaModelsPath() string {
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".ollama", "models")
	case "linux":
		// Check common locations
		if _, err := os.Stat("/usr/share/ollama/.ollama/models"); err == nil {
			return "/usr/share/ollama/.ollama/models"
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".ollama", "models")
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".ollama", "models")
	}
}

// ListModels returns all models found in the given models directory.
func ListModels(modelsPath string) ([]ModelInfo, error) {
	manifestsPath := filepath.Join(modelsPath, "manifests")
	var models []ModelInfo

	err := filepath.Walk(manifestsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Parse the path to extract model info
		relPath, _ := filepath.Rel(manifestsPath, path)
		parts := strings.Split(relPath, string(filepath.Separator))

		// Expected: registry/namespace/model/tag
		if len(parts) < 4 {
			return nil
		}

		tag := parts[len(parts)-1]
		model := parts[len(parts)-2]

		// Display name matches ollama's format: model:tag
		fullName := fmt.Sprintf("%s:%s", model, tag)

		// Calculate total size from manifest
		size, _ := getModelSize(modelsPath, path)

		models = append(models, ModelInfo{
			Name: fullName,
			Tag:  tag,
			Size: size,
		})

		return nil
	})

	return models, err
}

// getModelSize reads a manifest and sums up all blob sizes.
func getModelSize(modelsPath string, manifestPath string) (int64, error) {
	manifest, err := ReadManifest(manifestPath)
	if err != nil {
		return 0, err
	}

	var total int64
	for _, layer := range manifest.Layers {
		total += layer.Size
	}
	total += manifest.Config.Size
	return total, nil
}

// ReadManifest reads and parses a manifest file.
func ReadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// GetAllBlobs returns all blob digests referenced by manifests in the models directory.
func GetAllBlobs(modelsPath string) (map[string]int, error) {
	manifestsPath := filepath.Join(modelsPath, "manifests")
	referenced := make(map[string]int) // digest -> reference count

	err := filepath.Walk(manifestsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		manifest, err := ReadManifest(path)
		if err != nil {
			return nil
		}

		referenced[manifest.Config.Digest]++
		for _, layer := range manifest.Layers {
			referenced[layer.Digest]++
		}

		return nil
	})

	return referenced, err
}

// GetStoredBlobs returns all blobs actually present on disk.
func GetStoredBlobs(modelsPath string) (map[string]int64, error) {
	blobsPath := filepath.Join(modelsPath, "blobs")
	stored := make(map[string]int64) // digest -> size

	entries, err := os.ReadDir(blobsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return stored, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Blob files are named like "sha256-<hash>" on disk
		name := entry.Name()
		digest := strings.Replace(name, "-", ":", 1)

		info, err := entry.Info()
		if err != nil {
			continue
		}

		stored[digest] = info.Size()
	}

	return stored, nil
}

// Vacuum finds orphaned blobs and optionally removes them.
func Vacuum(modelsPath string, dryRun bool) (*VacuumResult, error) {
	referenced, err := GetAllBlobs(modelsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan manifests: %w", err)
	}

	stored, err := GetStoredBlobs(modelsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan blobs: %w", err)
	}

	// Count models
	models, err := ListModels(modelsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	result := &VacuumResult{
		TotalBlobs:  len(stored),
		TotalModels: len(models),
		UniqueBlobs: len(referenced),
	}

	// Calculate dedup ratio: total blob references / unique blobs
	totalRefs := 0
	for _, count := range referenced {
		totalRefs += count
	}
	if len(referenced) > 0 {
		result.DedupRatio = float64(totalRefs) / float64(len(referenced))
	}

	// Find orphaned blobs
	blobsPath := filepath.Join(modelsPath, "blobs")
	for digest, size := range stored {
		if _, exists := referenced[digest]; !exists {
			result.OrphanedBlobs++
			result.BytesFreed += size

			if !dryRun {
				blobFile := filepath.Join(blobsPath, strings.Replace(digest, ":", "-", 1))
				if err := os.Remove(blobFile); err != nil {
					return result, fmt.Errorf("failed to remove blob %s: %w", digest, err)
				}
				result.RemovedBlobs++
			}
		}
	}

	return result, nil
}

// CopyModel copies a model from source to destination models directory.
// It copies the manifest and any blobs that don't already exist at the destination.
// dstModelName can differ from srcModelName to rename during copy.
func CopyModel(srcModelsPath, dstModelsPath, srcModelName, dstModelName string) error {
	// Find the manifest
	manifestPath, err := FindManifest(srcModelsPath, srcModelName)
	if err != nil {
		return fmt.Errorf("model %s not found: %w", srcModelName, err)
	}

	// Read the manifest
	manifest, err := ReadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// Collect all blobs to copy
	blobs := []Layer{manifest.Config}
	blobs = append(blobs, manifest.Layers...)

	// Copy blobs
	srcBlobsPath := filepath.Join(srcModelsPath, "blobs")
	dstBlobsPath := filepath.Join(dstModelsPath, "blobs")
	if err := os.MkdirAll(dstBlobsPath, 0755); err != nil {
		return fmt.Errorf("failed to create blobs directory: %w", err)
	}

	for _, blob := range blobs {
		blobFile := strings.Replace(blob.Digest, ":", "-", 1)
		srcBlob := filepath.Join(srcBlobsPath, blobFile)
		dstBlob := filepath.Join(dstBlobsPath, blobFile)

		// Skip if already exists at destination
		if _, err := os.Stat(dstBlob); err == nil {
			continue
		}

		if err := copyFileVerified(srcBlob, dstBlob, blob.Digest); err != nil {
			return fmt.Errorf("failed to copy blob %s: %w", blob.Digest, err)
		}
	}

	// Determine destination manifest path
	var dstManifest string
	if dstModelName == srcModelName {
		// Same name — mirror the source path
		relManifest, _ := filepath.Rel(filepath.Join(srcModelsPath, "manifests"), manifestPath)
		dstManifest = filepath.Join(dstModelsPath, "manifests", relManifest)
	} else {
		// Different name — build path from destination model name
		dstName, dstTag := parseModelName(dstModelName)
		// Use registry.ollama.ai/library as default namespace
		if !strings.Contains(dstName, "/") {
			dstName = "library/" + dstName
		}
		dstManifest = filepath.Join(dstModelsPath, "manifests", "registry.ollama.ai", dstName, dstTag)
	}

	if err := os.MkdirAll(filepath.Dir(dstManifest), 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}

	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	if err := os.WriteFile(dstManifest, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// RemoveModel removes a model manifest and vacuums unused blobs.
func RemoveModel(modelsPath, modelName string) (*VacuumResult, error) {
	manifestPath, err := FindManifest(modelsPath, modelName)
	if err != nil {
		return nil, fmt.Errorf("model %s not found: %w", modelName, err)
	}

	// Remove the manifest
	if err := os.Remove(manifestPath); err != nil {
		return nil, fmt.Errorf("failed to remove manifest: %w", err)
	}

	// Clean up empty directories
	cleanEmptyDirs(filepath.Dir(manifestPath), filepath.Join(modelsPath, "manifests"))

	// Vacuum unused blobs
	return Vacuum(modelsPath, false)
}

// FindManifest locates the manifest file for a given model name.
// Model name can be: "namespace/model:tag", "model:tag", or "model" (defaults to latest).
func FindManifest(modelsPath string, modelName string) (string, error) {
	manifestsPath := filepath.Join(modelsPath, "manifests")

	// Parse model name
	name, tag := parseModelName(modelName)

	// Search for matching manifest
	var found string
	err := filepath.Walk(manifestsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(manifestsPath, path)
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) < 4 {
			return nil
		}

		fileTag := parts[len(parts)-1]
		fileModel := parts[len(parts)-2]
		fileNamespace := parts[len(parts)-3]

		// Match against the requested name
		fullName := fmt.Sprintf("%s/%s", fileNamespace, fileModel)
		if (fullName == name || fileModel == name) && fileTag == tag {
			found = path
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if found == "" {
		return "", fmt.Errorf("manifest not found for %s", modelName)
	}

	return found, nil
}

// parseModelName splits a model name into name and tag parts.
func parseModelName(modelName string) (string, string) {
	parts := strings.SplitN(modelName, ":", 2)
	name := parts[0]
	tag := "latest"
	if len(parts) == 2 {
		tag = parts[1]
	}
	return name, tag
}

// copyFileVerified copies a file and verifies the sha256 digest.
func copyFileVerified(src, dst, expectedDigest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Write to file and compute hash simultaneously
	hash := sha256.New()
	writer := io.MultiWriter(dstFile, hash)

	if _, err := io.Copy(writer, srcFile); err != nil {
		os.Remove(dst)
		return err
	}

	// Verify integrity
	computed := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if computed != expectedDigest {
		os.Remove(dst)
		return fmt.Errorf("integrity check failed: expected %s, got %s", expectedDigest, computed)
	}

	return nil
}

// cleanEmptyDirs removes empty parent directories up to the stop path.
func cleanEmptyDirs(dir, stopAt string) {
	for dir != stopAt {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}

// FormatSize returns a human-readable size string.
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
