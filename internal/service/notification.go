package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/gabkaclassic/GitQuest/internal/config"
	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/metrics/pkg/httpclient"
)

type NotificationService interface {
	Notify(*[]dto.Notification) error
}

type notificationService struct {
	client          httpclient.HTTPClient
	authHeader      string
	authHeaderValue string
}

func NewNotificationService(cfg *config.Notification) (NotificationService, error) {

	if cfg == nil {
		return nil, errors.New("create notification service error: config cannot be nil")
	}

	client := httpclient.NewClient(httpclient.BaseURL(cfg.URL))

	return &notificationService{
		client:          client,
		authHeader:      cfg.Header,
		authHeaderValue: cfg.HeaderValue,
	}, nil
}

func (service *notificationService) Notify(notifications *[]dto.Notification) error {

	raw, err := json.Marshal(*notifications)

	if err != nil {
		slog.Error("Notifications marchalling error", slog.Any("error", err))
		return err
	}

	reader := bytes.NewReader(raw)

	resp, err := service.client.Post("", &httpclient.RequestOptions{
		Headers: &httpclient.Headers{
			service.authHeader: service.authHeaderValue,
		},
		Body: reader,
	})

	if err != nil {
		slog.Error("Notifications send error", slog.Any("error", err))
		return err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)

	if err != nil {
		slog.Error("Notifications send read response body error", slog.Any("error", err))
		return err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("Notifications send request error", slog.Int("status", resp.StatusCode), slog.String("body", string(responseBody)))
	}

	slog.Info("Send notification completed successfully", slog.String("body", string(responseBody)))

	return nil
}
