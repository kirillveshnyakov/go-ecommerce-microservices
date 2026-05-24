package grpc

import (
	"context"
	"net"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartErrors "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	stocksv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/stocks/v1"
	"github.com/stretchr/testify/require"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestStocksClient_GetStock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		server    stocksv1.StocksServer
		wantStock uint64
		wantErr   error
	}{
		{
			name:      "success",
			server:    stocksServerStub{count: 42},
			wantStock: 42,
		},
		{
			name:    "not found",
			server:  stocksServerStub{err: status.Error(codes.NotFound, "product not found")},
			wantErr: cartErrors.ErrProductNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conn := stocksClientConn(t, tt.server)
			client := NewStocksClient(conn)

			stock, err := client.GetStock(context.Background(), entity.SKU(10))
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantStock, stock)
		})
	}
}

type stocksServerStub struct {
	stocksv1.UnimplementedStocksServer
	count uint64
	err   error
}

func (s stocksServerStub) GetStock(context.Context, *stocksv1.GetStockRequest) (*stocksv1.GetStockResponse, error) {
	if s.err != nil {
		return nil, s.err
	}

	return &stocksv1.GetStockResponse{Count: s.count}, nil
}

func stocksClientConn(t *testing.T, server stocksv1.StocksServer) *grpclib.ClientConn {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpclib.NewServer()
	stocksv1.RegisterStocksServer(grpcServer, server)

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
