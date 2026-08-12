package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/paularlott/cli"
	"github.com/paularlott/ollama-vault/gguf"
	"github.com/paularlott/ollama-vault/vault"
)

func init() {
	infoCmd := &cli.Command{
		Name:  "info",
		Usage: "Show information about a model",
		Description: `Display metadata from a GGUF model file or an ollama/vault model.

Examples:
  ollama-vault info ollama/llama3:latest
  ollama-vault info vault/mistral:7b
  ollama-vault info ./model.gguf
  ollama-vault info --template ollama/llama3:latest
  ollama-vault info --json ollama/llama3:latest`,
		Arguments: []cli.Argument{
			&cli.StringArg{Name: "model", Usage: "Model reference or path to .gguf file", Required: true},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "template",
				Aliases: []string{"t"},
				Usage:   "Show the chat template",
			},
			&cli.BoolFlag{
				Name:  "all",
				Usage: "Show all metadata keys",
			},
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "Output as JSON",
			},
		},
		Run: runInfo,
	}

	Commands = append(Commands, infoCmd)
}

func runInfo(ctx context.Context, cmd *cli.Command) error {
	modelArg := cmd.GetStringArg("model")
	showTemplate := cmd.GetBool("template")
	showAll := cmd.GetBool("all")
	outputJSON := cmd.GetBool("json")

	// Determine if this is a file path or a model reference
	ggufPath, err := resolveGGUFPath(cmd, modelArg)
	if err != nil {
		return err
	}

	info, err := gguf.ReadInfo(ggufPath)
	if err != nil {
		return fmt.Errorf("failed to read GGUF: %w", err)
	}

	if outputJSON {
		return printJSON(info, showAll)
	}

	if showAll {
		printAllMetadata(info)
		return nil
	}

	printModelInfo(info)

	if showTemplate {
		fmt.Println()
		template := info.Metadata.GetString("tokenizer.chat_template")
		if template != "" {
			fmt.Println("Chat template:")
			fmt.Println(template)
		} else {
			fmt.Println("Chat template: not present in GGUF")
		}
	}

	return nil
}

