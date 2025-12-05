package dto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestCondition_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Condition
		expectError bool
	}{
		{
			name:     "greater than",
			input:    `">"`,
			expected: GT,
		},
		{
			name:     "greater than or equal",
			input:    `">="`,
			expected: GTE,
		},
		{
			name:     "equal",
			input:    `"=="`,
			expected: EQ,
		},
		{
			name:     "less than",
			input:    `"<"`,
			expected: LT,
		},
		{
			name:     "less than or equal",
			input:    `"<="`,
			expected: LTE,
		},
		{
			name:        "invalid condition",
			input:       `"!="`,
			expectError: true,
		},
		{
			name:        "empty string",
			input:       `""`,
			expectError: true,
		},
		{
			name:        "multiple chars wrong",
			input:       `">>>"`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Condition
			err := yaml.Unmarshal([]byte(tt.input), &c)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid condition")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, c)
			}
		})
	}
}

func TestCondition_Compare(t *testing.T) {
	tests := []struct {
		name     string
		cond     Condition
		a        int
		b        int
		expected bool
	}{
		{
			name:     "greater than true",
			cond:     GT,
			a:        10,
			b:        5,
			expected: true,
		},
		{
			name:     "greater than false",
			cond:     GT,
			a:        5,
			b:        10,
			expected: false,
		},
		{
			name:     "greater than or equal equal",
			cond:     GTE,
			a:        10,
			b:        10,
			expected: true,
		},
		{
			name:     "greater than or equal greater",
			cond:     GTE,
			a:        15,
			b:        10,
			expected: true,
		},
		{
			name:     "less than true",
			cond:     LT,
			a:        3,
			b:        7,
			expected: true,
		},
		{
			name:     "less than or equal equal",
			cond:     LTE,
			a:        8,
			b:        8,
			expected: true,
		},
		{
			name:     "equal true",
			cond:     EQ,
			a:        42,
			b:        42,
			expected: true,
		},
		{
			name:     "equal false",
			cond:     EQ,
			a:        42,
			b:        43,
			expected: false,
		},
		{
			name:     "invalid condition returns false",
			cond:     Condition("??"),
			a:        1,
			b:        1,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cond.Compare(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTimeRange_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    time.Duration
		expectError bool
	}{
		{
			name:     "valid hour duration",
			input:    "1h",
			expected: time.Hour,
		},
		{
			name:     "valid minute duration",
			input:    "30m",
			expected: 30 * time.Minute,
		},
		{
			name:     "valid day duration",
			input:    "24h",
			expected: 24 * time.Hour,
		},
		{
			name:     "valid complex duration",
			input:    "1h30m",
			expected: 90 * time.Minute,
		},
		{
			name:        "exceeds max time range",
			input:       "720h1s",
			expectError: true,
		},
		{
			name:        "invalid duration format",
			input:       "not-a-duration",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       `""`,
			expectError: true,
		},
		{
			name:        "negative duration",
			input:       "-1h",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tr TimeRange
			err := yaml.Unmarshal([]byte(tt.input), &tr)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, TimeRange(tt.expected), tr)
			}
		})
	}
}

