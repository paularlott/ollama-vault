package vault

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	MediaTypeModel = "application/vnd.ollama.image.model"
)

// FindModelBlob locates the GGUF weights blob for a given model.
// Returns the path to the blob file.
func FindModelBlob(modelsPath, modelName string) (string, error) {
	manifestPath, err := FindManifest(modelsPath, modelName)
	if err != nil {
		return "", err
	}

	manifest, err := ReadManifest(manifestPath)
	if err != nil {
		return "", fmt.Errorf("failed to read manifest: %w", err)
	}

	// Find the model layer (GGUF weights)
	for _, layer := range manifest.Layers {
		if layer.MediaType == MediaTypeModel {
			blobFile := strings.Replace(layer.Digest, ":", "-", 1)
			blobPath := filepath.Join(modelsPath, "blobs", blobFile)

			if _, err := os.Stat(blobPath); err != nil {
				return "", fmt.Errorf("model blob not found: %s", blobPath)
			}

			return blobPath, nil
		}
	}

	return "", fmt.Errorf("no model layer found in manifest for %s", modelName)
}

// ExportGGUF copies the model GGUF blob to the given output path.
func ExportGGUF(modelsPath, modelName, outputPath string) error {
	blobPath, err := FindModelBlob(modelsPath, modelName)
	if err != nil {
		return err
	}

	src, err := os.Open(blobPath)
	if err != nil {
		return fmt.Errorf("failed to open model blob: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer dst.Close()

	n, err := io.Copy(dst, src)
	if err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("failed to write GGUF: %w", err)
	}

	fmt.Printf("Exported %s (%s)\n", outputPath, FormatSize(n))
	return nil
}
