// Package report は nippo の構造化スキーマ v1 を定義する。
// Markdown 表示は renderer 層に任せ、永続化用フォーマット (YAML) を本 package で扱う。
package report

import (
	"fmt"
	"time"

	"github.com/goccy/go-yaml"
)

// dateLayout は YAML 上で扱う date の固定フォーマット。
// time.Time の RFC3339 既定を避けるため、Marshal/Unmarshal の双方で本定数を用いる。
const dateLayout = "2006-01-02"

// SupportedSchemaVersion は本 package が解釈できる schema_version。
// 将来 v2 を導入する際は migrate を別 step で実装する想定で、現状は厳格に 1 のみ許可する。
const SupportedSchemaVersion = 1

// FieldType は FieldValue.Type に入る列挙値。type 不明値は unmarshal で reject する。
const (
	FieldTypeText     = "text"
	FieldTypeTaskList = "task_list"
)

// Report は 1 日分の日報を表す構造化データ。
// Fields は質問キー → FieldValue の map で、順序は保存しない (renderer が QuestionConfig 順に並べる前提)。
type Report struct {
	SchemaVersion int
	Date          time.Time
	Fields        map[string]FieldValue
}

// FieldValue は質問への回答。Type に応じて Body もしくは Tasks のいずれかのみ意味を持つ。
// YAML 上では type に対応するフィールドだけがシリアライズされる。
type FieldValue struct {
	Type  string
	Body  string
	Tasks []Task
}

// Task は task_list 型フィールドの 1 要素。
// Time は Go の duration 文字列を想定するが、本 step では生文字列のまま保持する (検証は別 step)。
type Task struct {
	Title    string `yaml:"title"`
	Time     string `yaml:"time,omitempty"`
	Outcome  string `yaml:"outcome,omitempty"`
	Thoughts string `yaml:"thoughts,omitempty"`
}

// reportWire は YAML との中間表現。time.Time の独自フォーマットと
// schema_version の必須化を MarshalYAML/UnmarshalYAML 経由で制御する。
type reportWire struct {
	SchemaVersion *int                  `yaml:"schema_version"`
	Date          *string               `yaml:"date"`
	Fields        map[string]FieldValue `yaml:"fields"`
}

// MarshalYAML は Report を reportWire へ写してから既定エンコーダに委譲する。
// silent fallback 禁止: SchemaVersion=0 や zero Date は明示的にエラーで返す。
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

// UnmarshalYAML は schema_version と date の必須/形式チェックを行う。
// silent fallback はせず、欠落・形式違反は明示的にエラーで返す。
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

// fieldValueWire は FieldValue を YAML に書き出す際の中間表現。
// pointer 化することで「指定なし vs 空文字/空配列」を区別する。
type fieldValueWire struct {
	Type  string  `yaml:"type"`
	Body  *string `yaml:"body,omitempty"`
	Tasks *[]Task `yaml:"tasks,omitempty"`
}

// MarshalYAML は Type に応じて body / tasks のいずれかのみを出力する。
// task_list は tasks が空配列でも明示的に `tasks: []` を出して「0 件である」ことを示す。
func (v FieldValue) MarshalYAML() (interface{}, error) {
	switch v.Type {
	case FieldTypeText:
		b := v.Body
		return fieldValueWire{Type: v.Type, Body: &b}, nil
	case FieldTypeTaskList:
		tasks := v.Tasks
		if tasks == nil {
			tasks = []Task{}
		}
		return fieldValueWire{Type: v.Type, Tasks: &tasks}, nil
	default:
		return nil, fmt.Errorf("report: unsupported field type %q", v.Type)
	}
}

// UnmarshalYAML は type を見て対応するフィールドだけを採用する。
// type が text/task_list 以外なら silent fallback せずエラーで返す。
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

// 型シグネチャの担保 (interface 充足の compile-time check)。
var (
	_ yaml.InterfaceMarshaler   = Report{}
	_ yaml.InterfaceUnmarshaler = (*Report)(nil)
	_ yaml.InterfaceMarshaler   = FieldValue{}
	_ yaml.InterfaceUnmarshaler = (*FieldValue)(nil)
)
