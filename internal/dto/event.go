package dto

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type (
	EventType string

	Event struct {
		ID        string    `json:"id"`
		Actor     string    `json:"actor"`
		Timestamp time.Time `json:"timestamp"`
		EventType EventType `json:"eventType"`
	}
)

const (
	Commit             EventType = "COMMIT"
	MergeRequestOpened EventType = "MR_OPENED"
	MergeRequestMerged EventType = "MR_MERGED"
	MergeRequestClosed EventType = "MR_CLOSED"
	IssueOpened        EventType = "ISSUE_OPENED"
	IssueClosed        EventType = "ISSUE_CLOSED"
	ReviewSubmitted    EventType = "REVIEW_SUBMITTED"
	CommentAdded       EventType = "COMMENT_ADDED"
	TagCreated         EventType = "TAG_CREATED"
	ReleasePublished   EventType = "RELEASE_PUBLISHED"
	RepositoryStarred  EventType = "REPO_STAR"
	RepositoryForked   EventType = "REPO_FORK"
)

var validEventTypes = map[EventType]struct{}{
	Commit:             {},
	MergeRequestOpened: {},
	MergeRequestMerged: {},
	MergeRequestClosed: {},
	IssueOpened:        {},
	IssueClosed:        {},
	ReviewSubmitted:    {},
	CommentAdded:       {},
	TagCreated:         {},
	ReleasePublished:   {},
	RepositoryStarred:  {},
	RepositoryForked:   {},
}

func (e *EventType) UnmarshalYAML(unmarshal func(any) error) error {
	var raw string
	if err := unmarshal(&raw); err != nil {
		return err
	}

	evt := EventType(strings.ToUpper(raw))

	if err := ValidateEventType(evt); err != nil {
		return fmt.Errorf("invalid event type: %s", raw)
	}

	*e = evt
	return nil
}

func (et *EventType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	eventType := EventType(s)
	if err := ValidateEventType(eventType); err != nil {
		return err
	}

	*et = eventType
	return nil
}

func ValidateEventType(value EventType) error {
	if _, ok := validEventTypes[value]; !ok {
		return fmt.Errorf("invalid event type: %s", value)
	}

	return nil
}

func (e *Event) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *Event) MarshalBinary() ([]byte, error) {
	return json.Marshal(e)
}
