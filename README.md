# go-template

`go-template` is a small Go CLI template with a thin `kong`-based entrypoint, XDG-backed config loading, and release automation for GitHub Releases.

## Usage

```bash
go run ./cmd/template/main.go --help
go run ./cmd/template/main.go --version
```

### Config

The CLI loads config from one of these locations:

1. `--config /path/to/config.yaml`
2. `GO_TEMPLATE_CONFIG=/path/to/config.yaml`
3. `~/.config/go-template/config.yaml`

Config format:

```yaml
version: 1
```

If the file is missing, the CLI starts with defaults.

### Hooks（外部コマンド連携）

`hooks:` セクションを追加すると、`nippo generate` のエディタ起動時に任意の外部コマンドの実行結果を `<!-- ... -->` 形式で参考情報として埋め込める。認証は外部 CLI（`gh`, `gcalcli`, `icalBuddy` 等）に委譲する設計のため、nippo 自体は token/OAuth を保持しない。

- `command` は `sh -c` で実行されるので、`|` や `$()` をそのまま使える
- 環境変数 `NIPPO_DATE=YYYY-MM-DD` が hook プロセスに渡される（対象日に応じてクエリを組み立てられる）
- `keys` に列挙した質問 key のエディタ画面に stdout が挿入される
- `timeout` は Go の `time.ParseDuration` 形式（デフォルト `30s`）。タイムアウト・非 0 終了・コマンド未存在の場合は警告を stderr に出して generate 自体は続行する
- hook は並列実行される

#### 設定例

```yaml
hooks:
  - name: github-opened
    command: "gh search prs --author=@me --created=$NIPPO_DATE --json title,url,state -L 20 | jq -r '.[] | \"- [opened] \(.state) \(.title) \(.url)\"'"
    keys: [done]
  - name: github-reviewed
    command: "gh search prs --reviewed-by=@me --updated=$NIPPO_DATE --json title,url,state -L 20 | jq -r '.[] | \"- [reviewed] \(.state) \(.title) \(.url)\"'"
    keys: [done]
  - name: github-commented
    command: "gh search prs --commenter=@me --updated=$NIPPO_DATE --json title,url,state -L 20 | jq -r '.[] | \"- [commented] \(.state) \(.title) \(.url)\"'"
    keys: [done]
  - name: calendar
    command: "icalBuddy -ea -nrd -df '' -tf '%H:%M' eventsToday"
    keys: [done]
```

`$NIPPO_DATE` は hook 実行時に対象日（`YYYY-MM-DD`）が渡される。上記は当日 open / reviewed / commented した PR を 3 つの hook で取得し、エディタの「やった」欄に挿入する例。

### シェル補完

`nippo completion <shell>` で bash/zsh/fish 用の補完スクリプトを生成する。

```bash
# bash (現在のシェルで読み込む)
source <(nippo completion bash)

# bash (永続化)
nippo completion bash > /etc/bash_completion.d/nippo

# zsh (fpath 上のディレクトリに配置)
nippo completion zsh > "${fpath[1]}/_nippo"

# fish
nippo completion fish > ~/.config/fish/completions/nippo.fish
```

## Development

```bash
task fmt
task test
task lint
task vet
task ci
task snapshot
```
