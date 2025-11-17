package dto

import (
	"fmt"
	"strings"
	"time"
)

type (
	EventType string

	Event struct {
		ID        string
		Actor     string
		Timestamp time.Time
		EventType EventType
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
	if _, ok := validEventTypes[evt]; !ok {
		return fmt.Errorf("invalid event type: %s", raw)
	}

	*e = evt
	return nil
}
