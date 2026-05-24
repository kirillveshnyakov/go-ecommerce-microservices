package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	notificationsErrors "github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNormalizeCallbackURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: " ", want: ""},
		{name: "without scheme", in: "example.com", want: "http://example.com"},
		{name: "with scheme", in: "https://example.com/callback", want: "https://example.com/callback"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, normalizeCallbackURL(tt.in))
		})
	}
}

func TestClient_SendMessage(t *testing.T) {
	t.Parallel()

	message := port.CallbackPayload{
		UserID:  31,
		OrderID: 16,
		Status:  "paid",
	}

	tests := []struct {
		name     string
		callback func(*testing.T) string
		wantErr  error
	}{
		{
			name: "success",
			callback: func(t *testing.T) string {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

					var got port.CallbackPayload
					if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&got)) {
						return
					}
					assert.Equal(t, message, got)

					w.WriteHeader(http.StatusOK)
				}))
				t.Cleanup(server.Close)

				return server.URL
			},
		},
		{
			name: "empty callback",
			callback: func(*testing.T) string {
				return ""
			},
			wantErr: notificationsErrors.ErrEmptyCallbackAddr,
		},
		{
			name: "non 2xx response",
			callback: func(t *testing.T) string {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
				t.Cleanup(server.Close)

				return server.URL
			},
			wantErr: notificationsErrors.ErrSendNotification,
		},
		{
			name: "invalid callback URL",
			callback: func(*testing.T) string {
				return "http://[::1"
			},
			wantErr: notificationsErrors.ErrSendNotification,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := NewClient(tt.callback(t), zap.NewNop())

			err := client.SendMessage(context.Background(), message)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