func printJSON(info *gguf.Info, all bool) error {
	output := make(map[string]interface{})

	output["gguf_version"] = info.Header.Version
	output["tensor_count"] = info.Header.TensorCount
	output["metadata_count"] = info.Header.MetadataCount

	if all {
		// Include all metadata, but skip large arrays
		metadata := make(map[string]interface{})
		for key, value := range info.Metadata {
			switch v := value.(type) {
			case []interface{}:
				metadata[key] = fmt.Sprintf("[array, %d elements]", len(v))
			default:
				metadata[key] = v
			}
		}
		output["metadata"] = metadata
	} else {
		m := info.Metadata
		arch := m.GetString("general.architecture")

		output["architecture"] = arch
		if name := m.GetString("general.name"); name != "" {
			output["name"] = name
		}
		if ft := m.GetUint32("general.file_type"); ft != 0 {
			output["quantization"] = gguf.FileTypeString(ft)
		}
		if version := m.GetString("general.version"); version != "" {
			output["version"] = version
		}
		if author := m.GetString("general.author"); author != "" {
			output["author"] = author
		}
		if license := m.GetString("general.license"); license != "" {
			output["license"] = license
		}
		if tokModel := m.GetString("tokenizer.ggml.model"); tokModel != "" {
			output["tokenizer"] = tokModel
		}

		output["has_chat_template"] = m.Has("tokenizer.chat_template")
		if m.Has("tokenizer.chat_template") {
			output["chat_template"] = m.GetString("tokenizer.chat_template")
		}

		if arch != "" {
			if ctx := m.GetUint64(arch + ".context_length"); ctx != 0 {
				output["context_length"] = ctx
			}
			if embd := m.GetUint64(arch + ".embedding_length"); embd != 0 {
				output["embedding_length"] = embd
			}
			if blocks := m.GetUint64(arch + ".block_count"); blocks != 0 {
				output["block_count"] = blocks
			}
			if heads := m.GetUint64(arch + ".attention.head_count"); heads != 0 {
				output["attention_heads"] = heads
			}
			if kvHeads := m.GetUint64(arch + ".attention.head_count_kv"); kvHeads != 0 {
				output["kv_heads"] = kvHeads
			}
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func resolveGGUFPath(cmd *cli.Command, modelArg string) (string, error) {
	// Check if it's a direct file path
	if strings.HasSuffix(modelArg, ".gguf") || fileExists(modelArg) {
		if !fileExists(modelArg) {
			return "", fmt.Errorf("file not found: %s", modelArg)
		}
		return modelArg, nil
	}

	// Parse as a model reference
	ref, err := ParseRef(modelArg)
	if err != nil {
		return "", err
	}

	if ref.Model == "" {
		return "", fmt.Errorf("must specify a model name (e.g. ollama/llama3:latest)")
	}

	modelsPath, err := ResolvePath(cmd, ref.Prefix)
	if err != nil {
		return "", err
	}

	return vault.FindModelBlob(modelsPath, ref.Model)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func printModelInfo(info *gguf.Info) {
	m := info.Metadata

	fmt.Printf("GGUF version:    %d\n", info.Header.Version)
	fmt.Printf("Tensors:         %d\n", info.Header.TensorCount)
	fmt.Printf("Metadata keys:   %d\n", info.Header.MetadataCount)
	fmt.Println()

	// General info
	if name := m.GetString("general.name"); name != "" {
		fmt.Printf("Name:            %s\n", name)
	}
	if arch := m.GetString("general.architecture"); arch != "" {
		fmt.Printf("Architecture:    %s\n", arch)
	}
	if ft := m.GetUint32("general.file_type"); ft != 0 {
		fmt.Printf("Quantization:    %s\n", gguf.FileTypeString(ft))
	}
	if version := m.GetString("general.version"); version != "" {
		fmt.Printf("Version:         %s\n", version)
	}
	if author := m.GetString("general.author"); author != "" {
		fmt.Printf("Author:          %s\n", author)
	}
	if desc := m.GetString("general.description"); desc != "" {
		fmt.Printf("Description:     %s\n", desc)
	}
	if license := m.GetString("general.license"); license != "" {
		fmt.Printf("License:         %s\n", license)
	}

	// Architecture-specific info
	arch := m.GetString("general.architecture")
	if arch != "" {
		fmt.Println()
		if ctx := m.GetUint64(arch + ".context_length"); ctx != 0 {
			fmt.Printf("Context length:  %d\n", ctx)
		}
		if embd := m.GetUint64(arch + ".embedding_length"); embd != 0 {
			fmt.Printf("Embedding size:  %d\n", embd)
		}
		if blocks := m.GetUint64(arch + ".block_count"); blocks != 0 {
			fmt.Printf("Layers:          %d\n", blocks)
		}
		if heads := m.GetUint64(arch + ".attention.head_count"); heads != 0 {
			fmt.Printf("Attention heads: %d\n", heads)
		}
		if kvHeads := m.GetUint64(arch + ".attention.head_count_kv"); kvHeads != 0 {
			fmt.Printf("KV heads:        %d\n", kvHeads)
		}
	}

	// Template info
	fmt.Println()
	if m.Has("tokenizer.chat_template") {
		fmt.Println("Chat template:   present")
	} else {
		fmt.Println("Chat template:   not present")
	}

	if tokModel := m.GetString("tokenizer.ggml.model"); tokModel != "" {
		fmt.Printf("Tokenizer:       %s\n", tokModel)
	}
}

func printAllMetadata(info *gguf.Info) {
	fmt.Printf("GGUF version: %d | Tensors: %d | Metadata keys: %d\n\n",
		info.Header.Version, info.Header.TensorCount, info.Header.MetadataCount)

	for key, value := range info.Metadata {
		switch v := value.(type) {
		case string:
			if len(v) > 200 {
				fmt.Printf("%-45s = %s... (%d bytes)\n", key, v[:200], len(v))
			} else {
				fmt.Printf("%-45s = %s\n", key, v)
			}
		case []interface{}:
			fmt.Printf("%-45s = [array, %d elements]\n", key, len(v))
		default:
			fmt.Printf("%-45s = %v\n", key, v)
		}
	}
}
