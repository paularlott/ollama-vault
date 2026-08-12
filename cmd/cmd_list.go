package cmd

import (
	"context"
	"fmt"

	"github.com/paularlott/cli"
	"github.com/paularlott/ollama-vault/vault"
)

func init() {
	listCmd := &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List models",
		Description: `List models in a location. Defaults to vault/ if no prefix given.

Examples:
  ollama-vault list
  ollama-vault list vault/
  ollama-vault list ollama/`,
		Arguments: []cli.Argument{
			&cli.StringArg{Name: "target", Usage: "Target to list (e.g. vault/, ollama/)"},
		},
		Run: runList,
	}

	Commands = append(Commands, listCmd)
}

func runList(ctx context.Context, cmd *cli.Command) error {
	ref, err := ParseRef(cmd.GetStringArg("target"))
	if err != nil {
		return err
	}

	targetPath, err := ResolvePath(cmd, ref.Prefix)
	if err != nil {
		return err
	}

	models, err := vault.ListModels(targetPath)
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	if len(models) == 0 {
		fmt.Printf("No models in %s.\n", ref.Prefix)
		return nil
	}

	fmt.Printf("%-40s %10s\n", "MODEL", "SIZE")
	for _, model := range models {
		fmt.Printf("%-40s %10s\n", model.Name, vault.FormatSize(model.Size))
	}

	return nil
}
