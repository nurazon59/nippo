#!/usr/bin/env bash
# 指定日（デフォルト: 昨日）の macOS カレンダー予定を icalBuddy で集約して出力する
# 使い方: ./calendar-daily-events.sh [YYYY-MM-DD]

set -uo pipefail

TARGET_DATE="${1:-$(date -v-1d '+%Y-%m-%d')}"

echo "## カレンダー予定（${TARGET_DATE}）"
echo ""

events=$(icalBuddy \
  -npn -nc -nrd \
  -b "- " \
  -ps "| @ |" \
  -iep "title,datetime" \
  -po "datetime,title" \
  -df "" \
  eventsFrom:"$TARGET_DATE" to:"$TARGET_DATE" 2>/dev/null) || true

if [[ -z "$events" ]]; then
  echo "予定なし"
  exit 0
fi

echo "$events"
