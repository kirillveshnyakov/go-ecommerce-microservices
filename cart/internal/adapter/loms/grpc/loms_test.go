package grpc

import (
	"context"
	"net"
	"testing"

	cartErrors "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/port"
	lomsv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/loms/v1"
	"github.com/stretchr/testify/require"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestLOMSClient_CreateOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		server      lomsv1.LomsServer
		wantOrderID int64
		wantErr     error
	}{
		{
			name:        "success",
			server:      lomsServerStub{orderID: 777},
			wantOrderID: 777,
		},
		{
			name:    "insufficient stock",
			server:  lomsServerStub{err: status.Error(codes.FailedPrecondition, "insufficient stock")},
			wantErr: cartErrors.ErrInsufficientStock,
		},
		{
			name:    "not found",
			server:  lomsServerStub{err: status.Error(codes.NotFound, "product not found")},
			wantErr: cartErrors.ErrProductNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conn := lomsClientConn(t, tt.server)
			client := NewLOMSClient(conn)

			orderID, err := client.CreateOrder(context.Background(), port.CreateOrderRequest{
				UserID: 42,
				Items:  []port.Item{{SKU: 10, Count: 2}},
			})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantOrderID, orderID)
		})
	}
}

type lomsServerStub struct {
	lomsv1.UnimplementedLomsServer
	orderID int64
	err     error
}

func (s lomsServerStub) CreateOrder(context.Context, *lomsv1.CreateOrderRequest) (*lomsv1.CreateOrderResponse, error) {
	if s.err != nil {
		return nil, s.err
	}

	return &lomsv1.CreateOrderResponse{OrderId: s.orderID}, nil
}

func lomsClientConn(t *testing.T, server lomsv1.LomsServer) *grpclib.ClientConn {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpclib.NewServer()
	lomsv1.RegisterLomsServer(grpcServer, server)

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
