# nippo プロジェクト規約

## テスト

- **テーブルドリブンを採用する。** `map[string]struct{...}` でケースを宣言し、`for name, tt := range tests { t.Run(name, ...) }` で回す。
  - ケース名はキーで表現（`"timeout drops entry but other hooks survive"` のような宣言的な文）。
  - 参考実装: `latest_test.go`, `hooks_test.go`, `config_test.go`, `reference_test.go`。
- 同種の検証は1つのテーブルにまとめる。`t.Run` を別関数で並べる書き方は採らない。
- ケース構造体には「入力」と「期待値（want* / absent* / maxElapsed 等）」を持たせ、ループ内で1回だけ対象を実行して全 assertion を流す。
- assertion は `stretchr/testify` の `assert` / `require`。失敗時のキー特定が難しい場合は `assert.Equal(t, want, got, "key=%s", k)` のようにフォーマット引数で明示。

## 言語

- コメント・コミットメッセージ・ドキュメントはすべて日本語。
- コミット規約: `<type>(<scope>): <subject>` （例: `feat(hooks): ...`, `test(latest): ...`, `refactor(cli): ...`）。

## 開発フロー

- TDD: テスト先行 → 失敗確認 → 実装 → パス。
- 変更後は `go build ./...` / `go test ./... -race -count=1` / `go vet ./...` / `gofmt -l .` を通してからコミットする。
- Lint の無視コメント (`//nolint` 等) は禁止。
