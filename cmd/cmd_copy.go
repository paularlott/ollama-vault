package cmd

import (
	"context"
	"fmt"

	"github.com/paularlott/cli"
	"github.com/paularlott/ollama-vault/vault"
)

func init() {
	copyCmd := &cli.Command{
		Name:    "copy",
		Aliases: []string{"cp"},
		Usage:   "Copy models between ollama and vault",
		Description: `Copy a model from one location to another. Use ollama/ and vault/ prefixes
to specify source and destination.

Examples:
  ollama-vault copy ollama/llama3:latest vault/
  ollama-vault copy vault/llama3:latest ollama/
  ollama-vault copy ollama/llama3:latest vault/my-llama:latest`,
		Arguments: []cli.Argument{
			&cli.StringArg{Name: "source", Usage: "Source reference (e.g. ollama/llama3:latest)", Required: true},
			&cli.StringArg{Name: "destination", Usage: "Destination reference (e.g. vault/)", Required: true},
		},
		Run: runCopy,
	}

	Commands = append(Commands, copyCmd)
}

func runCopy(ctx context.Context, cmd *cli.Command) error {
	src, err := ParseRef(cmd.GetStringArg("source"))
	if err != nil {
		return err
	}

	dst, err := ParseRef(cmd.GetStringArg("destination"))
	if err != nil {
		return err
	}

	if src.Model == "" {
		return fmt.Errorf("source must include a model name (e.g. ollama/llama3:latest)")
	}

	if src.Prefix == dst.Prefix {
		return fmt.Errorf("source and destination must be different (both are %s)", src.Prefix)
	}

	srcPath, err := ResolvePath(cmd, src.Prefix)
	if err != nil {
		return err
	}

	dstPath, err := ResolvePath(cmd, dst.Prefix)
	if err != nil {
		return err
	}

	// If destination has no model name, use the source model name
	dstModel := dst.Model
	if dstModel == "" {
		dstModel = src.Model
	}

	fmt.Printf("Copying %s/%s -> %s/%s\n", src.Prefix, src.Model, dst.Prefix, dstModel)

	if err := vault.CopyModel(srcPath, dstPath, src.Model, dstModel); err != nil {
		return err
	}

	fmt.Println("Done.")
	return nil
}
