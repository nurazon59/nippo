package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
)

type tuiCmd struct{}

func (c *tuiCmd) Run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	storage, err := NewStorage(cfg)
	if err != nil {
		return err
	}
	defer storage.Close()

	model := newTUIModel(storage, cfg.Questions)
	if err := model.reload(""); err != nil {
		return err
	}
	_, err = tea.NewProgram(model).Run()
	return err
}

// tuiReportStore は既存の Storage とテスト用ストアの共通インターフェース。
type tuiReportStore interface {
	ListReports() ([]string, error)
	LoadReportMarkdown(date time.Time, questions []QuestionConfig) (string, error)
}

type tuiEntry struct {
	date    time.Time
	content string
}

type tuiModel struct {
	store      tuiReportStore
	questions  []QuestionConfig
	entries    []tuiEntry
	filtered   []int
	cursor     int
	listTop    int
	previewTop int
	width      int
	height     int
	query      string
	searching  bool
	preview    bool
	help       bool
	status     string
}

type tuiEditorDone struct {
	date string
	err  error
}

func newTUIModel(store tuiReportStore, questions []QuestionConfig) tuiModel {
	return tuiModel{store: store, questions: questions, width: 90, height: 24}
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tuiEditorDone:
		if err := m.reload(msg.date); err != nil {
			m.status = "更新に失敗: " + err.Error()
		} else if msg.err != nil {
			m.status = "編集終了: " + msg.err.Error()
		} else {
			m.status = "更新しました"
		}
	case tea.KeyPressMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.help {
			if key == "?" || key == "esc" || key == "q" {
				m.help = false
			}
			return m, nil
		}
		if m.searching {
			switch key {
			case "esc":
				m.searching = false
				m.query = ""
				m.refilter("")
			case "enter":
				m.searching = false
			case "backspace", "ctrl+h":
				chars := []rune(m.query)
				if len(chars) > 0 {
					m.query = string(chars[:len(chars)-1])
				}
				m.refilter("")
			case "ctrl+u":
				m.query = ""
				m.refilter("")
			default:
				if msg.Text != "" && !strings.ContainsAny(msg.Text, "\r\n\x1b") {
					m.query += msg.Text
					m.refilter("")
				}
			}
			return m, nil
		}
		switch key {
		case "q":
			return m, tea.Quit
		case "?":
			m.help = true
		case "/":
			m.searching = true
			m.status = ""
		case "esc":
			m.query = ""
			m.refilter("")
		case "tab":
			m.preview = !m.preview
		case "j", "down":
			if m.preview {
				m.scroll(1)
			} else {
				m.move(1)
			}
		case "k", "up":
			if m.preview {
				m.scroll(-1)
			} else {
				m.move(-1)
			}
		case "g":
			if !m.preview {
				m.move(-len(m.filtered))
			}
		case "G":
			if !m.preview {
				m.move(len(m.filtered))
			}
		case "pgdown", "ctrl+d":
			m.scroll(max(1, (m.height-3)/2))
		case "pgup", "ctrl+u":
			m.scroll(-max(1, (m.height-3)/2))
		case "r":
			if err := m.reload(m.selectedDate()); err != nil {
				m.status = err.Error()
			} else {
				m.status = "再読込しました"
			}
		case "n":
			m.query = ""
			m.refilter("")
			return m, m.edit(time.Now())
		case "e", "enter":
			if entry, ok := m.selectedEntry(); ok {
				return m, m.edit(entry.date)
			}
			m.status = "日報がありません。n で作成できます"
		}
	}
	return m, nil
}

// 既存の nippo edit を別プロセスで起動する。YAML 正本と sidecar の整合性を保つ。
func (m tuiModel) edit(date time.Time) tea.Cmd {
	args := []string{}
	if CLI.Config != "" {
		args = append(args, "--config", CLI.Config)
	}
	args = append(args, "edit", date.Format("2006-01-02"))
	return tea.ExecProcess(exec.Command(os.Args[0], args...), func(err error) tea.Msg {
		return tuiEditorDone{date: date.Format("2006-01-02"), err: err}
	})
}

func datesFromPaths(paths []string) []time.Time {
	seen := make(map[string]bool, len(paths))
	var dates []time.Time
	for _, path := range paths {
		date, err := time.Parse("2006/01/02.md", filepath.ToSlash(path))
		if err != nil {
			continue
		}
		key := date.Format("2006-01-02")
		if seen[key] {
			continue
		}
		seen[key] = true
		dates = append(dates, date)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].After(dates[j]) })
	return dates
}

func (m *tuiModel) reload(selected string) error {
	paths, err := m.store.ListReports()
	if err != nil {
		return err
	}
	dates := datesFromPaths(paths)
	entries := make([]tuiEntry, 0, len(dates))
	for _, date := range dates {
		content, err := m.store.LoadReportMarkdown(date, m.questions)
		if err != nil {
			return fmt.Errorf("%s: %w", date.Format("2006-01-02"), err)
		}
		entries = append(entries, tuiEntry{date: date, content: content})
	}
	m.entries = entries
	m.refilter(selected)
	return nil
}

