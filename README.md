# ccc

[English](./README.md) | [日本語](./README.ja.md)

`ccc` is a small CLI for finding a Claude Code session and resuming it quickly.

## Install

Install with:

```bash
curl -fsSL https://raw.githubusercontent.com/ishiyama0530/ccc/main/install.sh | bash
```

- Downloads the latest GitHub Release
- Supports macOS / Linux on `amd64` / `arm64`
- Installs `ccc` into `~/.local/bin` by default

If `~/.local/bin` is not on your PATH, add it in your shell config.

To change the install location:

```bash
curl -fsSL https://raw.githubusercontent.com/ishiyama0530/ccc/main/install.sh | env CCC_INSTALL_DIR="$HOME/bin" bash
```

To pin a version:

```bash
curl -fsSL https://raw.githubusercontent.com/ishiyama0530/ccc/main/install.sh | env CCC_INSTALL_VERSION=vX.Y.Z bash
```

## Use

```bash
ccc
ccc -d <dir>
ccc -n <count>
ccc <query>
ccc -d <dir> <query>
ccc -n <count> <query>
ccc --dir <dir> <query>
```

- By default, `ccc` searches the Claude history for the current working directory.
- `-d` / `--dir` switches the target working directory.
- `-n` / `--limit` sets the maximum number of history entries to display. The default is `100`.
- With no query, `ccc` lists up to `100` session history entries for the target directory by default.
- Search is case-insensitive.
- If matches are found, `ccc` opens the TUI.
- Extra Claude args are typed into the bottom command bar. `ccc` always keeps `claude --resume <session_id>` fixed and appends your args after it.
- If matches are found without a TTY, `ccc` exits with an error.
- If nothing matches, `ccc` prints an error to stderr and exits non-zero.

Examples:

```bash
ccc
ccc -n 200
ccc bug
ccc -d ~/src/app
ccc -d ~/src/app -n 50 timeout
ccc -d ~/src/app timeout
```

## Keys

- `↑` / `↓`: move
- `Shift+↑` / `Shift+↓`: scroll preview line by line
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

GitHub Actions now publishes releases automatically when commits are pushed or merged into `main`. The workflow finds the latest `vMAJOR.MINOR.PATCH` tag and publishes the next patch version.

You can also run the `Release` workflow manually from GitHub Actions and optionally pass a `version` input such as `v1.2.3`. If you leave the input blank, it will auto-increment the latest patch version.

For local releases:

```bash
export GITHUB_TOKEN=...
make release VERSION=vX.Y.Z
```

`make release` publishes the GitHub Release assets used by `install.sh`.
