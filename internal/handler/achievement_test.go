package handler

import (
	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/service"
	api "github.com/gabkaclassic/metrics/pkg/error"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewAchievementHandler(t *testing.T) {
	tests := []struct {
		name        string
		service     service.AchievementService
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid service",
			service:     &service.MockAchievementService{},
			expectError: false,
		},
		{
			name:        "nil service",
			service:     nil,
			expectError: true,
			errorMsg:    "service is nil",
		},
		{
			name:        "mock service implementation",
			service:     &service.MockAchievementService{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewAchievementHandler(tt.service)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, handler)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, handler)
				assert.IsType(t, &AchievementHandler{}, handler)
				assert.Equal(t, tt.service, handler.service)
			}
		})
	}
}

func TestAchievementHandler_GetByUser(t *testing.T) {
	type mocks struct {
		service *service.MockAchievementService
	}

	tests := []struct {
		name         string
		setup        func(m mocks)
		path         string
		expectedCode int
		expectedBody string
	}{
		{
			name: "service returns error",
			path: "/achievements/user1",
			setup: func(m mocks) {
				m.service.On("GetSummaryByUser", "user1").
					Return(nil, api.BadRequest("fail"))
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"fail"}`,
		},
		{
			name: "service returns summary",
			path: "/achievements/user1",
			setup: func(m mocks) {
				m.service.On("GetSummaryByUser", "user1").
					Return(&dto.AchievementsSummary{
						Achievements: []dto.AchievementInfo{
							{RuleName: "rule1", Reward: 10},
						},
						RewardSum: 10,
					}, nil)
			},
			expectedCode: http.StatusOK,
			expectedBody: `{"achievements":[{"rule":"rule1","reward":10}],"rewardSum":10}`,
		},
		{
			name: "encode fails (writer returns error)",
			path: "/achievements/user1",
			setup: func(m mocks) {
				m.service.On("GetSummaryByUser", "user1").
					Return(&dto.AchievementsSummary{}, nil)
			},
			expectedCode: http.StatusOK,
			expectedBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			m := mocks{
				service: new(service.MockAchievementService),
			}
			tt.setup(m)

			h := &AchievementHandler{service: m.service}

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.SetPathValue("user", "user1")

			w := httptest.NewRecorder()

			h.GetByUser(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, strings.TrimSpace(w.Body.String()))
			}

			m.service.AssertExpectations(t)
		})
	}
}
