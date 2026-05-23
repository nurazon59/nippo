package main

import (
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/nurazon59/nippo/renderer"
)

// editCmd は指定 date の .yaml を再ロードし runForm を再実行する「フォーム再開モデル」。
// 旧 vim 直接編集 (.md を上書き) は canonical (.yaml) と乖離するため Step 8 で廃止。
// 各質問は既存値を default に流し込んだ状態で再表示され、enter で既存値維持・(e) でエディタ再起動になる。
type editCmd struct {
	Date string `arg:"" help:"Target date (YYYY-MM-DD)."`
}

func (c *editCmd) Run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	date, err := time.Parse("2006-01-02", c.Date)
	if err != nil {
		return fmt.Errorf("invalid date format: %w", err)
	}

	storage, err := NewStorage(cfg)
	if err != nil {
		return err
	}
	defer storage.Close()

	// 既存 .yaml が無ければ新規 Report として扱い、generate と同等のフォームに合流させる。
	// LoadReportStruct のその他エラー (parse 失敗等) は silent fallback せず即返す。
	existing, err := storage.LoadReportStruct(date)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		existing = nil
	}

	r, err := runForm(storage, cfg, date, existing)
	if err != nil {
		return err
	}

	if err := storage.SaveReportStruct(r); err != nil {
		return err
	}

	md := renderer.Markdown(r, questionsToRendererQuestions(cfg.Questions))
	if err := storage.WriteSidecar(date, ".md", []byte(md)); err != nil {
		return err
	}

	fmt.Println("Report updated successfully.")
	return nil
}
