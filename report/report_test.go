package report_test

import (
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nurazon59/nippo/report"
)

func TestUnmarshal(t *testing.T) {
	tests := map[string]struct {
		yaml          string
		want          *report.Report
		wantErr       bool
		wantErrSubstr string
	}{
		"text + task_list mixed": {
			yaml: `schema_version: 1
date: 2026-05-23
fields:
  done:
    type: task_list
    tasks:
      - title: "PR #47 review"
        time: 1h30m
        outcome: merged
        thoughts: refactor しやすかった
  todo:
    type: text
    body: |
      明日は issue 46 の続き
  thoughts:
    type: text
    body: 集中できた一日
`,
			want: &report.Report{
				SchemaVersion: 1,
				Date:          time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
				Fields: map[string]report.FieldValue{
					"done": {
						Type: "task_list",
						Tasks: []report.Task{
							{
								Title:    "PR #47 review",
								Time:     "1h30m",
								Outcome:  "merged",
								Thoughts: "refactor しやすかった",
							},
						},
					},
					"todo": {
						Type: "text",
						Body: "明日は issue 46 の続き\n",
					},
					"thoughts": {
						Type: "text",
						Body: "集中できた一日",
					},
				},
			},
		},
		"only text fields": {
			yaml: `schema_version: 1
date: 2026-01-02
fields:
  memo:
    type: text
    body: hello
`,
			want: &report.Report{
				SchemaVersion: 1,
				Date:          time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
				Fields: map[string]report.FieldValue{
					"memo": {Type: "text", Body: "hello"},
				},
			},
		},
		"empty fields": {
			yaml: `schema_version: 1
date: 2026-05-23
fields: {}
`,
			want: &report.Report{
				SchemaVersion: 1,
				Date:          time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
				Fields:        map[string]report.FieldValue{},
			},
		},
		"task with only title": {
			yaml: `schema_version: 1
date: 2026-05-23
fields:
  done:
    type: task_list
    tasks:
      - title: 単発タスク
`,
			want: &report.Report{
				SchemaVersion: 1,
				Date:          time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
				Fields: map[string]report.FieldValue{
					"done": {
						Type:  "task_list",
						Tasks: []report.Task{{Title: "単発タスク"}},
					},
				},
			},
		},
		"task_list with zero tasks": {
			yaml: `schema_version: 1
date: 2026-05-23
fields:
  done:
    type: task_list
    tasks: []
`,
			want: &report.Report{
				SchemaVersion: 1,
				Date:          time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
				Fields: map[string]report.FieldValue{
					"done": {Type: "task_list", Tasks: []report.Task{}},
				},
			},
		},
		"unknown type rejected": {
			yaml: `schema_version: 1
date: 2026-05-23
fields:
  done:
    type: bogus
    body: x
`,
			wantErr:       true,
			wantErrSubstr: "bogus",
		},
		"missing schema_version rejected": {
			yaml: `date: 2026-05-23
fields: {}
`,
			wantErr:       true,
			wantErrSubstr: "schema_version is required",
		},
		"unsupported schema_version rejected": {
			yaml: `schema_version: 99
date: 2026-05-23
fields: {}
`,
			wantErr:       true,
			wantErrSubstr: "unsupported schema_version",
		},
		"invalid date rejected": {
			yaml: `schema_version: 1
date: 2026/05/23
fields: {}
`,
			wantErr:       true,
			wantErrSubstr: "invalid date",
		},
		"missing field type rejected": {
			yaml: `schema_version: 1
date: 2026-05-23
fields:
  done:
    body: x
`,
			wantErr:       true,
			wantErrSubstr: "field type is required",
		},
		"text with tasks rejected": {
			yaml: `schema_version: 1
date: 2026-05-23
fields:
  done:
    type: text
    body: hi
    tasks: []
`,
			wantErr:       true,
			wantErrSubstr: "type=text must not contain tasks",
		},
		"task_list with body rejected": {
			yaml: `schema_version: 1
date: 2026-05-23
fields:
  done:
    type: task_list
    body: nope
    tasks: []
`,
			wantErr:       true,
			wantErrSubstr: "type=task_list must not contain body",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var got report.Report
			err := yaml.Unmarshal([]byte(tt.yaml), &got)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrSubstr != "" {
					assert.Contains(t, err.Error(), tt.wantErrSubstr)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.SchemaVersion, got.SchemaVersion)
			assert.True(t, tt.want.Date.Equal(got.Date), "date mismatch: want=%s got=%s", tt.want.Date, got.Date)
			assert.Equal(t, tt.want.Fields, got.Fields)
		})
	}
}

