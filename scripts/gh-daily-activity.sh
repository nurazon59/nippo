#!/usr/bin/env bash
# 指定日の自分の PR 作成・レビュー・コメントを、起動した repo の org に限定して集約する。
# org は実行ディレクトリの git remote から判定する（NIPPO_ORG で上書き可）。
# 使い方: ./gh-daily-activity.sh [YYYY-MM-DD]

set -euo pipefail

UTC_OFFSET_SECONDS="${NIPPO_UTC_OFFSET_SECONDS:-32400}"
if ! [[ "$UTC_OFFSET_SECONDS" =~ ^-?[0-9]+$ ]]; then
  echo "NIPPO_UTC_OFFSET_SECONDS は整数で指定してください。" >&2
  exit 1
fi

shift_date() {
  jq -nr --arg date "$1" --argjson delta "$2" \
    '$date | strptime("%Y-%m-%d") | mktime + $delta | strftime("%Y-%m-%d")'
}

date_to_epoch() {
  jq -nr --arg date "$1" --argjson offset "$UTC_OFFSET_SECONDS" \
    '$date | strptime("%Y-%m-%d") | mktime - $offset'
}

if [[ $# -ge 1 && -n "$1" ]]; then
  TARGET_DATE="$1"
else
  TARGET_DATE=$(shift_date "$(date '+%Y-%m-%d')" -86400)
fi
NEXT_DATE=$(shift_date "$TARGET_DATE" 86400)
SEARCH_START_DATE=$(shift_date "$TARGET_DATE" -86400)
START_EPOCH=$(date_to_epoch "$TARGET_DATE")
END_EPOCH=$(date_to_epoch "$NEXT_DATE")

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

LOGIN=$(gh api user --jq '.login')

echo "## GitHub PR アクティビティ（${TARGET_DATE} / ${ORG}）"
echo ""

search_prs() {
  local activity="$1"
  shift
  gh search prs "$@" \
    --owner "$ORG" \
    --updated "${SEARCH_START_DATE}..${NEXT_DATE}" \
    --limit 100 \
    --json number,title,updatedAt,state,repository \
    | jq -r --arg activity "$activity" \
      '.[] | "\(.repository.nameWithOwner)\t\(.number)\t\(.state)\t\(.title)\t\($activity)"'
}

# GitHub の検索 qualifier は活動種別ごとに分かれているため、結果を和集合にする。
prs=$(
  {
    search_prs "作成" --author "@me"
    search_prs "レビュー" --reviewed-by "@me"
    search_prs "コメント" --commenter "@me"
  } | awk -F '\t' '
    BEGIN { OFS = "\t" }
    {
      key = $1 "\t" $2
      if (!(key in seen)) {
        seen[key] = 1
        order[++count] = key
        repo[key] = $1
        number[key] = $2
        state[key] = $3
        title[key] = $4
      }
      if (activities[key] == "") {
        activities[key] = $5
      } else if (activities[key] !~ "(^|/)" $5 "(/|$)") {
        activities[key] = activities[key] "/" $5
      }
    }
    END {
      for (i = 1; i <= count; i++) {
        key = order[i]
        print repo[key], number[key], state[key], title[key], activities[key]
      }
    }'
)

if [[ -z "$prs" ]]; then
  echo "対象 PR なし"
  exit 0
fi

echo "### PR 一覧"
while IFS=$'\t' read -r repo num state title activity; do
  echo "- ${repo}#${num} [${activity}/${state}] ${title}"
done <<< "$prs"
echo ""

fetch_activities() {
  local endpoint="$1"
  local kind="$2"
  local timestamp_field="$3"

  gh api --paginate --slurp "$endpoint" \
    | jq -r \
      --arg login "$LOGIN" \
      --arg kind "$kind" \
      --arg timestamp_field "$timestamp_field" \
      --argjson start "$START_EPOCH" \
      --argjson end "$END_EPOCH" \
      'add[]?
       | select((.user.login // "") == $login)
       | (.[$timestamp_field] // "") as $timestamp
       | select((try ($timestamp | fromdateiso8601) catch -1) >= $start)
       | select((try ($timestamp | fromdateiso8601) catch -1) < $end)
       | [
           $kind,
           (.state // "-"),
           ((.body // "") | split("\n") | .[0] // "" | gsub("[\t\r]"; " ") | .[0:120]
             | if . == "" then "（本文なし）" else . end)
         ]
       | @tsv'
}

echo "### 自分のレビュー・コメント"
found_any=false
activity_tmp=$(mktemp -d "${TMPDIR:-/tmp}/nippo-gh.XXXXXX")
trap 'rm -rf "$activity_tmp"' EXIT
activity_index=0

while IFS=$'\t' read -r repo num _state title _activity; do
  if [[ "$_activity" != *"レビュー"* && "$_activity" != *"コメント"* ]]; then
    continue
  fi

  activity_index=$((activity_index + 1))
  current_tmp="${activity_tmp}/${activity_index}"
  mkdir -p "$current_tmp"
  pids=()

  if [[ "$_activity" == *"レビュー"* ]]; then
    fetch_activities "repos/${repo}/pulls/${num}/reviews" "レビュー" "submitted_at" \
      > "${current_tmp}/reviews" &
    pids+=("$!")
  fi
  if [[ "$_activity" == *"コメント"* ]]; then
    fetch_activities "repos/${repo}/issues/${num}/comments" "コメント" "created_at" \
      > "${current_tmp}/comments" &
    pids+=("$!")
    fetch_activities "repos/${repo}/pulls/${num}/comments" "インラインコメント" "created_at" \
      > "${current_tmp}/inline-comments" &
    pids+=("$!")
  fi

  fetch_status=0
  for pid in "${pids[@]}"; do
    wait "$pid" || fetch_status=1
  done
  if [[ "$fetch_status" -ne 0 ]]; then
    echo "警告: ${repo}#${num} のアクティビティ取得に失敗しました。" >&2
    continue
  fi

  activities=$(
    cat "${current_tmp}"/*
  )

  [[ -z "$activities" ]] && continue

  found_any=true
  echo ""
  echo "#### ${repo}#${num} ${title}"
  while IFS=$'\t' read -r kind state body; do
    if [[ -n "$state" ]]; then
      echo "- [${kind}/${state}] ${body}"
    else
      echo "- [${kind}] ${body}"
    fi
  done <<< "$activities"
done <<< "$prs"

if [[ "$found_any" == false ]]; then
  echo "（指定日に自分が投稿したレビュー・コメントなし）"
fi
