#!/usr/bin/env bash
# 指定日（デフォルト: 昨日）の Claude Code セッション内容を、起動した repo の org に限定して集約・出力する
# org は実行ディレクトリの git remote から判定する（work で personal を拾わないため）
# 使い方: ./claude-daily-sessions.sh [YYYY-MM-DD]

set -uo pipefail

TARGET_DATE="${1:-$(date -v-1d '+%Y-%m-%d')}"
PROJECTS_DIR="$HOME/.config/claude/projects"

# 起動した repo の org を判定（環境変数 NIPPO_ORG で上書き可）
ORG="${NIPPO_ORG:-}"
if [[ -z "$ORG" ]]; then
  ORG=$(gh repo view --json owner --jq '.owner.login' 2>/dev/null) || true
fi
if [[ -z "$ORG" ]]; then
  ORG=$(git remote get-url origin 2>/dev/null | sed -E 's#.*github\.com[:/]([^/]+)/.*#\1#') || true
fi
if [[ -z "$ORG" ]]; then
  echo "org を判定できませんでした（git repo 外、または remote 未設定）。NIPPO_ORG で指定してください。" >&2
  exit 1
fi

echo "## Claude Code セッション（${TARGET_DATE} / ${ORG}）"
echo ""

while IFS= read -r f; do
  file_date=$(date -r "$f" '+%Y-%m-%d')
  [[ "$file_date" == "$TARGET_DATE" ]] || continue

  # プロジェクト名: -Users-<user>-src-github-com-<org>-<repo> → <org>/<repo>
  project=$(basename "$(dirname "$f")" \
    | sed 's/^-Users-[^-]*-src-github-com-//' \
    | sed 's/-/\//')  # 最初の - だけ / に置換

  file_time=$(date -r "$f" '+%H:%M')

  # ユーザーメッセージを抽出
  # - 配列の場合は最初のtextブロックのみ取得（後続はskillコンテキスト注入）
  # - skill注入マーカー以降を削除
  # - XMLタグブロック・自動挿入ノイズを除去
  messages=$(jq -r 'select(.type == "user") | .message.content
    | if type == "array" then (.[0] | select(.type == "text") | .text) // ""
      elif type == "string" then .
      else "" end' "$f" 2>/dev/null \
    | sed '/^Base directory for this skill:/,$d' \
    | sed '/^ARGUMENTS:/,$d' \
    | sed '/<[a-z][a-z-]*/,/<\/[a-z]/d' \
    | sed '/^\[Request interrupted/d' \
    | sed 's/^[[:space:]]*//' \
    | grep -v '^$' \
    | head -3) || true

  [[ -z "$messages" ]] && continue

  echo "### ${file_time} — ${project}"
  while IFS= read -r line; do
    echo "- ${line:0:150}"
  done <<< "$messages"
  echo ""
done < <(find "$PROJECTS_DIR" -maxdepth 2 -path "*-github-com-${ORG}-*" -name "*.jsonl" | sort)