func TestTimeRange_ToNanoseconds(t *testing.T) {
	tests := []struct {
		name     string
		tr       TimeRange
		expected int64
	}{
		{
			name:     "one hour",
			tr:       TimeRange(time.Hour),
			expected: time.Hour.Nanoseconds(),
		},
		{
			name:     "zero duration",
			tr:       TimeRange(0),
			expected: 0,
		},
		{
			name:     "one minute",
			tr:       TimeRange(time.Minute),
			expected: time.Minute.Nanoseconds(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tr.ToNanoseconds()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromNanoseconds(t *testing.T) {
	tests := []struct {
		name     string
		ns       int64
		expected TimeRange
	}{
		{
			name:     "hour in nanoseconds",
			ns:       time.Hour.Nanoseconds(),
			expected: TimeRange(time.Hour),
		},
		{
			name:     "zero nanoseconds",
			ns:       0,
			expected: TimeRange(0),
		},
		{
			name:     "minute in nanoseconds",
			ns:       time.Minute.Nanoseconds(),
			expected: TimeRange(time.Minute),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FromNanoseconds(tt.ns)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRule_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		expected    Rule
		expectError bool
	}{
		{
			name: "valid rule with commit event",
			yamlContent: `
name: "Commit rule"
description: "Rule for commits"
event_type: COMMIT
window: 24h
count: 5
condition: ">="
streak: false
reward: 100
`,
			expected: Rule{
				Name:        "Commit rule",
				Description: "Rule for commits",
				EventType:   Commit,
				Window:      TimeRange(24 * time.Hour),
				Count:       5,
				Condition:   GTE,
				Streak:      false,
				Reward:      100,
			},
		},
		{
			name: "valid rule with streak",
			yamlContent: `
name: "Streak rule"
description: "Daily streak rule"
event_type: MR_OPENED
window: 168h
count: 7
condition: "=="
streak: true
reward: 500
`,
			expected: Rule{
				Name:        "Streak rule",
				Description: "Daily streak rule",
				EventType:   MergeRequestOpened,
				Window:      TimeRange(168 * time.Hour),
				Count:       7,
				Condition:   EQ,
				Streak:      true,
				Reward:      500,
			},
		},
		{
			name: "invalid condition",
			yamlContent: `
name: "Invalid rule"
event_type: ISSUE_OPENED
window: 1h
count: 1
condition: "!="
streak: false
reward: 10
`,
			expectError: true,
		},
		{
			name: "invalid event type",
			yamlContent: `
name: "Invalid event"
event_type: INVALID_EVENT
window: 1h
count: 1
condition: ">"
streak: false
reward: 10
`,
			expectError: true,
		},
		{
			name: "window exceeds max",
			yamlContent: `
name: "Too long window"
event_type: COMMENT_ADDED
window: 800h
count: 1
condition: ">"
streak: false
reward: 10
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rule Rule
			err := yaml.Unmarshal([]byte(tt.yamlContent), &rule)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Name, rule.Name)
				assert.Equal(t, tt.expected.Description, rule.Description)
				assert.Equal(t, tt.expected.EventType, rule.EventType)
				assert.Equal(t, tt.expected.Window, rule.Window)
				assert.Equal(t, tt.expected.Count, rule.Count)
				assert.Equal(t, tt.expected.Condition, rule.Condition)
				assert.Equal(t, tt.expected.Streak, rule.Streak)
				assert.Equal(t, tt.expected.Reward, rule.Reward)
				assert.NotEmpty(t, rule.Version)
			}
		})
	}
}

func TestRule_VersionConsistency(t *testing.T) {
	tests := []struct {
		name        string
		rule1       string
		rule2       string
		sameVersion bool
		expectError bool
	}{
		{
			name: "identical rules have same version",
			rule1: `
name: "Rule 1"
event_type: COMMIT
window: 24h
count: 5
condition: ">="
streak: false
reward: 100
`,
			rule2: `
name: "Rule 2"
event_type: COMMIT
window: 24h
count: 5
condition: ">="
streak: false
reward: 100
`,
			sameVersion: true,
		},
		{
			name: "different count different version",
			rule1: `
name: "Rule A"
event_type: MR_MERGED
window: 48h
count: 3
condition: ">"
streak: true
reward: 200
`,
			rule2: `
name: "Rule A"
event_type: MR_MERGED
window: 48h
count: 4
condition: ">"
streak: true
reward: 200
`,
			sameVersion: false,
		},
		{
			name: "different reward different version",
			rule1: `
event_type: ISSUE_CLOSED
window: 12h
count: 2
condition: "=="
streak: false
reward: 50
`,
			rule2: `
event_type: ISSUE_CLOSED
window: 12h
count: 2
condition: "=="
streak: false
reward: 75
`,
			sameVersion: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rule1, rule2 Rule
			err1 := yaml.Unmarshal([]byte(tt.rule1), &rule1)
			err2 := yaml.Unmarshal([]byte(tt.rule2), &rule2)

			if tt.expectError {
				assert.True(t, err1 != nil || err2 != nil)
				return
			}

			assert.NoError(t, err1)
			assert.NoError(t, err2)

			if tt.sameVersion {
				assert.Equal(t, rule1.Version, rule2.Version)
			} else {
				assert.NotEqual(t, rule1.Version, rule2.Version)
			}
		})
	}
}

func TestComputeHash(t *testing.T) {
	tests := []struct {
		name     string
		input    ruleHash
		expected string
	}{
		{
			name: "compute hash for valid rule hash",
			input: ruleHash{
				EventType: Commit,
				Window:    TimeRange(time.Hour),
				Count:     5,
				Condition: GTE,
				Streak:    false,
				Reward:    100,
			},
		},
		{
			name: "hash changes with different values",
			input: ruleHash{
				EventType: MergeRequestOpened,
				Window:    TimeRange(2 * time.Hour),
				Count:     3,
				Condition: EQ,
				Streak:    true,
				Reward:    200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1, err1 := computeHash(tt.input)
			hash2, err2 := computeHash(tt.input)

			assert.NoError(t, err1)
			assert.NoError(t, err2)
			assert.Equal(t, hash1, hash2)
			assert.NotEmpty(t, hash1)
		})
	}
}
