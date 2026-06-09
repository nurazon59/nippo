package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/nurazon59/nippo/report"
)

// migrateCmd は legacy `.md` のみが存在する日報を parse して canonical `.yaml` を生成する。
// CLAUDE.md のスクリプト設計方針に従い、デフォルトは dry-run、`--apply` 指定時のみ副作用を実行する。
type migrateCmd struct {
	Apply  bool   `help:"Apply the migration (default is dry-run)."`
	Date   string `help:"Migrate a single date (YYYY-MM-DD). Omit to scan all."`
	Format string `help:"Output format: text or json." default:"text" enum:"text,json"`
}

// migrateAction は migrate 1 件の結果分類。
// dry-run と apply では action 文字列が変わる (would-migrate vs migrated) ため emit 時に解決する。
type migrateAction string

const (
	migrateActionMigrate migrateAction = "migrate"
	migrateActionSkip    migrateAction = "skip"
	migrateActionFail    migrateAction = "fail"
)

// migrationResult は 1 日報の processOne の出力。emitReport で JSON / text に整形される。
type migrationResult struct {
	Date            time.Time
	Action          migrateAction
	Sections        []string // matched question keys (順序は cfg.Questions 順)
	UnknownSections []string // cfg.Questions に対応しなかった heading label (警告用)
	Reason          string   // skip/fail の理由
}

func (c *migrateCmd) Run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	storage, err := NewStorage(cfg)
	if err != nil {
		return err
	}
	defer storage.Close()

	dates, err := c.findTargets(storage)
	if err != nil {
		return err
	}

	results := make([]migrationResult, 0, len(dates))
	for _, d := range dates {
		results = append(results, c.processOne(storage, cfg, d))
	}

	return c.emitReport(os.Stdout, os.Stderr, results)
}

// findTargets は migrate 候補となる日付 slice を返す。
// --date 指定があれば単一日付のみ、無ければ ListReports で全 .md を拾う。
// .yaml の有無判定は processOne 側で行う (findTargets は対象集合の決定だけに集中)。
func (c *migrateCmd) findTargets(storage *Storage) ([]time.Time, error) {
	if c.Date != "" {
		d, err := time.Parse("2006-01-02", c.Date)
		if err != nil {
			return nil, fmt.Errorf("invalid date format: %w", err)
		}
		return []time.Time{d}, nil
	}

	rels, err := storage.ListReports()
	if err != nil {
		return nil, err
	}

	dates := make([]time.Time, 0, len(rels))
	for _, rel := range rels {
		d, err := parseRelReportDate(rel)
		if err != nil {
			continue
		}
		dates = append(dates, d)
	}
	// ListReports は新しい順 sort 済みだが、migrate は古い順に処理した方が
	// summary を見ながら追従しやすいため昇順に並び替える。
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	return dates, nil
}

// parseRelReportDate は ListReports が返す rel path ("2024/06/15.md") から date を切り出す。
// filesystem 以外の backend では rel フォーマットが異なる可能性があるため、parse 失敗は skip 扱いとする。
func parseRelReportDate(rel string) (time.Time, error) {
	// filepath.ToSlash は呼び出し側で済んでいない可能性があるため、ここで正規化する。
	norm := strings.ReplaceAll(rel, "\\", "/")
	return time.Parse("2006/01/02.md", norm)
}

