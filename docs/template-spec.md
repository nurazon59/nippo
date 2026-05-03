# Go CLI テンプレート仕様

## 目的

このリポジトリは、単一バイナリの Go 製 CLI を作るための template repository として使う。

テンプレートには、新しいリポジトリを始める時点で最低限実用になる次の要素を含める。

- 動く CLI エントリポイント
- XDG ベースの設定ファイル読込
- CI
- tag ベースの release フロー
- GitHub Releases への配布

この文書は、実装前に合意した要件をまとめたもの。

## 対象範囲

- 対象は単一バイナリの Go CLI
- リポジトリ名がそのままバイナリ名になる想定
- 配布先は GitHub Releases のみ
- Homebrew は初期テンプレートの対象外
- Docker image 配布は初期テンプレートの対象外
- Scoop は初期テンプレートの対象外

## CLI 設計

- CLI フレームワークは `kong`
- 初期 CLI は意図的に薄く保つ
- 明示的に持つ初期サブコマンドは `version` のみ
- `help` は Kong 標準の挙動に任せる
- `version` と `--version` の両方をサポートする
- `version` と `--version` の出力はプレーンテキストで `v0.1.0` のみ
- commit hash や build date などの build metadata は埋め込まない

## コード構成

- エントリポイントは `cmd/main.go`
- 初期テンプレートでは `internal/` を導入しない
- 初期構成はできるだけ薄く保つ
- `config.go` は分けて持つ前提
- `cli.go` や `version.go` は初期状態では必須ではない

## 設定

### 基本方針

- 設定ファイル読込の仕組みは最初から持つ
- 設定ファイル形式は YAML
- YAML ライブラリは `github.com/goccy/go-yaml`
- 初期の config schema は `version` のみを持つ
- この `version` はアプリケーション version ではなく config schema version

設定イメージ:

```yaml
version: 1
```

### 保存場所

- 設定保存先は XDG に従う
- 設定ファイル path は `~/.config/<app>/config.yaml`
- 設定ファイル名は `config.yaml`

### 実行時挙動

- 設定ファイルが存在しない場合は default 値で起動する
- 設定ファイルが存在し、かつ不正な内容なら明示的にエラー終了する
- 初期テンプレートには config 生成コマンドは入れない
- 初期状態では「存在すれば読む」だけでよい

## バージョン管理

- アプリケーション version はソースコード上で固定文字列 `v0.1.0` として持つ
- config schema version と application version は分離する
- build 時の metadata 埋め込みは対象外
- `--version` の出力は最小構成で安定させる

## 開発ツール

### ローカルツール管理

- ローカルのツール管理は `mise`
- `mise.toml` に含めるツール:
- `go`
- `task`
- `golangci-lint`
- `goreleaser`
- これらはすべて version pin する

### タスクランナー

- タスクランナーは `Taskfile.yml`
- 最低限入れるタスク:
- `test`
- `fmt`
- `lint`
- `build`
- `ci`
- `snapshot`

各タスクの想定:

- `fmt`: `go fmt ./...`
- `test`: `go test ./...`
- `lint`: `golangci-lint` 実行
- `build`: `./bin/<app>` へのローカル確認用 build
- `ci`: format check、`go vet`、lint、test の集約
- `snapshot`: `goreleaser release --snapshot --clean`

### format / lint 方針

- format は `golangci-lint` に寄せない
- format コマンドは `go fmt ./...`
- CI の format check は format 実行後に差分が残らないことを確認する
- `go vet` は `golangci-lint` に寄せず別で実行する
- `golangci-lint` は薄く実用的に設定する
- 主目的は厳格運用ではなく、一元実行の入口にすること
- `staticcheck` は明示的に有効化する
- それ以外の lint 設定は最小限に留める

## テスト

- 最初からサンプルテストを入れる
- テスト方針は black-box 寄り
- 初期テスト対象は `version` と `--help`
- `help` サブコマンド専用のテストは不要
- 基本の test コマンドは `go test ./...`

## GitHub Actions

### Workflow 構成

- 初期 workflow は次の 3 本
- `CI`
- `Tagpr`
- `Release`

workflow file 名:

- `.github/workflows/ci.yml`
- `.github/workflows/tagpr.yml`
- `.github/workflows/release.yml`

### CI 方針

- CI では `actions/setup-go` を使う
- Go version は `stable`
- CI は `mise` に依存しない
- CI で回す内容:
- test
- format check
- `go vet`
- `golangci-lint`

## リリースフロー

### tag / release 管理

- `tagpr` を使う
- `tagpr` は GitHub Actions 上だけで動かす
- `tagpr` をローカルツールとして入れる必要はない
- 想定フロー:
- 通常の PR が `main` にマージされる
- `tagpr` が release PR を作る
- release PR をマージする
- tag が作られる
- その tag をトリガーに release automation が走る

### リリースノート

- リリースノートは `tagpr` と GitHub の自動生成ベースでよい

### GoReleaser

- `goreleaser` を使う
- 配布先は GitHub Releases のみ
- 圧縮アーカイブと checksum file を出す
- artifact 名は GoReleaser のデフォルト寄りでよい
- 対応ターゲット:
- `darwin`
- `linux`
- `windows`
- `amd64`
- `arm64`
- `CGO_ENABLED=0` を基本とする

## リポジトリファイル

### README

- README は英語
- README は使い方中心にする
- テンプレート利用直後のチェックリスト中心にはしない

### License

- MIT license を維持する

### Git Ignore

- `.gitignore` は生成物中心でよい
- ローカル config はリポジトリ配下に置かない前提
- 想定する ignore 対象:
- build 生成物
- `dist/`
- 一時ファイル

## 初期版でやらないこと

- Homebrew tap 自動化
- Docker image 配布
- Scoop 対応
- commit/date などの build metadata 埋め込み
- 重い lint policy
- `internal/` を使った構成
- config 生成コマンド
- 初期から大きい command surface を持つこと

## 実装メモ

初期実装は小さく、読めばすぐ分かる形にする。

- 薄い CLI
- 薄い config 処理
- 薄い automation
- tag ベースで release できる最低限のツール群

テンプレート全体として、賢さよりも追いやすさを優先する。
