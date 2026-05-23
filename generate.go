package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
	"github.com/AlecAivazis/survey/v2/terminal"
	shellquote "github.com/kballard/go-shellquote"
	"golang.org/x/term"

	"github.com/nurazon59/nippo/renderer"
	"github.com/nurazon59/nippo/report"
)

var defaultEditor = "vim"

func init() {
	if e := os.Getenv("EDITOR"); e != "" {
		defaultEditor = e
	}
}

type editorPrompt struct {
	message       string
	defaultValue  string
	editorContent string
	editorCommand string
}

func (e *editorPrompt) prompt() (string, error) {
	editorCmd := e.editorCommand
	if editorCmd == "" {
		editorCmd = defaultEditor
	}
	editorName := filepath.Base(editorCmd)
	if args, err := shellquote.Split(editorCmd); err == nil {
		editorName = args[0]
	}

	config := &survey.PromptConfig{
		Icons: survey.IconSet{
			Question: survey.Icon{Text: "?", Format: "green+hb"},
		},
		HelpInput: "?",
	}

	tmpl := `{{color .Config.Icons.Question.Format}}{{.Config.Icons.Question.Text}}{{color "reset"}} {{color "default+hb"}}{{.Message}}{{color "reset"}}{{if .Default}} {{color "white"}}({{.Default}}){{color "reset"}}{{end}} {{color "cyan"}}[(e) to launch {{.EditorName}}, enter to skip]{{color "reset"}} `

	data := struct {
		Message    string
		Default    string
		EditorName string
		Config     *survey.PromptConfig
	}{
		Message:    e.message,
		Default:    e.defaultValue,
		EditorName: editorName,
		Config:     config,
	}

	out, _, err := core.RunTemplate(tmpl, data)
	if err != nil {
		return "", err
	}
	fmt.Fprint(os.Stderr, out)

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	restored := false
	defer func() {
		if !restored {
			term.Restore(fd, oldState)
		}
	}()

	for {
		b := make([]byte, 1)
		n, err := os.Stdin.Read(b)
		if err != nil || n == 0 {
			break
		}

		if b[0] == 'e' {
			term.Restore(fd, oldState)
			restored = true
			fmt.Fprint(os.Stderr, "\r\n")
			result, err := launchEditor(editorCmd, e.editorContent)
			if err != nil {
				return "", err
			}

			ansiOut, _, err := core.RunTemplate(
				`{{color .Config.Icons.Question.Format}}{{.Config.Icons.Question.Text}}{{color "reset"}} {{color "default+hb"}}{{.Message}}{{color "reset"}} {{color "cyan"}}{{.Answer}}{{color "reset"}}
`,
				struct {
					Message string
					Answer  string
					Config  *survey.PromptConfig
				}{
					Message: e.message,
					Answer:  result,
					Config:  config,
				},
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "? %s %s\n", e.message, result)
			} else {
				fmt.Fprint(os.Stderr, ansiOut)
			}
			return result, nil
		}

		if b[0] == '\r' || b[0] == '\n' {
			fmt.Fprint(os.Stderr, "\r\033[K")
			return e.defaultValue, nil
		}

		if b[0] == terminal.KeyInterrupt {
			return "", terminal.InterruptErr
		}
	}

	return "", nil
}

func launchEditor(editorCmd, defaultContent string) (string, error) {
	if editorCmd == "" {
		editorCmd = defaultEditor
	}
	tmp, err := os.CreateTemp("", "nippo-*.md")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(defaultContent); err != nil {
		return "", err
	}
	tmp.Close()

	args, err := shellquote.Split(editorCmd)
	if err != nil {
		return "", err
	}
	args = append(args, tmp.Name())

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	bytes, err := os.ReadFile(tmp.Name())
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

type generateCmd struct {
	Date string `help:"Target date (YYYY-MM-DD)."`
}

func (c *generateCmd) Run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	date := time.Now()
	if c.Date != "" {
		date, err = time.Parse("2006-01-02", c.Date)
		if err != nil {
			return fmt.Errorf("invalid date format: %w", err)
		}
	}

	storage, err := NewStorage(cfg)
	if err != nil {
		return err
	}
	defer storage.Close()

	r := &report.Report{
		SchemaVersion: report.SupportedSchemaVersion,
		Date:          date,
		Fields:        make(map[string]report.FieldValue, len(cfg.Questions)),
	}
	if err := c.runForm(storage, cfg, r); err != nil {
		return err
	}

	if err := storage.SaveReportStruct(r); err != nil {
		return err
	}

	md := renderer.Markdown(r, questionsToRendererQuestions(cfg.Questions))
	if err := storage.WriteSidecar(date, ".md", []byte(md)); err != nil {
		return err
	}

	fmt.Print(md)
	return nil
}