func TestRoundTrip(t *testing.T) {
	tests := map[string]*report.Report{
		"basic text only": {
			SchemaVersion: 1,
			Date:          time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
			Fields: map[string]report.FieldValue{
				"memo": {Type: "text", Body: "hello world"},
			},
		},
		"task_list with multiple tasks": {
			SchemaVersion: 1,
			Date:          time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
			Fields: map[string]report.FieldValue{
				"done": {
					Type: "task_list",
					Tasks: []report.Task{
						{Title: "A", Time: "1h", Outcome: "ok", Thoughts: "良かった"},
						{Title: "B"},
						{Title: "C", Time: "30m"},
					},
				},
			},
		},
		"task_list with zero tasks": {
			SchemaVersion: 1,
			Date:          time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
			Fields: map[string]report.FieldValue{
				"done": {Type: "task_list", Tasks: []report.Task{}},
			},
		},
		"mixed text and task_list": {
			SchemaVersion: 1,
			Date:          time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
			Fields: map[string]report.FieldValue{
				"done": {
					Type: "task_list",
					Tasks: []report.Task{
						{Title: "PR review", Time: "1h30m", Outcome: "merged"},
					},
				},
				"todo":     {Type: "text", Body: "明日の続き"},
				"thoughts": {Type: "text", Body: "集中できた"},
			},
		},
		"empty fields map": {
			SchemaVersion: 1,
			Date:          time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
			Fields:        map[string]report.FieldValue{},
		},
		"text with multiline body": {
			SchemaVersion: 1,
			Date:          time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
			Fields: map[string]report.FieldValue{
				"todo": {Type: "text", Body: "line1\nline2\n"},
			},
		},
	}

	for name, want := range tests {
		t.Run(name, func(t *testing.T) {
			raw, err := yaml.Marshal(want)
			require.NoError(t, err)

			var got report.Report
			err = yaml.Unmarshal(raw, &got)
			require.NoError(t, err, "round-trip yaml:\n%s", string(raw))

			assert.Equal(t, want.SchemaVersion, got.SchemaVersion)
			assert.True(t, want.Date.Equal(got.Date))
			assert.Equal(t, want.Fields, got.Fields)
		})
	}
}

func TestMarshalShape(t *testing.T) {
	tests := map[string]struct {
		input        *report.Report
		wantContains []string
		wantExcludes []string
	}{
		"text field excludes tasks": {
			input: &report.Report{
				SchemaVersion: 1,
				Date:          time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
				Fields: map[string]report.FieldValue{
					"memo": {Type: "text", Body: "hello"},
				},
			},
			wantContains: []string{"schema_version: 1", "date: \"2026-05-23\"", "type: text", "body: hello"},
			wantExcludes: []string{"tasks:"},
		},
		"task_list excludes body": {
			input: &report.Report{
				SchemaVersion: 1,
				Date:          time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
				Fields: map[string]report.FieldValue{
					"done": {
						Type:  "task_list",
						Tasks: []report.Task{{Title: "X"}},
					},
				},
			},
			wantContains: []string{"type: task_list", "tasks:", "title: X"},
			wantExcludes: []string{"body:"},
		},
		"task omits empty optional fields": {
			input: &report.Report{
				SchemaVersion: 1,
				Date:          time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
				Fields: map[string]report.FieldValue{
					"done": {
						Type:  "task_list",
						Tasks: []report.Task{{Title: "only"}},
					},
				},
			},
			wantContains: []string{"title: only"},
			wantExcludes: []string{"time:", "outcome:", "thoughts:"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			raw, err := yaml.Marshal(tt.input)
			require.NoError(t, err)
			out := string(raw)
			for _, s := range tt.wantContains {
				assert.Contains(t, out, s, "yaml:\n%s", out)
			}
			for _, s := range tt.wantExcludes {
				assert.NotContains(t, out, s, "yaml:\n%s", out)
			}
		})
	}
}

func TestMarshalRejectsInvalid(t *testing.T) {
	tests := map[string]struct {
		input         *report.Report
		wantErrSubstr string
	}{
		"zero schema_version": {
			input: &report.Report{
				Date:   time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
				Fields: map[string]report.FieldValue{},
			},
			wantErrSubstr: "unsupported schema_version",
		},
		"zero date": {
			input: &report.Report{
				SchemaVersion: 1,
				Fields:        map[string]report.FieldValue{},
			},
			wantErrSubstr: "date is required",
		},
		"unknown field type": {
			input: &report.Report{
				SchemaVersion: 1,
				Date:          time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
				Fields: map[string]report.FieldValue{
					"x": {Type: "bogus"},
				},
			},
			wantErrSubstr: "unsupported field type",
		},
		"text with tasks rejected": {
			input: &report.Report{
				SchemaVersion: 1,
				Date:          time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
				Fields: map[string]report.FieldValue{
					"x": {Type: "text", Tasks: []report.Task{{Title: "oops"}}},
				},
			},
			wantErrSubstr: "type=text must not contain tasks",
		},
		"task_list with body rejected": {
			input: &report.Report{
				SchemaVersion: 1,
				Date:          time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
				Fields: map[string]report.FieldValue{
					"x": {Type: "task_list", Body: "oops"},
				},
			},
			wantErrSubstr: "type=task_list must not contain body",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := yaml.Marshal(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrSubstr)
		})
	}
}

func TestDateFormat(t *testing.T) {
	r := &report.Report{
		SchemaVersion: 1,
		Date:          time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC),
		Fields:        map[string]report.FieldValue{},
	}
	raw, err := yaml.Marshal(r)
	require.NoError(t, err)
	out := string(raw)
	assert.True(t, strings.Contains(out, "2026-01-02"), "yaml:\n%s", out)
	assert.NotContains(t, out, "15:04:05")
	assert.NotContains(t, out, "T15")
}
