#!/usr/bin/env bash
# 指定日（デフォルト: 昨日）の PR コメント・レビューを、起動した repo の org に限定して集約・出力する
# org は実行ディレクトリの git remote から判定する（work で personal を拾わないため）
# 使い方: ./gh-daily-activity.sh [YYYY-MM-DD]

set -uo pipefail

TARGET_DATE="${1:-$(date -v-1d '+%Y-%m-%d')}"
NEXT_DATE=$(date -j -v+1d -f '%Y-%m-%d' "$TARGET_DATE" '+%Y-%m-%d')

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

BOT_LOGINS="coderabbitai chatgpt-codex-connector notion-workspace"

is_bot() {
  local login="$1"
  for bot in $BOT_LOGINS; do
    [[ "$login" == "$bot" ]] && return 0
  done
  return 1
}

echo "## GitHub PR アクティビティ（${TARGET_DATE} / ${ORG}）"
echo ""

prs=$(gh search prs --author "@me" --owner "$ORG" --updated "${TARGET_DATE}..${NEXT_DATE}" --limit 50 \
  --json number,title,updatedAt,state,repository \
  | jq -r '.[] | "\(.repository.nameWithOwner)\t\(.number)\t\(.state)\t\(.title)"')

if [[ -z "$prs" ]]; then
  echo "対象 PR なし"
  exit 0
fi

echo "### PR 一覧"
while IFS=$'\t' read -r repo num state title; do
  echo "- ${repo}#${num} [${state}] ${title}"
done <<< "$prs"
echo ""

echo "### レビュー・コメント詳細"

found_any=false
while IFS=$'\t' read -r repo num _state title; do
  # PR が削除・移動されている場合は 404 になりうる
  if ! reviews=$(gh pr view "$num" --repo "$repo" \
    --jq ".reviews[] | select(.submittedAt >= \"${TARGET_DATE}\" and .submittedAt < \"${NEXT_DATE}\") | \"\(.author.login)\t\(.state)\t\(.body | split(\"\n\")[0] | .[0:100])\"" \
    --json reviews 2>&1); then
    echo "警告: ${repo}#${num} のレビュー取得に失敗しました: ${reviews}" >&2
    continue
  fi

  if ! comments=$(gh pr view "$num" --repo "$repo" \
    --jq ".comments[] | select(.createdAt >= \"${TARGET_DATE}\" and .createdAt < \"${NEXT_DATE}\") | \"\(.author.login)\t\(.body | split(\"\n\")[0] | .[0:100])\"" \
    --json comments 2>&1); then
    echo "警告: ${repo}#${num} のコメント取得に失敗しました: ${comments}" >&2
    continue
  fi

  # ボット行を除去
  human_reviews=""
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    login=$(echo "$line" | cut -f1)
    is_bot "$login" || human_reviews+="${line}"$'\n'
  done <<< "$reviews"

  human_comments=""
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    login=$(echo "$line" | cut -f1)
    is_bot "$login" || human_comments+="${line}"$'\n'
  done <<< "$comments"

  if [[ -z "$human_reviews" && -z "$human_comments" ]]; then
    continue
  fi

  found_any=true
  echo ""
  echo "#### ${repo}#${num} ${title}"

  if [[ -n "$human_reviews" ]]; then
    while IFS=$'\t' read -r login state body; do
      [[ -z "$login" ]] && continue
      echo "- **[${state}]** @${login}: ${body}"
    done <<< "$human_reviews"
  fi

  if [[ -n "$human_comments" ]]; then
    while IFS=$'\t' read -r login body; do
      [[ -z "$login" ]] && continue
      echo "- @${login}: ${body}"
    done <<< "$human_comments"
  fi
done <<< "$prs"

$found_any || echo "（人間によるレビュー・コメントなし）"