// processOne は 1 日報を migrate する。
// 既存 .yaml がある: skip / .md が無い or 0 section: skip / parse 成功: apply モードでのみ書き込み。
func (c *migrateCmd) processOne(storage *Storage, cfg *Config, date time.Time) migrationResult {
	res := migrationResult{Date: date}

	// 既に .yaml があれば冪等性のためスキップ (上書きしない)。
	// fs.ErrNotExist 以外のエラーは「壊れた .yaml」等の可能性があるため fail で集約する。
	if _, err := storage.LoadReportStruct(date); err == nil {
		res.Action = migrateActionSkip
		res.Reason = "already has .yaml"
		return res
	} else if !errors.Is(err, fs.ErrNotExist) {
		res.Action = migrateActionFail
		res.Reason = fmt.Sprintf("read existing yaml: %v", err)
		return res
	}

	content, err := storage.LoadReport(date)
	if err != nil {
		res.Action = migrateActionFail
		res.Reason = fmt.Sprintf("read legacy md: %v", err)
		return res
	}

	sections := parseLegacyMarkdown(content)
	if len(sections) == 0 {
		res.Action = migrateActionSkip
		res.Reason = "no recognizable sections"
		return res
	}

	r := &report.Report{
		SchemaVersion: report.SupportedSchemaVersion,
		Date:          date,
		Fields:        make(map[string]report.FieldValue, len(cfg.Questions)),
	}

	matchedLabels := make(map[string]struct{}, len(cfg.Questions))
	for _, q := range cfg.Questions {
		body, ok := sections[q.Label]
		if !ok {
			continue
		}
		matchedLabels[q.Label] = struct{}{}
		switch q.Type {
		case "", "text":
			r.Fields[q.Key] = report.FieldValue{Type: report.FieldTypeText, Body: body}
		case "task_list":
			r.Fields[q.Key] = report.FieldValue{Type: report.FieldTypeTaskList, Tasks: parseLegacyTaskBullets(body)}
		default:
			// config load 時に reject 済みのはずだが、defensive に fail へ落とす。
			res.Action = migrateActionFail
			res.Reason = fmt.Sprintf("unsupported question type %q (key=%s)", q.Type, q.Key)
			return res
		}
		res.Sections = append(res.Sections, q.Key)
	}

	for label := range sections {
		if _, ok := matchedLabels[label]; !ok {
			res.UnknownSections = append(res.UnknownSections, label)
		}
	}
	sort.Strings(res.UnknownSections)

	if len(res.Sections) == 0 {
		res.Action = migrateActionSkip
		res.Reason = "no sections matched configured questions"
		return res
	}

	if c.Apply {
		if err := storage.SaveReportStruct(r); err != nil {
			res.Action = migrateActionFail
			res.Reason = fmt.Sprintf("write yaml: %v", err)
			return res
		}
	}
	res.Action = migrateActionMigrate
	return res
}

// parseLegacyMarkdown は `## ` で始まる heading で section 分割し、Label → body の map を返す。
// body は heading 直下から次の `## ` (もしくは EOF) までの中身を、前後の空行を trim した形で返す。
// CRLF も `\n` として扱うため事前に正規化する。
func parseLegacyMarkdown(content string) map[string]string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	sections := make(map[string]string)
	currentLabel := ""
	var currentBody []string

	flush := func() {
		if currentLabel == "" {
			return
		}
		body := strings.Join(currentBody, "\n")
		body = strings.Trim(body, "\n")
		sections[currentLabel] = body
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			flush()
			currentLabel = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			currentBody = currentBody[:0]
			continue
		}
		if currentLabel == "" {
			// `# 日報 ...` を含む heading 前の行は無視する (1st-level title など)。
			continue
		}
		currentBody = append(currentBody, line)
	}
	flush()

	return sections
}

// parseLegacyTaskBullets は section body から `- ` で始まる bullet 行のみ抽出して 1 タスクにする。
// 「- title (time) outcome」+ 任意の「  thoughts」継続行を解釈する。
// renderer.writeTask の逆変換になるよう極力対称に作るが、人手で書き換えられた markdown には深追いしない。
func parseLegacyTaskBullets(body string) []report.Task {
	tasks := []report.Task{}
	lines := strings.Split(body, "\n")
	var current *report.Task
	flush := func() {
		if current != nil {
			tasks = append(tasks, *current)
			current = nil
		}
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "- ") {
			flush()
			t := parseTaskHeadline(strings.TrimPrefix(line, "- "))
			current = &t
			continue
		}
		// 2 スペースインデントの thoughts 継続行。直前の bullet が無ければ捨てる。
		if strings.HasPrefix(line, "  ") && current != nil {
			cont := strings.TrimPrefix(line, "  ")
			if current.Thoughts == "" {
				current.Thoughts = cont
			} else {
				current.Thoughts += "\n" + cont
			}
			continue
		}
		// それ以外 (空行・自由文) は task の区切りとして flush する。
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
	}
	flush()
	return tasks
}

// parseTaskHeadline は "title (time) outcome" 形式 1 行をパースする。
// time / outcome は省略可。renderer 側の writeTask が出すフォーマットに対応する。
func parseTaskHeadline(s string) report.Task {
	t := report.Task{}
	rest := s

	// "(...)" の time セクションを検出する。最初の "(" 〜 対応する ")" を取り出す。
	if openIdx := strings.Index(rest, "("); openIdx >= 0 {
		if closeIdx := strings.Index(rest[openIdx:], ")"); closeIdx > 0 {
			t.Title = strings.TrimSpace(rest[:openIdx])
			t.Time = strings.TrimSpace(rest[openIdx+1 : openIdx+closeIdx])
			tail := strings.TrimSpace(rest[openIdx+closeIdx+1:])
			if tail != "" {
				t.Outcome = tail
			}
			return t
		}
	}
	t.Title = strings.TrimSpace(rest)
	return t
}

