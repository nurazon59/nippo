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

### 保存形式（構造化スキーマ v1）

日報の正本 (canonical) は YAML、Markdown はそれを renderer が組み立てる派生物として扱う。

- `<storage_dir>/nippo/YYYY/MM/DD.yaml` ... canonical。`generate` / `edit` / `migrate --apply` が書き込む。
- `<storage_dir>/nippo/YYYY/MM/DD.md` ... .yaml から render される副生成物 (sidecar)。閲覧・grep 用。

サンプル (`schema_version: 1`):

```yaml
schema_version: 1
date: 2026-05-23
fields:
  done:
    type: task_list
    tasks:
      - title: PR レビュー対応
        time: 1h30m
        outcome: マージ済
        thoughts: |
          指摘事項は概ね妥当だった
      - title: 設計ドキュメント更新
  todo:
    type: text
    body: |
      - 残タスクの優先度整理
  thoughts:
    type: text
    body: ""
```

`schema_version` と `date` は必須。`fields` の各値は `type` に応じて `body` (text) もしくは `tasks` (task_list) のいずれかだけを持つ。未知の `type` は load 時に reject する。

### 質問設定（`questions`）

`questions` は config の最上位に並べる。`type` を省略すると `text` 扱い。

```yaml
questions:
  - key: done
    label: やった
    required: true
    type: task_list
  - key: todo
    label: やる
    required: true
  - key: thoughts
    label: 所感
    required: false
```

- `type: text` ... 自由記述。`body` を持つ。
- `type: task_list` ... 1 タスクごとに title / time / outcome / thoughts を入力するフォーム。`tasks` を持つ。
- `reference_key` / `same_day_reference_key` ... 別質問の既存内容を参照プレビューとして差し込む（既存機能）。

未知の `type` は config load 時にエラーで終了する（silent fallback はしない）。

### edit コマンド（フォーム再開モデル）

`nippo edit YYYY-MM-DD` は当該日の `.yaml` を読み戻し、`generate` と同じフォームを既存値を default として再表示する。enter で既存値維持、`(e)` でエディタ再起動。保存時は `.yaml` と `.md` の両方が更新される。

旧来の「`.md` を vim で直接編集する」モデルは canonical (.yaml) と乖離するため廃止した。

### migrate コマンド（legacy .md → .yaml）

`schema_version: 1` 導入前に書かれた `.md` だけの日報を、構造化 YAML に変換する。デフォルトは dry-run、`--apply` 指定時のみ書き込む。

```bash
# 何が変わるか確認 (dry-run がデフォルト)
nippo migrate

# 実際に書き込む
nippo migrate --apply

# 単一日だけ移行する
nippo migrate --apply --date 2024-06-15

# 機械可読な JSON 出力 (mode / per-date result / summary)
nippo migrate --format json
```

挙動メモ:

- 既に `.yaml` がある日付は skip する（冪等）。
- `.md` 内の `## <label>` が `questions[].label` と一致する section だけを採用する。一致しない見出しは stderr に警告として出すだけで、変換は続行する。
- 該当 question の `type: task_list` の section は `- title (time) outcome` 形式の bullet を best-effort で再パースする。
- 変換に失敗した日付は summary の `fail` にカウントされ、他の日付は影響を受けない。

### 互換性メモ

- legacy `.md` だけが存在する日報は `nippo show` / `nippo latest` で従来通り表示できる。
- 一方で `nippo edit` のフォーム再開や `reference_key` の preset 化は `.yaml` を前提に動くため、過去日を活用したいなら `nippo migrate --apply` を推奨する。

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
