package report

import (
	"fmt"
	"time"

	"github.com/goccy/go-yaml"
)

const dateLayout = "2006-01-02"

const SupportedSchemaVersion = 1

const (
	FieldTypeText     = "text"
	FieldTypeTaskList = "task_list"
)

type Report struct {
	SchemaVersion int
	Date          time.Time
	Fields        map[string]FieldValue
}

type FieldValue struct {
	Type  string
	Body  string
	Tasks []Task
}

type Task struct {
	Title    string `yaml:"title"`
	Time     string `yaml:"time,omitempty"`
	Outcome  string `yaml:"outcome,omitempty"`
	Thoughts string `yaml:"thoughts,omitempty"`
}

type reportWire struct {
	SchemaVersion *int                  `yaml:"schema_version"`
	Date          *string               `yaml:"date"`
	Fields        map[string]FieldValue `yaml:"fields"`
}

func (r Report) MarshalYAML() (interface{}, error) {
	if r.SchemaVersion != SupportedSchemaVersion {
		return nil, fmt.Errorf("report: unsupported schema_version %d (want %d)", r.SchemaVersion, SupportedSchemaVersion)
	}
	if r.Date.IsZero() {
		return nil, fmt.Errorf("report: date is required")
	}
	ver := r.SchemaVersion
	d := r.Date.Format(dateLayout)
	fields := r.Fields
	if fields == nil {
		fields = map[string]FieldValue{}
	}
	return reportWire{
		SchemaVersion: &ver,
		Date:          &d,
		Fields:        fields,
	}, nil
}

func (r *Report) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var wire reportWire
	if err := unmarshal(&wire); err != nil {
		return err
	}
	if wire.SchemaVersion == nil {
		return fmt.Errorf("report: schema_version is required")
	}
	if *wire.SchemaVersion != SupportedSchemaVersion {
		return fmt.Errorf("report: unsupported schema_version %d (want %d)", *wire.SchemaVersion, SupportedSchemaVersion)
	}
	if wire.Date == nil {
		return fmt.Errorf("report: date is required")
	}
	d, err := time.Parse(dateLayout, *wire.Date)
	if err != nil {
		return fmt.Errorf("report: invalid date %q (want %s): %w", *wire.Date, dateLayout, err)
	}
	r.SchemaVersion = *wire.SchemaVersion
	r.Date = d
	if wire.Fields == nil {
		r.Fields = map[string]FieldValue{}
	} else {
		r.Fields = wire.Fields
	}
	return nil
}

type fieldValueWire struct {
	Type  string  `yaml:"type"`
	Body  *string `yaml:"body,omitempty"`
	Tasks *[]Task `yaml:"tasks,omitempty"`
}

func (v FieldValue) MarshalYAML() (interface{}, error) {
	switch v.Type {
	case FieldTypeText:
		if v.Tasks != nil {
			return nil, fmt.Errorf("report: field type=text must not contain tasks")
		}
		b := v.Body
		return fieldValueWire{Type: v.Type, Body: &b}, nil
	case FieldTypeTaskList:
		if v.Body != "" {
			return nil, fmt.Errorf("report: field type=task_list must not contain body")
		}
		tasks := v.Tasks
		if tasks == nil {
			tasks = []Task{}
		}
		return fieldValueWire{Type: v.Type, Tasks: &tasks}, nil
	default:
		return nil, fmt.Errorf("report: unsupported field type %q", v.Type)
	}
}

func (v *FieldValue) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var wire fieldValueWire
	if err := unmarshal(&wire); err != nil {
		return err
	}
	switch wire.Type {
	case "":
		return fmt.Errorf("report: field type is required")
	case FieldTypeText:
		if wire.Tasks != nil {
			return fmt.Errorf("report: field type=text must not contain tasks")
		}
		v.Type = FieldTypeText
		if wire.Body != nil {
			v.Body = *wire.Body
		}
		v.Tasks = nil
	case FieldTypeTaskList:
		if wire.Body != nil {
			return fmt.Errorf("report: field type=task_list must not contain body")
		}
		v.Type = FieldTypeTaskList
		v.Body = ""
		if wire.Tasks != nil {
			v.Tasks = *wire.Tasks
		} else {
			v.Tasks = []Task{}
		}
	default:
		return fmt.Errorf("report: unsupported field type %q", wire.Type)
	}
	return nil
}

var (
	_ yaml.InterfaceMarshaler   = Report{}
	_ yaml.InterfaceUnmarshaler = (*Report)(nil)
	_ yaml.InterfaceMarshaler   = FieldValue{}
	_ yaml.InterfaceUnmarshaler = (*FieldValue)(nil)
)
