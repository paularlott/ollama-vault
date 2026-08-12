# ollama-vault

A CLI tool to manage ollama models with a vault — back them up, share them between machines, and deduplicate storage.

The vault stores models in the same blob format as ollama, so you can point `OLLAMA_MODELS` at your vault and use models without copying.

## Install

### Homebrew

```sh
brew install paularlott/tap/ollama-vault
```

### From source

```sh
task build
# Binary at dist/ollama-vault
```

## Configuration

ollama-vault needs to know where your vault is. Set it up with a config file, environment variable, or pass it as a flag.

Create `~/.config/ollama-vault/ollama-vault.toml`:

```toml
[vault]
path = "/mnt/shared/ollama-vault"

[ollama]
models_path = "/Users/you/.ollama/models"
```

The `ollama.models_path` is optional — it defaults to the standard ollama location for your OS.

Config files are also searched in the current directory and home directory.

Alternatively, use environment variables or flags directly:

| Flag | Environment Variable | Config Path |
|------|---------------------|-------------|
| `--vault`, `-V` | `OLLAMA_VAULT_PATH` | `vault.path` |
| `--ollama`, `-o` | `OLLAMA_MODELS` | `ollama.models_path` |

Priority: flags > environment variables > config file.

## Quick start

```sh
# Set your vault path (or use the config file above)
export OLLAMA_VAULT_PATH=~/ollama-vault

# Copy models from ollama into the vault
ollama-vault cp ollama/llama3:latest vault/
ollama-vault cp ollama/codellama:13b vault/

# On another machine, copy from the vault to ollama
ollama-vault cp vault/llama3:latest ollama/

# Copy and rename
ollama-vault cp ollama/llama3:latest vault/my-llama:latest

# See what's where
ollama-vault diff
```

## Commands

Use `ollama/` and `vault/` as virtual prefixes to specify where models live. When no prefix is given, `vault/` is assumed.

Commands have short aliases: `copy`/`cp`, `list`/`ls`, `remove`/`rm`.

### copy (cp)

Copy a model between ollama and vault. Blobs are verified with sha256 during copy. Existing blobs at the destination are skipped (dedup).

```sh
ollama-vault cp ollama/llama3:latest vault/
ollama-vault cp vault/mistral:7b ollama/
ollama-vault cp ollama/llama3:latest vault/my-llama:latest
```

### list (ls)

List models in a location. Defaults to `vault/`.

```sh
ollama-vault ls
ollama-vault ls vault/
ollama-vault ls ollama/
```

### diff

Show which models exist in ollama only, vault only, or both.

```sh
ollama-vault diff
```

### remove (rm)

Remove a model and clean up any blobs that are no longer referenced.

```sh
ollama-vault rm vault/codellama:13b
ollama-vault rm ollama/llama3:latest
```

### vacuum

Scan for orphaned blobs and remove them. Reports model count, blob count, and dedup ratio.

```sh
ollama-vault vacuum
ollama-vault vacuum vault/
ollama-vault vacuum ollama/
ollama-vault vacuum --dry-run
```

### export

Export the GGUF weights file from a model. The exported file is a standard GGUF that works with llama.cpp, vLLM, LM Studio, and any other GGUF-compatible tool.

```sh
ollama-vault export ollama/llama3:latest ./llama3.gguf
ollama-vault export vault/mistral:7b /tmp/mistral.gguf
```

### info

Show model metadata — architecture, quantization, context length, etc. Works with ollama/vault models or any `.gguf` file on disk.

```sh
ollama-vault info ollama/llama3:latest
ollama-vault info vault/mistral:7b
ollama-vault info ./model.gguf

# Show the Jinja2 chat template
ollama-vault info --template ollama/llama3:latest

# Show all metadata keys
ollama-vault info --all ./model.gguf

# Output as JSON
ollama-vault info --json ollama/llama3:latest
```

## How it works

Ollama stores models as:
- **Manifests** — JSON files that reference blob digests, stored at `manifests/<registry>/<namespace>/<model>/<tag>`
- **Blobs** — Content-addressed files named `sha256-<hash>` in the `blobs/` directory

ollama-vault uses this same layout. When multiple models share layers (which is common), those blobs are stored once. The vault naturally deduplicates.

During copy operations, each blob is verified against its sha256 digest to ensure integrity.

## License

MIT License — Copyright (c) 2026 Paul Arlott

## Disclaimer

This is a personal tool. It works for me, but it might eat your models, corrupt your blobs, or do something else unexpected. Back up anything you care about. Use at your own risk.
