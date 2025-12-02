package service

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gabkaclassic/GitQuest/internal/config"
	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/metrics/pkg/httpclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewNotificationService(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Notification
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration with all fields",
			cfg: &config.Notification{
				URL:         "https://api.example.com",
				Header:      "Authorization",
				HeaderValue: "Bearer token123",
			},
			expectError: false,
		},
		{
			name: "valid configuration with empty header",
			cfg: &config.Notification{
				URL:         "https://api.example.com",
				Header:      "",
				HeaderValue: "",
			},
			expectError: false,
		},
		{
			name: "valid configuration with only URL",
			cfg: &config.Notification{
				URL: "https://api.example.com",
			},
			expectError: false,
		},
		{
			name: "configuration with empty URL",
			cfg: &config.Notification{
				URL: "",
			},
			expectError: false,
		},
		{
			name:        "nil configuration",
			cfg:         nil,
			expectError: true,
		},
		{
			name: "configuration with invalid URL format",
			cfg: &config.Notification{
				URL: "://invalid-url",
			},
			expectError: false,
		},
		{
			name: "configuration with HTTP URL",
			cfg: &config.Notification{
				URL:         "http://localhost:8080",
				Header:      "X-API-Key",
				HeaderValue: "test-key",
			},
			expectError: false,
		},
		{
			name: "configuration with long header value",
			cfg: &config.Notification{
				URL:         "https://api.example.com",
				Header:      "X-Custom-Header",
				HeaderValue: strings.Repeat("a", 1000),
			},
			expectError: false,
		},
		{
			name: "configuration with special characters in URL",
			cfg: &config.Notification{
				URL: "https://api.example.com:8080/api/v1/notifications?param=value",
			},
			expectError: false,
		},
		{
			name: "configuration with only header but no value",
			cfg: &config.Notification{
				URL:    "https://api.example.com",
				Header: "Authorization",
			},
			expectError: false,
		},
		{
			name: "configuration with only header value but no header",
			cfg: &config.Notification{
				URL:         "https://api.example.com",
				HeaderValue: "Bearer token",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewNotificationService(tt.cfg)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, service)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
				assert.IsType(t, &notificationService{}, service)

				svc := service.(*notificationService)
				assert.NotNil(t, svc.client)

				if tt.cfg != nil {
					assert.Equal(t, tt.cfg.Header, svc.authHeader)
					assert.Equal(t, tt.cfg.HeaderValue, svc.authHeaderValue)
				} else {
					assert.Empty(t, svc.authHeader)
					assert.Empty(t, svc.authHeaderValue)
				}
			}
		})
	}
}

func TestNotificationService_Notify(t *testing.T) {
	type mocks struct {
		client *httpclient.MockHTTPClient
	}

	notifications := []dto.Notification{
		{User: "u1", Reward: 10, RuleName: "r1"},
	}

	tests := []struct {
		name      string
		setup     func(m mocks)
		expectErr bool
	}{
		{
			name: "post returns error",
			setup: func(m mocks) {
				m.client.On("Post", "", mock.Anything).
					Return(nil, errors.New("post err"))
			},
			expectErr: true,
		},
		{
			name: "status != 200",
			setup: func(m mocks) {
				resp := &http.Response{
					StatusCode: 500,
					Body:       io.NopCloser(bytes.NewBufferString("error here")),
				}
				m.client.On("Post", "", mock.Anything).
					Return(resp, nil)
			},
			expectErr: false,
		},
		{
			name: "status != 200 + body read error",
			setup: func(m mocks) {
				broken := io.NopCloser(errReader{})
				resp := &http.Response{
					StatusCode: 500,
					Body:       broken,
				}
				m.client.On("Post", "", mock.Anything).
					Return(resp, nil)
			},
			expectErr: true,
		},
		{
			name: "success but response body read error",
			setup: func(m mocks) {
				broken := io.NopCloser(errReader{})
				resp := &http.Response{
					StatusCode: 200,
					Body:       broken,
				}
				m.client.On("Post", "", mock.Anything).
					Return(resp, nil)
			},
			expectErr: true,
		},
		{
			name: "success",
			setup: func(m mocks) {
				resp := &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString("OK")),
				}
				m.client.On("Post", "", mock.Anything).
					Return(resp, nil)
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockObjs := mocks{
				client: new(httpclient.MockHTTPClient),
			}

			tt.setup(mockObjs)

			svc := &notificationService{
				client:          mockObjs.client,
				authHeader:      "Authorization",
				authHeaderValue: "Bearer X",
			}

			input := notifications
			err := svc.Notify(input)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockObjs.client.AssertExpectations(t)
		})
	}
}

type errReader struct{}

func (errReader) Read(p []byte) (int, error) {
	return 0, errors.New("read error")
}
func (errReader) Close() error { return nil }