// emitReport は results を text / json で出力する。
// json は CLAUDE.md の「LLM が結果を parse 可能」要件のため stdout に書く。
func (c *migrateCmd) emitReport(stdout, stderr io.Writer, results []migrationResult) error {
	switch c.Format {
	case "", "text":
		c.emitText(stdout, stderr, results)
		return nil
	case "json":
		return c.emitJSON(stdout, results)
	default:
		return fmt.Errorf("unsupported format %q (want text or json)", c.Format)
	}
}

// emitText は人間可読の per-line サマリと summary 行を stdout に書き出す。
// 警告 (UnknownSections) は stderr に出すことで「サマリ stdout、警告 stderr」を分離する。
func (c *migrateCmd) emitText(stdout, stderr io.Writer, results []migrationResult) {
	prefix := "[dry-run] "
	verbMigrate := "would migrate"
	verbSkip := "would skip"
	if c.Apply {
		prefix = ""
		verbMigrate = "migrated"
		verbSkip = "skipped"
	}

	var migrated, skipped, failed int
	for _, r := range results {
		dateStr := r.Date.Format("2006-01-02")
		switch r.Action {
		case migrateActionMigrate:
			migrated++
			fmt.Fprintf(stdout, "%s%s %s (%d sections -> %s)\n",
				prefix, verbMigrate, dateStr, len(r.Sections), strings.Join(r.Sections, "/"))
		case migrateActionSkip:
			skipped++
			fmt.Fprintf(stdout, "%s%s %s (%s)\n", prefix, verbSkip, dateStr, r.Reason)
		case migrateActionFail:
			failed++
			fmt.Fprintf(stdout, "%sfailed %s (%s)\n", prefix, dateStr, r.Reason)
		}
		for _, label := range r.UnknownSections {
			fmt.Fprintf(stderr, "warning: %s: section %q has no matching question\n", dateStr, label)
		}
	}
	fmt.Fprintf(stdout, "summary: %s %d / skip %d / fail %d (total %d)\n",
		summaryVerb(c.Apply), migrated, skipped, failed, len(results))
}

func summaryVerb(apply bool) string {
	if apply {
		return "migrated"
	}
	return "would migrate"
}

// migrationJSONResult は emitJSON 用の wire 表現。time.Time を ISO 日付文字列で出す。
type migrationJSONResult struct {
	Date            string   `json:"date"`
	Action          string   `json:"action"`
	Sections        []string `json:"sections,omitempty"`
	UnknownSections []string `json:"unknown_sections,omitempty"`
	Reason          string   `json:"reason,omitempty"`
}

type migrationJSONReport struct {
	Mode    string                `json:"mode"`
	Results []migrationJSONResult `json:"results"`
	Summary migrationJSONSummary  `json:"summary"`
}

type migrationJSONSummary struct {
	Migrated int `json:"migrated"`
	Skipped  int `json:"skipped"`
	Failed   int `json:"failed"`
	Total    int `json:"total"`
}

func (c *migrateCmd) emitJSON(stdout io.Writer, results []migrationResult) error {
	mode := "dry-run"
	if c.Apply {
		mode = "apply"
	}
	out := migrationJSONReport{
		Mode:    mode,
		Results: make([]migrationJSONResult, 0, len(results)),
	}
	for _, r := range results {
		action := string(r.Action)
		if c.Apply && r.Action == migrateActionMigrate {
			// apply モードでは past tense (migrated) を返し、消費側が dry-run と区別できるようにする。
			action = "migrated"
		}
		jr := migrationJSONResult{
			Date:            r.Date.Format("2006-01-02"),
			Action:          action,
			Sections:        r.Sections,
			UnknownSections: r.UnknownSections,
			Reason:          r.Reason,
		}
		out.Results = append(out.Results, jr)

		switch r.Action {
		case migrateActionMigrate:
			out.Summary.Migrated++
		case migrateActionSkip:
			out.Summary.Skipped++
		case migrateActionFail:
			out.Summary.Failed++
		}
	}
	out.Summary.Total = len(results)

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
