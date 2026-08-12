package cmd

import (
	"context"
	"fmt"

	"github.com/paularlott/cli"
	"github.com/paularlott/ollama-vault/gguf"
	"github.com/paularlott/ollama-vault/vault"
)

func init() {
	exportCmd := &cli.Command{
		Name:  "export",
		Usage: "Export a model as a GGUF file",
		Description: `Export the GGUF weights from a model in ollama or the vault.

Examples:
  ollama-vault export ollama/llama3:latest ./llama3.gguf
  ollama-vault export vault/mistral:7b /tmp/mistral.gguf`,
		Arguments: []cli.Argument{
			&cli.StringArg{Name: "model", Usage: "Model to export (e.g. ollama/llama3:latest)", Required: true},
			&cli.StringArg{Name: "output", Usage: "Output path for the GGUF file", Required: true},
		},
		Run: runExport,
	}

	Commands = append(Commands, exportCmd)
}

func runExport(ctx context.Context, cmd *cli.Command) error {
	ref, err := ParseRef(cmd.GetStringArg("model"))
	if err != nil {
		return err
	}

	if ref.Model == "" {
		return fmt.Errorf("must specify a model name (e.g. ollama/llama3:latest)")
	}

	outputPath := cmd.GetStringArg("output")

	modelsPath, err := ResolvePath(cmd, ref.Prefix)
	if err != nil {
		return err
	}

	fmt.Printf("Exporting %s/%s...\n", ref.Prefix, ref.Model)

	if err := vault.ExportGGUF(modelsPath, ref.Model, outputPath); err != nil {
		return err
	}

	// Verify the exported file is valid GGUF and report info
	info, err := gguf.ReadInfo(outputPath)
	if err != nil {
		fmt.Println("Warning: exported file may not be a valid GGUF")
		return nil
	}

	arch := info.Metadata.GetString("general.architecture")
	name := info.Metadata.GetString("general.name")
	if name != "" {
		fmt.Printf("Model: %s\n", name)
	}
	if arch != "" {
		fmt.Printf("Architecture: %s\n", arch)
	}
	fmt.Printf("Tensors: %d\n", info.Header.TensorCount)

	if info.Metadata.Has("tokenizer.chat_template") {
		fmt.Println("Chat template: present")
	} else {
		fmt.Println("Chat template: not present (may need to be set manually)")
	}

	return nil
}
