package handler

import (
	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/service"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewRuleHandler(t *testing.T) {
	tests := []struct {
		name        string
		service     service.RuleService
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid service",
			service:     &service.MockRuleService{},
			expectError: false,
		},
		{
			name:        "nil service",
			service:     nil,
			expectError: true,
			errorMsg:    "service is nil",
		},
		{
			name:        "mock implementation",
			service:     &service.MockRuleService{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			handler, err := NewRuleHandler(tt.service)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, handler)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, handler)
			assert.IsType(t, &RuleHandler{}, handler)
			assert.Equal(t, tt.service, handler.service)
		})
	}
}

func TestRuleHandler_GetAll(t *testing.T) {
	type mocks struct {
		service *service.MockRuleService
	}

	tests := []struct {
		name         string
		setup        func(m mocks)
		expectedCode int
		expectedBody string
	}{
		{
			name: "service returns rules",
			setup: func(m mocks) {
				m.service.On("GetAll").Return([]dto.Rule{
					{
						Name:        "1",
						Description: "1",
						EventType:   dto.Commit,
						Window:      123,
						Count:       1,
						Condition:   dto.EQ,
						Streak:      true,
						Reward:      1,
						Version:     "1",
					},
					{
						Name:        "2",
						Description: "2",
						EventType:   dto.CommentAdded,
						Window:      456,
						Count:       2,
						Condition:   dto.GT,
						Streak:      false,
						Reward:      2,
						Version:     "2",
					},
				})
			},
			expectedCode: http.StatusOK,
			expectedBody: `[
				{
					"name":"1",
					"description":"1",
					"eventType":"COMMIT",
					"window":123,
					"count":1,
					"condition":"==",
					"streak":true,
					"reward":1
				},
				{
					"name":"2",
					"description":"2",
					"eventType":"COMMENT_ADDED",
					"window":456,
					"count":2,
					"condition":">",
					"streak":false,
					"reward":2
				}
				]`,
		},
		{
			name: "service returns empty",
			setup: func(m mocks) {
				m.service.On("GetAll").Return([]dto.Rule{})
			},
			expectedCode: http.StatusOK,
			expectedBody: `[]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			m := mocks{
				service: new(service.MockRuleService),
			}
			tt.setup(m)

			h := &RuleHandler{service: m.service}

			req := httptest.NewRequest(http.MethodGet, "/rules", nil)
			w := httptest.NewRecorder()

			h.GetAll(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			body := strings.TrimSpace(w.Body.String())
			assert.JSONEq(t, tt.expectedBody, body)

			m.service.AssertExpectations(t)
		})
	}
}
