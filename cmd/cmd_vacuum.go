package cmd

import (
	"context"
	"fmt"

	"github.com/paularlott/cli"
	"github.com/paularlott/ollama-vault/vault"
)

func init() {
	vacuumCmd := &cli.Command{
		Name:  "vacuum",
		Usage: "Find and remove orphaned blobs",
		Description: `Scan for orphaned blobs and remove them. Defaults to vault/ if no prefix.

Reports model count, blob count, and dedup ratio.

Examples:
  ollama-vault vacuum
  ollama-vault vacuum vault/
  ollama-vault vacuum ollama/
  ollama-vault vacuum --dry-run`,
		Arguments: []cli.Argument{
			&cli.StringArg{Name: "target", Usage: "Target to vacuum (e.g. vault/, ollama/)"},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"n"},
				Usage:   "Show what would be removed without deleting",
			},
		},
		Run: runVacuum,
	}

	Commands = append(Commands, vacuumCmd)
}

func runVacuum(ctx context.Context, cmd *cli.Command) error {
	ref, err := ParseRef(cmd.GetStringArg("target"))
	if err != nil {
		return err
	}

	dryRun := cmd.GetBool("dry-run")

	targetPath, err := ResolvePath(cmd, ref.Prefix)
	if err != nil {
		return err
	}

	fmt.Printf("Scanning %s at: %s\n", ref.Prefix, targetPath)

	if dryRun {
		fmt.Println("(dry run - no files will be deleted)")
	}

	result, err := vault.Vacuum(targetPath, dryRun)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Models:          %d\n", result.TotalModels)
	fmt.Printf("Total blobs:     %d\n", result.TotalBlobs)
	fmt.Printf("Unique blobs:    %d\n", result.UniqueBlobs)
	fmt.Printf("Dedup ratio:     %.2fx\n", result.DedupRatio)
	fmt.Printf("Orphaned blobs:  %d\n", result.OrphanedBlobs)

	if result.OrphanedBlobs > 0 {
		if dryRun {
			fmt.Printf("Would free:      %s\n", vault.FormatSize(result.BytesFreed))
		} else {
			fmt.Printf("Removed:         %d blob(s)\n", result.RemovedBlobs)
			fmt.Printf("Freed:           %s\n", vault.FormatSize(result.BytesFreed))
		}
	} else {
		fmt.Println("No orphaned blobs found.")
	}

	return nil
}