func (m *tuiModel) refilter(selected string) {
	m.filtered = m.filtered[:0]
	q := strings.ToLower(m.query)
	for i, entry := range m.entries {
		if strings.Contains(strings.ToLower(entry.date.Format("2006-01-02")+"\n"+entry.content), q) {
			m.filtered = append(m.filtered, i)
		}
	}
	m.cursor, m.listTop, m.previewTop = 0, 0, 0
	for i, id := range m.filtered {
		if m.entries[id].date.Format("2006-01-02") == selected {
			m.cursor = i
			break
		}
	}
	m.ensureVisible()
}

func (m tuiModel) selectedEntry() (tuiEntry, bool) {
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return tuiEntry{}, false
	}
	return m.entries[m.filtered[m.cursor]], true
}

func (m tuiModel) selectedDate() string {
	if entry, ok := m.selectedEntry(); ok {
		return entry.date.Format("2006-01-02")
	}
	return ""
}

func (m *tuiModel) move(n int) {
	if len(m.filtered) == 0 {
		return
	}
	m.cursor = max(0, min(len(m.filtered)-1, m.cursor+n))
	m.previewTop = 0
	m.ensureVisible()
}

func (m *tuiModel) ensureVisible() {
	page := max(1, m.height-4)
	if m.cursor < m.listTop {
		m.listTop = m.cursor
	}
	if m.cursor >= m.listTop+page {
		m.listTop = m.cursor-page+1
	}
}

func (m *tuiModel) scroll(n int) {
	entry, ok := m.selectedEntry()
	if !ok {
		return
	}
	count := len(strings.Split(entry.content, "\n"))
	m.previewTop = max(0, min(max(0, count-max(1, m.height-4)), m.previewTop+n))
}

func (m tuiModel) View() tea.View {
	var output string
	switch {
	case m.width < 32 || m.height < 8:
		output = "nippo: terminal is too small (32x8)"
	case m.help:
		output = m.helpView()
	default:
		output = m.mainView()
	}
	view := tea.NewView(output)
	view.AltScreen = true
	return view
}

func (m tuiModel) mainView() string {
	h := m.height-3
	leftWidth := min(28, max(20, m.width/3))
	rightWidth := m.width-leftWidth-3
	lines := []string{fitTUI(fmt.Sprintf(" nippo  %d/%d 件", len(m.filtered), len(m.entries)), m.width)}
	list := m.listLines(h, leftWidth)
	preview := m.previewLines(h, max(1, rightWidth))
	for i := 0; i < h; i++ {
		if m.width < 72 {
			if m.preview {
				lines = append(lines, fitTUI(preview[i], m.width))
			} else {
				lines = append(lines, fitTUI(list[i], m.width))
			}
		} else {
			lines = append(lines, fitTUI(list[i], leftWidth)+" │ "+fitTUI(preview[i], rightWidth))
		}
	}
	status := " "+m.status
	if m.searching {
		status = " 検索: /"+m.query+"_"
	} else if m.query != "" && m.status == "" {
		status = " フィルタ: /"+m.query
	}
	lines = append(lines, fitTUI(status, m.width))
	lines = append(lines, fitTUI(" j/k 移動  tab 切替  n 新規  e 編集  / 検索  r 再読込  ? ヘルプ  q 終了", m.width))
	return strings.Join(lines, "\n")
}

func (m tuiModel) listLines(height, width int) []string {
	lines := make([]string, height)
	lines[0] = "日報一覧"
	if len(m.filtered) == 0 {
		lines[1] = "日報がありません"
		return lines
	}
	for row := 1; row < height; row++ {
		index := m.listTop+row-1
		if index >= len(m.filtered) {
			break
		}
		entry := m.entries[m.filtered[index]]
		marker := "  "
		if index == m.cursor {
			marker = "> "
		}
		lines[row] = fitTUI(marker+entry.date.Format("2006-01-02 Mon"), width)
	}
	return lines
}

func (m tuiModel) previewLines(height, width int) []string {
	lines := make([]string, height)
	entry, ok := m.selectedEntry()
	if !ok {
		lines[0] = "プレビュー"
		return lines
	}
	lines[0] = entry.date.Format("2006-01-02") + "  プレビュー"
	body := strings.Split(strings.ReplaceAll(entry.content, "\r\n", "\n"), "\n")
	for row := 1; row < height; row++ {
		index := m.previewTop+row-1
		if index >= len(body) {
			break
		}
		lines[row] = fitTUI(body[index], width)
	}
	return lines
}

func (m tuiModel) helpView() string {
	return strings.Join([]string{
		"nippo TUI",
		"",
		" j/k, ↑/↓    日付移動・プレビュースクロール",
		" g/G         先頭・末尾の日付",
		" tab         一覧・プレビュー切替",
		" ctrl+d/u    プレビュースクロール",
		" n           今日の日報を作成・編集",
		" e, enter    選択中の日報を編集",
		" /           本文・日付検索 (enter で確定)",
		" esc         検索解除",
		" r           再読込",
		" ?           ヘルプを閉じる",
		" q           終了",
	}, "\n")
}

// 日本語の全角文字を考慮した表示幅。制御文字は画面に表示しない。
func fitTUI(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	var b strings.Builder
	size := 0
	for _, r := range s {
		if unicode.IsControl(r) {
			continue
		}
		n := 1
		if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) ||
			(r >= 0xFF01 && r <= 0xFF60) {
			n = 2
		}
		if size+n > limit {
			break
		}
		b.WriteRune(r)
		size += n
	}
	b.WriteString(strings.Repeat(" ", limit-size))
	return b.String()
}
