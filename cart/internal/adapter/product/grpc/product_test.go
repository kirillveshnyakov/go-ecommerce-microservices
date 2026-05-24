package grpc

import (
	"context"
	"net"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartErrors "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	productv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/product/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func TestProductClient_GetProduct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		server    productv1.ProductServiceServer
		wantName  string
		wantPrice uint32
		wantErr   error
	}{
		{
			name:      "success",
			server:    productServerStub{name: "keyboard", price: 100},
			wantName:  "keyboard",
			wantPrice: 100,
		},
		{
			name:    "not found",
			server:  productServerStub{err: status.Error(codes.NotFound, "product not found")},
			wantErr: cartErrors.ErrProductNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conn := productClientConn(t, tt.server)
			client := NewProductClient(conn)

			product, err := client.GetProduct(context.Background(), entity.SKU(10))
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantName, product.Name)
			require.Equal(t, tt.wantPrice, product.Price)
		})
	}
}

type productServerStub struct {
	productv1.UnimplementedProductServiceServer
	name  string
	price uint32
	err   error
}

func (s productServerStub) GetProduct(context.Context, *productv1.GetProductRequest) (*productv1.GetProductResponse, error) {
	if s.err != nil {
		return nil, s.err
	}

	return &productv1.GetProductResponse{Name: s.name, Price: s.price}, nil
}

func productClientConn(t *testing.T, server productv1.ProductServiceServer) *grpclib.ClientConn {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpclib.NewServer()
	productv1.RegisterProductServiceServer(grpcServer, server)

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
