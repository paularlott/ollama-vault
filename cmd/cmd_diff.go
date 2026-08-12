package cmd

import (
	"context"
	"fmt"

	"github.com/paularlott/cli"
	"github.com/paularlott/ollama-vault/vault"
)

func init() {
	diffCmd := &cli.Command{
		Name:  "diff",
		Usage: "Show differences between ollama and vault",
		Run:   runDiff,
	}

	Commands = append(Commands, diffCmd)
}

func runDiff(ctx context.Context, cmd *cli.Command) error {
	vaultPath, err := ResolvePath(cmd, PrefixVault)
	if err != nil {
		return err
	}

	ollamaPath, err := ResolvePath(cmd, PrefixOllama)
	if err != nil {
		return err
	}

	vaultModels, err := vault.ListModels(vaultPath)
	if err != nil {
		return fmt.Errorf("failed to list vault models: %w", err)
	}

	ollamaModels, err := vault.ListModels(ollamaPath)
	if err != nil {
		return fmt.Errorf("failed to list ollama models: %w", err)
	}

	// Build lookup maps
	vaultSet := make(map[string]vault.ModelInfo)
	for _, m := range vaultModels {
		vaultSet[m.Name] = m
	}

	ollamaSet := make(map[string]vault.ModelInfo)
	for _, m := range ollamaModels {
		ollamaSet[m.Name] = m
	}

	// Categorize
	var onlyOllama, onlyVault, both []string

	for name := range ollamaSet {
		if _, exists := vaultSet[name]; exists {
			both = append(both, name)
		} else {
			onlyOllama = append(onlyOllama, name)
		}
	}

	for name := range vaultSet {
		if _, exists := ollamaSet[name]; !exists {
			onlyVault = append(onlyVault, name)
		}
	}

	if len(both) > 0 {
		fmt.Println("In both ollama and vault:")
		for _, name := range both {
			fmt.Printf("  %s\n", name)
		}
		fmt.Println()
	}

	if len(onlyOllama) > 0 {
		fmt.Println("Only in ollama (copy to vault with: copy ollama/<model> vault/):")
		for _, name := range onlyOllama {
			fmt.Printf("  %s\n", name)
		}
		fmt.Println()
	}

	if len(onlyVault) > 0 {
		fmt.Println("Only in vault (copy to ollama with: copy vault/<model> ollama/):")
		for _, name := range onlyVault {
			fmt.Printf("  %s\n", name)
		}
		fmt.Println()
	}

	if len(onlyOllama) == 0 && len(onlyVault) == 0 {
		fmt.Println("Ollama and vault are in sync.")
	}

	return nil
}
