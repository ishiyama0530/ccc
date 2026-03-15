# ccc

[English](./README.md) | [日本語](./README.ja.md)

`ccc` is a small CLI for finding a Claude Code session and resuming it quickly.

## Install

Homebrew:

```bash
brew tap owner/homebrew-tap
brew install ccc
```

## Use

```bash
ccc <query>
ccc -d <dir> <query>
ccc --dir <dir> <query>
```

- By default, `ccc` searches the Claude history for the current working directory.
- `-d` / `--dir` switches the target working directory.
- Search is case-insensitive.
- If matches are found, `ccc` opens the TUI.
- Extra Claude args are typed into the bottom command bar. `ccc` always keeps `claude --resume <session_id>` fixed and appends your args after it.
- If matches are found without a TTY, `ccc` exits with an error.
- If nothing matches, `ccc` prints an error to stderr and exits non-zero.

Examples:

```bash
ccc bug
ccc -d ~/src/app timeout
```

## Keys

- `↑` / `↓`: move
- `Enter`: resume the selected session
- Type text: add extra Claude args
- `Backspace`: edit extra args
- `PgUp` / `PgDn`: scroll preview
- `Ctrl+U` / `Ctrl+D`: scroll preview faster
- `esc` / `ctrl+c`: quit

## Develop

```bash
make build
make test
make lint
make run QUERY="bug"
# run the local bin/ccc without installing it
PATH="$PWD/bin:$PATH" ccc bug
```

## Release

```bash
export GITHUB_TOKEN=...
make release VERSION=vX.Y.Z TAP_REPO=owner/homebrew-tap
```
