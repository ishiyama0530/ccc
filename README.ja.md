# ccc

[English](./README.md) | [日本語](./README.ja.md)

`ccc` は、Claude Code のセッションを見つけてすばやく再開するための小さな CLI です。

## インストール

Homebrew:

```bash
brew tap owner/homebrew-tap
brew install ccc
```

## 使い方

```bash
ccc <query>
ccc -d <dir> <query>
ccc --dir <dir> <query>
```

- デフォルトでは、現在の作業ディレクトリに対応する Claude 履歴を検索します。
- `-d` / `--dir` で検索対象の作業ディレクトリを切り替えます。
- 検索は大文字小文字を区別しません。
- 一致が見つかると TUI を開きます。
- 追加引数は下部のコマンドバーに入力します。`claude --resume <session_id>` は固定で、その後ろに引数を追加します。
- TTY なしで一致が見つかった場合はエラー終了します。
- 0 件なら stderr にエラーを出して非 0 で終了します。

例:

```bash
ccc bug
ccc -d ~/src/app timeout
```

## キー操作

- `↑` / `↓`: 移動
- `Enter`: 選択したセッションを再開
- 文字入力: 追加引数を入力
- `Backspace`: 追加引数を編集
- `PgUp` / `PgDn`: プレビューをスクロール
- `Ctrl+U` / `Ctrl+D`: プレビューを速くスクロール
- `esc` / `ctrl+c`: 終了

## 開発

```bash
make build
make test
make lint
make run QUERY="bug"
# インストールせずにローカルの bin/ccc を一時的に実行
PATH="$PWD/bin:$PATH" ccc bug
```

## リリース

```bash
export GITHUB_TOKEN=...
make release VERSION=vX.Y.Z TAP_REPO=owner/homebrew-tap
```
