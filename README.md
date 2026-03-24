# Claude Code Continue (claudecc)

[English](./README.md) | [日本語](./README.ja.md)

`claudecc` is a small CLI for finding a Claude Code session and resuming it quickly.

## Install

Install with:

```bash
curl -fsSL https://raw.githubusercontent.com/ishiyama0530/ccc/main/install.sh | bash
```

- Downloads the latest GitHub Release
- Supports macOS / Linux on `amd64` / `arm64`
- Installs `claudecc` into `~/.local/bin` by default

Install with npm instead:

```bash
npm install -g claudecc
# or
npx claudecc
```

- Downloads the matching GitHub Release asset during `npm install`
- Supports macOS / Linux / Windows on `amd64` / `arm64`
- Installs `claudecc`

If `~/.local/bin` is not on your PATH, add it in your shell config.

To change the install location:

```bash
curl -fsSL https://raw.githubusercontent.com/ishiyama0530/ccc/main/install.sh | env CLAUDECC_INSTALL_DIR="$HOME/bin" bash
```

To pin a version:

```bash
curl -fsSL https://raw.githubusercontent.com/ishiyama0530/ccc/main/install.sh | env CLAUDECC_INSTALL_VERSION=vX.Y.Z bash
```

## Use

```bash
claudecc
claudecc -d <dir>
claudecc -n <count>
claudecc <query>
claudecc -d <dir> <query>
claudecc -n <count> <query>
claudecc --dir <dir> <query>
```

- By default, `claudecc` searches the Claude history for the current working directory.
- `-d` / `--dir` switches the target working directory.
- `-n` / `--limit` sets the maximum number of history entries to display. The default is `100`.
- With no query, `claudecc` lists up to `100` session history entries for the target directory by default.
- Search is case-insensitive.
- If matches are found, `claudecc` opens the TUI.
- Extra Claude args are typed into the bottom command bar. `claudecc` always keeps `claude --resume <session_id>` fixed and appends your args after it.
- If matches are found without a TTY, `claudecc` exits with an error.
- If nothing matches, `claudecc` prints an error to stderr and exits non-zero.

Examples:

```bash
claudecc
claudecc -n 200
claudecc bug
claudecc -d ~/src/app
claudecc -d ~/src/app -n 50 timeout
claudecc -d ~/src/app timeout
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
# run the local bin/claudecc without installing it
PATH="$PWD/bin:$PATH" claudecc bug
```

## Release

```bash
export GITHUB_TOKEN=...
export NPM_TOKEN=...
make release VERSION=vX.Y.Z
```

`make release` publishes the GitHub Release assets used by `install.sh`, then publishes `claudecc` to npm.