// runForm はフォームを回し、回答を r.Fields に直接詰める。
// Step 7 で QuestionConfig.Type に応じて text / task_list 分岐をディスパッチする。
// 受け取った r は呼び出し元で SaveReportStruct / Markdown 描画に使い回す前提。
func (c *generateCmd) runForm(storage *Storage, cfg *Config, r *report.Report) error {
	presets, err := buildReferencePresets(storage, r.Date, cfg.Questions)
	if err != nil {
		return err
	}

	hookOut := RunHooks(context.Background(), cfg.Hooks, r.Date)
	presets = mergePresets(presets, hookOut)

	for _, q := range cfg.Questions {
		switch q.Type {
		case "", "text":
			value, err := promptText(q, presets[q.Key], r.Fields)
			if err != nil {
				return err
			}
			applyTextAnswer(r, q, value)
		case "task_list":
			tasks, err := promptTaskList(q)
			if err != nil {
				return err
			}
			applyTaskListAnswer(r, q, tasks)
		default:
			// config load 時に reject されるはずだが、defensive に明示エラー
			return fmt.Errorf("unsupported question type %q (key=%s)", q.Type, q.Key)
		}
	}

	options := []string{"Submit", "Cancel"}
	var next string
	selectPrompt := &survey.Select{
		Message: "What's next?",
		Options: options,
		Default: options[0],
	}
	err = survey.AskOne(selectPrompt, &next, survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = "?"
		icons.Question.Format = "green+hb"
		icons.SelectFocus.Text = ">"
		icons.SelectFocus.Format = "green"
	}))
	if err != nil {
		if err == terminal.InterruptErr {
			return err
		}
		return err
	}

	if next == "Cancel" {
		return terminal.InterruptErr
	}

	return nil
}

// applyTextAnswer は text 型の回答を r.Fields に書き込む。
// Type は canonical な report.FieldTypeText に固定 (空文字 ("") のままにしない)。
func applyTextAnswer(r *report.Report, q QuestionConfig, value string) {
	r.Fields[q.Key] = report.FieldValue{Type: report.FieldTypeText, Body: value}
}

// applyTaskListAnswer は task_list 型の回答を r.Fields に書き込む。
// tasks が nil の場合も []report.Task{} に正規化することで、YAML 上 `tasks: []` が出るようにする。
func applyTaskListAnswer(r *report.Report, q QuestionConfig, tasks []report.Task) {
	if tasks == nil {
		tasks = []report.Task{}
	}
	r.Fields[q.Key] = report.FieldValue{Type: report.FieldTypeTaskList, Tasks: tasks}
}

// promptText は text 型の 1 質問を editor prompt で取得する。
// 同日 reference / hook preset を editor 初期値に合成して返す。
func promptText(q QuestionConfig, refPreset string, answered map[string]report.FieldValue) (string, error) {
	editorContent := refPreset
	if sameDay := buildSameDayPreset(answered, q); sameDay != "" {
		if editorContent != "" {
			editorContent = editorContent + "\n\n" + sameDay
		} else {
			editorContent = sameDay
		}
	}
	prompt := &editorPrompt{
		message:       q.Label,
		defaultValue:  "",
		editorContent: editorContent,
	}
	return prompt.prompt()
}

// promptTaskList は task_list 型の連続入力フォームを回す。
// ユーザーがタスクを 1 件ずつ追加していき "Done" を選んだ時点で確定する。
// title 空のタスクは silent skip しない (明示的に Done を選ばせる方が UI 心理学的に混乱が少ない)。
func promptTaskList(q QuestionConfig) ([]report.Task, error) {
	var tasks []report.Task
	for {
		t, err := promptSingleTask(q, len(tasks)+1)
		if err != nil {
			return nil, err
		}
		// title 空はスキップ扱い (誤って Add another を選んだ場合の救済)
		if strings.TrimSpace(t.Title) != "" {
			tasks = append(tasks, t)
		}

		next, err := askAddAnother(q.Label, len(tasks))
		if err != nil {
			return nil, err
		}
		if !next {
			return tasks, nil
		}
	}
}

// promptSingleTask は task_list の 1 要素を順番に尋ねる。
// title が必須、time / outcome / thoughts は任意。thoughts のみ Multiline で複数行を許容する。
func promptSingleTask(q QuestionConfig, index int) (report.Task, error) {
	prompts := []*survey.Question{
		{
			Name:   "title",
			Prompt: &survey.Input{Message: fmt.Sprintf("%s #%d title", q.Label, index)},
		},
		{
			Name:   "time",
			Prompt: &survey.Input{Message: fmt.Sprintf("%s #%d time (e.g. 30m, 1h)", q.Label, index)},
		},
		{
			Name:   "outcome",
			Prompt: &survey.Input{Message: fmt.Sprintf("%s #%d outcome", q.Label, index)},
		},
		{
			Name:   "thoughts",
			Prompt: &survey.Multiline{Message: fmt.Sprintf("%s #%d thoughts", q.Label, index)},
		},
	}
	answers := struct {
		Title    string
		Time     string
		Outcome  string
		Thoughts string
	}{}
	if err := survey.Ask(prompts, &answers, survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = "?"
		icons.Question.Format = "green+hb"
	})); err != nil {
		return report.Task{}, err
	}
	return report.Task{
		Title:    strings.TrimSpace(answers.Title),
		Time:     strings.TrimSpace(answers.Time),
		Outcome:  strings.TrimSpace(answers.Outcome),
		Thoughts: strings.TrimRight(answers.Thoughts, "\n"),
	}, nil
}

// askAddAnother は 1 タスク入力後に「もう 1 件追加するか終了するか」を二択で尋ねる。
// 明示的な選択肢を出すことで「空 enter で終了か継続か曖昧」になる UX 事故を避ける。
func askAddAnother(label string, count int) (bool, error) {
	options := []string{"Add another task", "Done"}
	var choice string
	prompt := &survey.Select{
		Message: fmt.Sprintf("%s: %d task(s) entered", label, count),
		Options: options,
		Default: options[0],
	}
	if err := survey.AskOne(prompt, &choice, survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = "?"
		icons.Question.Format = "green+hb"
		icons.SelectFocus.Text = ">"
		icons.SelectFocus.Format = "green"
	})); err != nil {
		return false, err
	}
	return choice == "Add another task", nil
}
