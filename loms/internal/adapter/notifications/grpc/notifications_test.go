package grpc

import (
	"context"
	"net"
	"testing"

	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/port"
	notificationsv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/notifications/api/v1"
	"github.com/stretchr/testify/require"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestNotificationsClient_SendOrderStatusChangedNotification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		server  notificationsv1.NotificationsServer
		wantErr error
	}{
		{
			name:   "success",
			server: notificationsServerStub{},
		},
		{
			name:    "unavailable",
			server:  notificationsServerStub{err: status.Error(codes.Unavailable, "send failed")},
			wantErr: lomsErrors.ErrSendNotification,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conn := notificationsClientConn(t, tt.server)
			client := NewNotificationsClient(conn)

			err := client.SendOrderStatusChangedNotification(context.Background(), port.Notification{
				UserID:  42,
				OrderID: 777,
				Status:  port.OrderStatusPaid,
			})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

type notificationsServerStub struct {
	notificationsv1.UnimplementedNotificationsServer
	err error
}

func (s notificationsServerStub) SendOrderStatusChangedNotification(context.Context, *notificationsv1.OrderStatusChangedNotificationRequest) (*emptypb.Empty, error) {
	if s.err != nil {
		return nil, s.err
	}

	return &emptypb.Empty{}, nil
}

func notificationsClientConn(t *testing.T, server notificationsv1.NotificationsServer) *grpclib.ClientConn {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpclib.NewServer()
	notificationsv1.RegisterNotificationsServer(grpcServer, server)

	go func() {
		_ = grpcServer.Serve(listener)
	}()
	t.Cleanup(grpcServer.Stop)

	conn, err := grpclib.NewClient(
		"passthrough:///bufnet",
		grpclib.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpclib.WithInsecure(),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
	})

	return conn
}
