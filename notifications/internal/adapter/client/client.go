package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	notificationsErrors "github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/port"
	"go.uber.org/zap"
)

type Client struct {
	httpClient  *http.Client
	callbackURL string
	logger      *zap.Logger
}

func NewClient(callbackURL string, logger *zap.Logger) *Client {
	return &Client{
		httpClient:  &http.Client{},
		callbackURL: normalizeCallbackURL(callbackURL),
		logger:      logger,
	}
}

func normalizeCallbackURL(url string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return url
	}
	if !strings.Contains(url, "://") {
		url = "http://" + url
	}
	return url
}

func (c *Client) SendMessage(ctx context.Context, message port.CallbackPayload) error {
	if c.callbackURL == "" {
		return fmt.Errorf("client - send message: %w", notificationsErrors.ErrEmptyCallbackAddr)
	}

	body, err := json.Marshal(message)
	if err != nil {
		return c.wrapSendMessageError(err, "marshalling message", message)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.callbackURL, bytes.NewBuffer(body))
	if err != nil {
		return c.wrapSendMessageError(err, "create request", message)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.wrapSendMessageError(err, "send request", message)
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			c.logger.Error("failed to close response body", zap.Error(err))
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return c.wrapSendMessageError(err, "non-2xx status", message)
	}

	return nil
}

func (c *Client) wrapSendMessageError(err error, errorMsg string, message port.CallbackPayload) error {
	c.logger.Error("client - send message",
		zap.String("cause of error", errorMsg),
		zap.Any("message", message),
		zap.Error(err),
	)

	return fmt.Errorf("client - send message: %w", notificationsErrors.ErrSendNotification)
}
