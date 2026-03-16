# ccc

[English](./README.md) | [日本語](./README.ja.md)

`ccc` は、Claude Code のセッションを見つけてすばやく再開するための小さな CLI です。

## インストール

### シェルインストーラー

まずはこれでインストールできます。

```bash
curl -fsSL https://raw.githubusercontent.com/ishiyama0530/ccc/main/install.sh | bash
```

- 最新の GitHub Release をダウンロードします。
- 対応環境は macOS / Linux、`amd64` / `arm64` です。
- `ccc` はデフォルトで `~/.local/bin` に入ります。

`~/.local/bin` が PATH に入っていない場合は、シェル設定に追加してください。


### npm

```bash
npm install -g @ishiyama0530/ccc
```

- macOS / Linux / Windows で利用できます。
- `amd64` / `arm64` に対応しています。
- `postinstall` 時に環境に合った GitHub Release バイナリを取得します。

インストール先を変える場合:

```bash
curl -fsSL https://raw.githubusercontent.com/ishiyama0530/ccc/main/install.sh | env CCC_INSTALL_DIR="$HOME/bin" bash
```

バージョンを固定する場合:

```bash
curl -fsSL https://raw.githubusercontent.com/ishiyama0530/ccc/main/install.sh | env CCC_INSTALL_VERSION=vX.Y.Z bash
```

## 使い方

```bash
ccc
ccc -d <dir>
ccc <query>
ccc -d <dir> <query>
ccc --dir <dir> <query>
```

- デフォルトでは、現在の作業ディレクトリに対応する Claude 履歴を検索します。
- `-d` / `--dir` で検索対象の作業ディレクトリを切り替えます。
- クエリを省略すると、対象ディレクトリの全セッション履歴を一覧表示します。
- 検索は大文字小文字を区別しません。
- 一致が見つかると TUI を開きます。
- 追加引数は下部のコマンドバーに入力します。`claude --resume <session_id>` は固定で、その後ろに引数を追加します。
- TTY なしで一致が見つかった場合はエラー終了します。
- 0 件なら stderr にエラーを出して非 0 で終了します。

例:

```bash
ccc
ccc bug
ccc -d ~/src/app
ccc -d ~/src/app timeout
```

## キー操作

- `↑` / `↓`: 移動
- `Shift+↑` / `Shift+↓`: プレビューを1行ずつスクロール
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
make release VERSION=vX.Y.Z
```

`make release` は、`install.sh` と npm `postinstall` が使う GitHub Release を公開します。

npm へ公開する場合は、続けて次を実行します。

```bash
npm version <major|minor|patch>
npm publish
```
