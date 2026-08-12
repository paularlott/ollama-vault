package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/paularlott/cli"
	cli_toml "github.com/paularlott/cli/toml"
	"github.com/paularlott/ollama-vault/build"
	"github.com/paularlott/ollama-vault/cmd"
)

func main() {
	configFileName := "ollama-vault.toml"

	app := &cli.Command{
		Name:    "ollama-vault",
		Version: build.Version,
		Usage:   "Manage ollama models with a vault for backup and sharing",
		ConfigFile: cli_toml.NewConfigFile(&configFileName, func() []string {
			home, _ := os.UserHomeDir()
			return []string{
				".",
				filepath.Join(home, ".config", "ollama-vault"),
				home,
			}
		}),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to config file",
				EnvVars: []string{"OLLAMA_VAULT_CONFIG"},
				Global:  true,
				Hidden:  true,
			},
			&cli.StringFlag{
				Name:       "vault",
				Aliases:    []string{"V"},
				Usage:      "Path to the vault",
				EnvVars:    []string{"OLLAMA_VAULT_PATH"},
				ConfigPath: []string{"vault.path"},
				Global:     true,
			},
			&cli.StringFlag{
				Name:       "ollama",
				Aliases:    []string{"o"},
				Usage:      "Path to ollama models directory",
				EnvVars:    []string{"OLLAMA_MODELS"},
				ConfigPath: []string{"ollama.models_path"},
				Global:     true,
			},
		},
		Commands: cmd.Commands,
		PreRun: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			if cfgPath := c.GetString("config"); cfgPath != "" {
				configFileName = cfgPath
				c.ReloadFlags()
			}
			return ctx, nil
		},
	}

	if err := app.Execute(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
