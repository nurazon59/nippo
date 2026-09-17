#!/usr/bin/env bash
# 指定日の Claude Code / Codex セッションから、実際のユーザー入力を短く集約する。
# org は実行ディレクトリの git remote から判定する（NIPPO_ORG で上書き可）。
# 使い方: ./claude-daily-sessions.sh [YYYY-MM-DD]

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

if [[ $# -ge 1 && -n "$1" ]]; then
  TARGET_DATE="$1"
else
  TARGET_DATE=$(shift_date "$(date '+%Y-%m-%d')" -86400)
fi

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

PROJECTS_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}/projects"
CODEX_SESSIONS_DIR="${CODEX_HOME:-$HOME/.codex}/sessions"

format_records() {
  local records="$1"
  local record

  [[ -z "$records" ]] && return

  while IFS= read -r record; do
    echo "- $(jq -r '.text' <<< "$record")"
  done <<< "$records"
}

echo "## Claude Code セッション（${TARGET_DATE} / ${ORG}）"
echo ""

if [[ -d "$PROJECTS_DIR" ]]; then
  while IFS= read -r f; do
    project=$(basename "$(dirname "$f")" \
      | sed 's/^-Users-[^-]*-src-github-com-//' \
      | sed 's/-/\//')

    records=$(jq -c \
      --arg date "$TARGET_DATE" \
      --argjson offset "$UTC_OFFSET_SECONDS" \
      '
        select(.type == "user" and (.timestamp // "") != "")
        | ([
             if (.message.content | type) == "array"
             then (.message.content[]? | select(.type == "text") | .text)
             else (.message.content // "")
             end
             | select(startswith("# AGENTS.md instructions") | not)
             | select(startswith("<environment_context>") | not)
             | select(startswith("<recommended_plugins>") | not)
             | select(startswith("<turn_aborted>") | not)
             | select(length > 0)
           ] | join("\n")) as $text
        | select($text != "")
        | ((.timestamp | sub("\\.[0-9]+Z$"; "Z") | fromdateiso8601) + $offset) as $epoch
        | select(($epoch | strftime("%Y-%m-%d")) == $date)
        | {time: ($epoch | strftime("%H:%M")), text: ($text | gsub("[[:space:]]+"; " ") | .[0:160])}
      ' "$f")

    [[ -z "$records" ]] && continue

    echo "### $(jq -r '.time' <<< "$records" | head -1) — ${project}"
    format_records "$records" | sed -n '1,3p'
    echo ""
  done < <(find "$PROJECTS_DIR" -maxdepth 2 -path "*-github-com-${ORG}-*" -name "*.jsonl" -print 2>/dev/null | sort)
fi

echo "## Codex セッション（${TARGET_DATE} / ${ORG}）"
echo ""

codex_found=false
if [[ -d "$CODEX_SESSIONS_DIR" ]]; then
  while IFS= read -r f; do
    records=$(jq -c -s \
      --arg date "$TARGET_DATE" \
      --arg org "$ORG" \
      --argjson offset "$UTC_OFFSET_SECONDS" \
      '
        first(.[] | select(.type == "session_meta") | .payload) as $meta
        | ($meta.git.repository_url // "") as $origin
        | select(
            ($origin | contains("github.com/" + $org + "/")) or
            ($origin | contains("github.com:" + $org + "/")) or
            (($meta.cwd // "") | contains("/src/github.com/" + $org + "/"))
          )
        | ($origin | split("/") | last | sub("\\.git$"; "")) as $repo
        | (if $repo != "" then $org + "/" + $repo else ($meta.cwd // "" | split("/") | last) end) as $project
        | [
            .[]
            | select(
                .type == "response_item" and
                .payload.type == "message" and
                .payload.role == "user" and
                (.timestamp // "") != ""
              )
            | ([
                 .payload.content[]?
                 | select(.type == "input_text")
                 | .text
                 | select(startswith("# AGENTS.md instructions") | not)
                 | select(startswith("<environment_context>") | not)
                 | select(startswith("<recommended_plugins>") | not)
                 | select(startswith("<turn_aborted>") | not)
                 | select(length > 0)
               ] | join("\n")) as $text
            | select($text != "")
            | ((.timestamp | sub("\\.[0-9]+Z$"; "Z") | fromdateiso8601) + $offset) as $epoch
            | select(($epoch | strftime("%Y-%m-%d")) == $date)
            | {time: ($epoch | strftime("%H:%M")), project: $project, text: ($text | gsub("[[:space:]]+"; " ") | .[0:160])}
          ]
        | .[0:3][]
      ' "$f")

    [[ -z "$records" ]] && continue

    codex_found=true
    project=$(jq -r '.project' <<< "$records" | head -1)
    echo "### $(jq -r '.time' <<< "$records" | head -1) — ${project}"
    format_records "$records"
    echo ""
  done < <(find "$CODEX_SESSIONS_DIR" -type f -name "*.jsonl" -print 2>/dev/null | sort)
fi

if [[ "$codex_found" == false ]]; then
  echo "（対象セッションなし）"
fi
