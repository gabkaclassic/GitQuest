package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestEventType_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    EventType
		expectError bool
	}{
		{
			name:     "valid commit event",
			input:    `"COMMIT"`,
			expected: Commit,
		},
		{
			name:     "valid merge request opened",
			input:    `"MR_OPENED"`,
			expected: MergeRequestOpened,
		},
		{
			name:     "valid lowercase input",
			input:    `"commit"`,
			expected: Commit,
		},
		{
			name:        "invalid event type",
			input:       `"INVALID_EVENT"`,
			expectError: true,
		},
		{
			name:        "empty string",
			input:       `""`,
			expectError: true,
		},
		{
			name:        "not a string",
			input:       `123`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var et EventType
			err := json.Unmarshal([]byte(tt.input), &et)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, et)
			}
		})
	}
}

func TestEventType_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    EventType
		expectError bool
	}{
		{
			name:     "valid release published",
			input:    "RELEASE_PUBLISHED",
			expected: ReleasePublished,
		},
		{
			name:     "valid lowercase input",
			input:    "mr_merged",
			expected: MergeRequestMerged,
		},
		{
			name:     "valid mixed case",
			input:    "RePo_StAr",
			expected: RepositoryStarred,
		},
		{
			name:        "invalid event type",
			input:       "RANDOM_EVENT",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var et EventType
			err := yaml.Unmarshal([]byte(tt.input), &et)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, et)
			}
		})
	}
}

func TestValidateEventType(t *testing.T) {
	tests := []struct {
		name        string
		input       EventType
		expectError bool
	}{
		{
			name:  "valid issue opened",
			input: IssueOpened,
		},
		{
			name:  "valid issue closed",
			input: IssueClosed,
		},
		{
			name:  "valid comment added",
			input: CommentAdded,
		},
		{
			name:  "valid tag created",
			input: TagCreated,
		},
		{
			name:  "valid repository forked",
			input: RepositoryForked,
		},
		{
			name:        "invalid event type",
			input:       "INVALID_TYPE",
			expectError: true,
		},
		{
			name:        "empty event type",
			input:       "",
			expectError: true,
		},
		{
			name:        "lowercase valid type",
			input:       "commit",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEventType(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid event type")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEvent_MarshalBinary(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		expected string
	}{
		{
			name: "valid event with commit type",
			event: Event{
				ID:        "123",
				Actor:     "user1",
				Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				EventType: Commit,
			},
			expected: `{"id":"123","actor":"user1","timestamp":"2024-01-01T12:00:00Z","eventType":"COMMIT"}`,
		},
		{
			name: "valid event with review type",
			event: Event{
				ID:        "456",
				Actor:     "user2",
				Timestamp: time.Date(2024, 2, 1, 10, 30, 0, 0, time.UTC),
				EventType: ReviewSubmitted,
			},
			expected: `{"id":"456","actor":"user2","timestamp":"2024-02-01T10:30:00Z","eventType":"REVIEW_SUBMITTED"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.event.MarshalBinary()
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestEvent_UnmarshalBinary(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Event
		expectError bool
	}{
		{
			name:  "valid json event",
			input: `{"id":"789","actor":"user3","timestamp":"2024-03-01T14:45:00Z","eventType":"MR_MERGED"}`,
			expected: Event{
				ID:        "789",
				Actor:     "user3",
				Timestamp: time.Date(2024, 3, 1, 14, 45, 0, 0, time.UTC),
				EventType: MergeRequestMerged,
			},
		},
		{
			name:  "valid event with mr closed",
			input: `{"id":"999","actor":"user4","timestamp":"2024-04-01T09:15:00Z","eventType":"MR_CLOSED"}`,
			expected: Event{
				ID:        "999",
				Actor:     "user4",
				Timestamp: time.Date(2024, 4, 1, 9, 15, 0, 0, time.UTC),
				EventType: MergeRequestClosed,
			},
		},
		{
			name:        "invalid json",
			input:       `{not valid json}`,
			expectError: true,
		},
		{
			name:        "invalid event type in json",
			input:       `{"id":"111","actor":"user5","timestamp":"2024-01-01T00:00:00Z","eventType":"UNKNOWN"}`,
			expectError: true,
		},
		{
			name:        "missing required field",
			input:       `{"actor":"user6","timestamp":"2024-01-01T00:00:00Z","eventType":"COMMIT"}`,
			expectError: false,
			expected: Event{
				Actor:     "user6",
				Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EventType: Commit,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event Event
			err := event.UnmarshalBinary([]byte(tt.input))

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.ID, event.ID)
				assert.Equal(t, tt.expected.Actor, event.Actor)
				assert.Equal(t, tt.expected.Timestamp, event.Timestamp)
				assert.Equal(t, tt.expected.EventType, event.EventType)
			}
		})
	}
}

func TestEvent_MarshalAndUnmarshalBinary(t *testing.T) {
	tests := []struct {
		name  string
		event Event
	}{
		{
			name: "roundtrip repo starred event",
			event: Event{
				ID:        "star-123",
				Actor:     "stargazer",
				Timestamp: time.Now().UTC().Truncate(time.Second),
				EventType: RepositoryStarred,
			},
		},
		{
			name: "roundtrip repo forked event",
			event: Event{
				ID:        "fork-456",
				Actor:     "forker",
				Timestamp: time.Now().UTC().Add(-time.Hour).Truncate(time.Second),
				EventType: RepositoryForked,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.event.MarshalBinary()
			assert.NoError(t, err)

			var decoded Event
			err = decoded.UnmarshalBinary(data)
			assert.NoError(t, err)

			assert.Equal(t, tt.event.ID, decoded.ID)
			assert.Equal(t, tt.event.Actor, decoded.Actor)
			assert.Equal(t, tt.event.Timestamp, decoded.Timestamp)
			assert.Equal(t, tt.event.EventType, decoded.EventType)
		})
	}
}
