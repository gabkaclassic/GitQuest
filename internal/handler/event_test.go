package handler

import (
	"bytes"
	"github.com/gabkaclassic/GitQuest/internal/service"
	api "github.com/gabkaclassic/metrics/pkg/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewEventHandler(t *testing.T) {
	tests := []struct {
		name        string
		service     service.EventService
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid service",
			service:     &service.MockEventService{},
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
			service:     &service.MockEventService{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewEventHandler(tt.service)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, handler)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, handler)
				assert.IsType(t, &EventHandler{}, handler)
				assert.Equal(t, tt.service, handler.service)
			}
		})
	}
}

func TestEventHandler_SaveAll(t *testing.T) {
	type mocks struct {
		service *service.MockEventService
	}

	tests := []struct {
		name         string
		body         string
		setup        func(m mocks)
		expectedCode int
	}{
		{
			name:         "invalid json",
			body:         `{"broken":`,
			setup:        func(m mocks) {},
			expectedCode: http.StatusUnprocessableEntity,
		},
		{
			name: "service returns error",
			body: `[{"user":"u1","action":"a1"}]`,
			setup: func(m mocks) {
				m.service.On("SaveAll", mock.Anything, mock.Anything).
					Return(api.BadRequest("fail"))
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "success",
			body: `[{"user":"u1","action":"a1"}]`,
			setup: func(m mocks) {
				m.service.On("SaveAll", mock.Anything, mock.Anything).
					Return(nil)
			},
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mocks{
				service: new(service.MockEventService),
			}
			tt.setup(m)

			h := &EventHandler{service: m.service}

			req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.SaveAll(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
			m.service.AssertExpectations(t)
		})
	}
}
