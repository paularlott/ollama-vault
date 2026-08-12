package cmd

import (
	"fmt"
	"strings"

	"github.com/paularlott/cli"
	"github.com/paularlott/ollama-vault/vault"
)

const (
	PrefixOllama = "ollama"
	PrefixVault  = "vault"
)

// ParsedRef represents a parsed reference like "ollama/llama3:latest" or "vault/"
type ParsedRef struct {
	Prefix string // "ollama" or "vault"
	Model  string // model name including tag, e.g. "llama3:latest", or empty
}

// ParseRef parses a string like "ollama/llama3:latest" or "vault/" into prefix and model.
// If no prefix is given, defaults to "vault".
func ParseRef(ref string) (ParsedRef, error) {
	if ref == "" {
		return ParsedRef{Prefix: PrefixVault}, nil
	}

	// Split on first /
	idx := strings.Index(ref, "/")
	if idx == -1 {
		// No slash — could be just a prefix like "ollama" or "vault", or a model name
		if ref == PrefixOllama || ref == PrefixVault {
			return ParsedRef{Prefix: ref}, nil
		}
		// Treat as a vault model reference
		return ParsedRef{Prefix: PrefixVault, Model: ref}, nil
	}

	prefix := ref[:idx]
	model := ref[idx+1:]

	if prefix != PrefixOllama && prefix != PrefixVault {
		return ParsedRef{}, fmt.Errorf("unknown prefix %q, expected 'ollama' or 'vault'", prefix)
	}

	return ParsedRef{Prefix: prefix, Model: model}, nil
}

// ResolvePath returns the filesystem path for the given prefix using flags/config.
func ResolvePath(cmd *cli.Command, prefix string) (string, error) {
	switch prefix {
	case PrefixVault:
		path := cmd.GetString("vault")
		if path == "" {
			return "", fmt.Errorf("vault path not set, use --vault, OLLAMA_VAULT_PATH env var, or config file")
		}
		return path, nil
	case PrefixOllama:
		path := cmd.GetString("ollama")
		if path == "" {
			path = vault.DefaultOllamaModelsPath()
		}
		return path, nil
	default:
		return "", fmt.Errorf("unknown prefix: %s", prefix)
	}
}
