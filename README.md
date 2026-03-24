# ✨ # Claude Code Continue (ccc)

[English](./README.md) | [日本語](./README.ja.md)

> Find the right Claude Code session fast, preview it instantly, and jump back in without digging through transcript files.

<p align="center">
  <img src="./docs/images/ccc-tui.png" alt="ccc TUI session picker with transcript preview and resume command" width="1100" />
</p>

`ccc` is a small, project-aware CLI for Claude Code history. Run it in the repo you are working on, see the sessions that actually matter, and resume the right one in seconds.

## 😩 Why ccc exists

Claude Code session history gets big fast.

When you are trying to get back to "that conversation from earlier," the annoying part is not the resume command. It is finding the right session first.

`ccc` fixes that. It searches the Claude history that belongs to your current working directory, shows a clean preview, and gives you a lightweight picker so you can get back to work with less friction.

## 🚀 Quick start

Recommended for everyday use:

```bash
npm install -g @ishiyama0530/ccc
ccc
```

> 💡 If you think you will use this more than once, the global npm install is the best experience. Install it once, then launch it with `ccc` whenever you need it.

Want to try it without installing globally?

```bash
npx @ishiyama0530/ccc
```

Prefer the GitHub Release installer?

```bash
curl -fsSL https://raw.githubusercontent.com/ishiyama0530/ccc/main/install.sh | bash
```

- `npm install -g @ishiyama0530/ccc`: recommended, supports macOS / Linux / Windows on `amd64` / `arm64`
- `npx @ishiyama0530/ccc`: great for a quick trial, same platform coverage as npm install
- `curl -fsSL ... | bash`: supports macOS / Linux on `amd64` / `arm64`, installs `ccc` into `~/.local/bin` by default

Need a custom install path or a pinned version for the shell installer?

```bash
curl -fsSL https://raw.githubusercontent.com/ishiyama0530/ccc/main/install.sh | env CCC_INSTALL_DIR="$HOME/bin" bash
curl -fsSL https://raw.githubusercontent.com/ishiyama0530/ccc/main/install.sh | env CCC_INSTALL_VERSION=vX.Y.Z bash
```

## ❤️ Why people like it

- ⚡ Fast path back to the exact Claude session you want
- 🎯 Project-aware by default, so results stay relevant to the repo you are in
- 👀 Preview-first flow, so you can confirm before you resume

## 🧠 How it works

```bash
ccc
ccc bug
ccc -d ~/src/app timeout
ccc -n 200
```

- Run `ccc` inside the project you care about
- By default, it searches the Claude history for your current working directory
- `-d` / `--dir` switches the target working directory
- `-n` / `--limit` sets how many history entries can be shown, default `100`
- With no query, `ccc` lists up to `100` session history entries for the target directory
- Search is case-insensitive
- If matches are found, `ccc` opens the TUI
- The command bar always keeps `claude --resume <session_id>` fixed and appends any extra Claude args you type
- If matches are found without a TTY, `ccc` exits with an error
- If nothing matches, `ccc` prints an error to stderr and exits non-zero

## 💡 Example commands

```bash
ccc
ccc -n 200
ccc bug
ccc -d ~/src/app
ccc -d ~/src/app -n 50 timeout
ccc --dir ~/src/app timeout
```

## ⌨️ Picker keys

- `↑` / `↓`: move
- `Shift+↑` / `Shift+↓`: scroll the preview line by line
- `Enter`: resume the selected session
- Type text: add extra Claude args
- `Backspace`: edit extra args
- `PgUp` / `PgDn`: scroll the preview
- `Ctrl+U` / `Ctrl+D`: scroll the preview faster
- `esc` / `ctrl+c`: quit

## 🛠️ Development

```bash
make build
make test
make lint
make run QUERY="bug"
# run the local bin/ccc without installing it
PATH="$PWD/bin:$PATH" ccc bug
```

## 📦 Release

```bash
export GITHUB_TOKEN=...
export NPM_TOKEN=...
make release VERSION=vX.Y.Z
```

`make release` publishes the GitHub Release assets used by `install.sh`, then publishes `@ishiyama0530/ccc` to npm.
