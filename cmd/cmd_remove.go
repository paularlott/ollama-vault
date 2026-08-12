package cmd

import (
	"context"
	"fmt"

	"github.com/paularlott/cli"
	"github.com/paularlott/ollama-vault/vault"
)

func init() {
	removeCmd := &cli.Command{
		Name:    "remove",
		Aliases: []string{"rm"},
		Usage:   "Remove a model and vacuum unused blobs",
		Description: `Remove a model and clean up orphaned blobs. Defaults to vault/ if no prefix.

Examples:
  ollama-vault remove vault/llama3:latest
  ollama-vault remove ollama/llama3:latest
  ollama-vault remove llama3:latest`,
		Arguments: []cli.Argument{
			&cli.StringArg{Name: "model", Usage: "Model to remove (e.g. vault/llama3:latest)", Required: true},
		},
		Run: runRemove,
	}

	Commands = append(Commands, removeCmd)
}

func runRemove(ctx context.Context, cmd *cli.Command) error {
	ref, err := ParseRef(cmd.GetStringArg("model"))
	if err != nil {
		return err
	}

	if ref.Model == "" {
		return fmt.Errorf("must specify a model name (e.g. vault/llama3:latest)")
	}

	targetPath, err := ResolvePath(cmd, ref.Prefix)
	if err != nil {
		return err
	}

	fmt.Printf("Removing %s/%s...\n", ref.Prefix, ref.Model)

	result, err := vault.RemoveModel(targetPath, ref.Model)
	if err != nil {
		return err
	}

	fmt.Printf("Model removed. Cleaned %d orphaned blob(s), freed %s.\n",
		result.RemovedBlobs, vault.FormatSize(result.BytesFreed))

	return nil
}
