package stocks

import (
	"context"
	"errors"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	stocksv "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/stocks/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

//go:generate mockgen -source=stocks.go -destination=mocks/stocks_mocks.go -package=mocks
type (
	stocksService interface {
		GetStock(ctx context.Context, sku entity.SKU) (uint64, error)
		SetStock(ctx context.Context, sku entity.SKU, count uint64) error
	}
)

type stocksServer struct {
	stocksService stocksService
	logger        *zap.Logger
	stocksv.UnimplementedStocksServer
}

func NewStocksServer(stocksService stocksService, logger *zap.Logger) *stocksServer {
	return &stocksServer{
		stocksService: stocksService,
		logger:        logger,
	}
}

func (s *stocksServer) GetStock(ctx context.Context, req *stocksv.GetStockRequest) (*stocksv.GetStockResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation: %v", err)
	}

	stock, err := s.stocksService.GetStock(ctx, entity.SKU(req.GetSku()))
	if err != nil {
		if errors.Is(err, lomsErrors.ErrProductNotFound) {
			return nil, status.Errorf(codes.NotFound, "product not found")
		}

		s.logger.Error("stocks controller - get stock failed",
			zap.Uint32("sku", req.GetSku()),
			zap.Error(err),
		)

		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &stocksv.GetStockResponse{
		Count: stock,
	}, nil
}

func (s *stocksServer) SetStock(ctx context.Context, req *stocksv.SetStockRequest) (*emptypb.Empty, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation: %v", err)
	}

	err := s.stocksService.SetStock(ctx, entity.SKU(req.GetSku()), req.GetCount())
	if err != nil {
		if errors.Is(err, lomsErrors.ErrProductNotFound) {
			return nil, status.Errorf(codes.NotFound, "product not found")
		}

		s.logger.Error("stocks controller - set stock failed",
			zap.Uint32("sku", req.GetSku()),
			zap.Uint64("count", req.GetCount()),
			zap.Error(err),
		)

		return nil, status.Error(codes.Internal, "internal server error")
	}

	s.logger.Info("stock controller - set stock success",
		zap.Uint32("sku", req.GetSku()),
		zap.Uint64("count", req.GetCount()),
	)

	return &emptypb.Empty{}, nil
}
