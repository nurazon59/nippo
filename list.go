package main

import "fmt"

type listCmd struct{}

func (c *listCmd) Run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	storage, err := NewStorage(cfg.StorageDir)
	if err != nil {
		return err
	}
	reports, err := storage.ListReports()
	if err != nil {
		return err
	}

	if len(reports) == 0 {
		fmt.Println("No saved reports.")
		return nil
	}

	for _, r := range reports {
		fmt.Println(r)
	}
	return nil
}
