package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
	"github.com/AlecAivazis/survey/v2/terminal"
	shellquote "github.com/kballard/go-shellquote"
	"golang.org/x/term"
)

var defaultEditor = "vim"

func init() {
	if e := os.Getenv("EDITOR"); e != "" {
		defaultEditor = e
	}
}

type formField struct {
	name    string
	message string
	setter  func(*Report, string)
}

var formFields = []formField{
	{name: "Done", message: "やった", setter: func(r *Report, v string) { r.Done = v }},
	{name: "Todo", message: "やる", setter: func(r *Report, v string) { r.Todo = v }},
	{name: "Thoughts", message: "所感", setter: func(r *Report, v string) { r.Thoughts = v }},
}

type editorPrompt struct {
	message       string
	defaultValue  string
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
			result, err := launchEditor(editorCmd, e.defaultValue)
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
			fmt.Fprint(os.Stderr, "\r\n")
			return e.defaultValue, nil
		}

		if b[0] == terminal.KeyInterrupt {
			return "", terminal.InterruptErr
		}
	}

	return "", nil
}

func launchEditor(editorCmd, defaultContent string) (string, error) {
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

	report := &Report{Date: date}
	if err := c.runForm(report); err != nil {
		return err
	}

	content := GenerateMarkdown(report)
	fmt.Print(content)

	storage, err := NewStorage(cfg.StorageDir)
	if err != nil {
		return err
	}
	return storage.SaveReport(content, date)
}

func (c *generateCmd) runForm(report *Report) error {
	for _, field := range formFields {
		prompt := &editorPrompt{
			message:      field.message,
			defaultValue: "",
		}
		value, err := prompt.prompt()
		if err != nil {
			if err == terminal.InterruptErr {
				return err
			}
			return err
		}
		field.setter(report, value)
	}

	options := []string{"Submit", "Cancel"}
	var next string
	selectPrompt := &survey.Select{
		Message: "What's next?",
		Options: options,
		Default: options[0],
	}
	err := survey.AskOne(selectPrompt, &next, survey.WithIcons(func(icons *survey.IconSet) {
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
