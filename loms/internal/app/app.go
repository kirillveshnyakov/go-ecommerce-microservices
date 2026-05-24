package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	notificationsKafka "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/adapter/notifications/kafka"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/config"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/controller"
	outboxCore "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/outbox"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/order"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/outbox"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/product"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/stocks"
	lomsUC "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/usecase/loms"
	productUC "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/usecase/product"
	stocksUC "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/usecase/stocks"
	db "github.com/igoroutine-courses/microservices.ecommerce.loms/migrations"
	"github.com/igoroutine-courses/microservices.ecommerce.pkg/transactor"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

var (
	ErrUnsupportedOutboxKind = errors.New("unsupported outbox kind")
)

func Run(logger *zap.Logger, cfg *config.Config) {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt, syscall.SIGINT, syscall.SIGTERM,
	)
	defer stop()

	pgxcfg, err := pgxpool.ParseConfig(cfg.ConstructPostgresURL())

	if err != nil {
		logger.Error("can not create pgxpool cfg", zap.Error(err))
		return
	}

	dbPool, err := pgxpool.NewWithConfig(ctx, pgxcfg)

	if err != nil {
		logger.Error("can not create pgxpool", zap.Error(err))
		return
	}

	defer dbPool.Close()

	db.SetupPostgres(dbPool, logger)

	transactor := transactor.NewTransactor(dbPool)

	productRepository := product.NewProductRepository(dbPool)
	orderRepository := order.NewOrdersRepository(dbPool)
	stocksRepository := stocks.NewStocksRepository(dbPool)
	outboxRepository := outbox.NewOutboxRepository(dbPool)

	kafkaBrokers := cfg.KafkaBrokerAddrs()
	if len(kafkaBrokers) == 0 {
		logger.Error("no kafka brokers configured (KAFKA_BROKERS)")
		return
	}

	notificationsPublisher := notificationsKafka.NewPublisher(kafkaBrokers, cfg.Kafka.Topic)
	defer func() {
		if err := notificationsPublisher.Close(); err != nil {
			logger.Error("close kafka notifications publisher", zap.Error(err))
		}
	}()

	lomsUC := lomsUC.NewLomsService(orderRepository, stocksRepository, outboxRepository, notificationsPublisher, logger, transactor)
	productUC := productUC.NewProductService(productRepository, logger)
	stocksUC := stocksUC.NewStocksService(stocksRepository, logger)

	api := controller.New(lomsUC, productUC, stocksUC, logger)

	globalOutboxHandler := func(kind outbox.Kind) (outboxCore.KindHandler, error) {
		switch kind {
		case outbox.KindNotification:
			return lomsUC.OrderStatusChangedNotificationKindHandler, nil
		default:
			return nil, ErrUnsupportedOutboxKind
		}
	}

	outboxCore := outboxCore.New(logger, outboxRepository, globalOutboxHandler, transactor)

	waitOutbox := outboxCore.Start(
		ctx,
		cfg.Outbox.Workers,
		cfg.Outbox.BatchSize,
		cfg.Outbox.FetchPeriod,
		cfg.Outbox.TTL,
	)

	var wg sync.WaitGroup
	wg.Go(func() {
		err1 := runGrpc(ctx, logger, cfg, api)
		if err1 != nil {
			logger.Error("failed to run grpc server", zap.Error(err1))
			stop()
		}
	})
	wg.Go(func() {
		err2 := runRest(ctx, logger, cfg, api)
		if err2 != nil {
			logger.Error("failed to run http server", zap.Error(err2))
			stop()
		}
	})

	wg.Wait()
	waitOutbox()
}

func runRest(ctx context.Context, logger *zap.Logger, cfg *config.Config, api *controller.API) error {
	mux := grpcruntime.NewServeMux()
	endpoint := net.JoinHostPort("localhost", cfg.GRPC.Port)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	if err := api.RegisterGateway(ctx, mux, endpoint, opts); err != nil {
		return fmt.Errorf("register gateway error: %w", err)
	}

	serveErr := make(chan error, 1)
	defer close(serveErr)

	server := &http.Server{
		Addr:    ":" + cfg.GRPC.GatewayPort,
		Handler: corsHandler(mux),
	}

	go func() {
		logger.Info("http gateway started", zap.String("address", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- fmt.Errorf("http gateway listen error: %w", err)
			return
		}
		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.GRPC.HTTPShutdownTime)
	defer cancel()

	logger.Info("shutting down http server")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Warn("http gateway shutdown error", zap.Error(err))
		if closeErr := server.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
			logger.Warn("http gateway forced close error", zap.Error(closeErr))
		}
		return fmt.Errorf("http gateway shutdown error: %w", err)
	}
	logger.Info("http gateway gracefully shutdown")

	return <-serveErr
}

func runGrpc(ctx context.Context, logger *zap.Logger, cfg *config.Config, api *controller.API) error {
	listener, err := net.Listen("tcp", ":"+cfg.GRPC.Port)
	if err != nil {
		return fmt.Errorf("grpc failed to listen: %w", err)
	}

	serveErr := make(chan error, 1)
	defer close(serveErr)

	server := grpc.NewServer()
	api.RegisterGRPC(server)
	reflection.Register(server)

	go func() {
		logger.Info("grpc server started", zap.String("address", listener.Addr().String()))

		if err2 := server.Serve(listener); err2 != nil && !errors.Is(err2, grpc.ErrServerStopped) {
			serveErr <- fmt.Errorf("grpc server error: %w", err2)
			return
		}
		serveErr <- nil
	}()

	select {
	case err = <-serveErr:
		return err
	case <-ctx.Done():
	}

	logger.Info("starting graceful stop grpc server")
	stopped := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(stopped)
	}()

	timer := time.NewTimer(cfg.GRPC.GrpcShutdownTime)
	defer timer.Stop()

	select {
	case <-stopped:
		logger.Info("grpc server stopped gracefully")
	case <-timer.C:
		logger.Warn(
			"grpc server graceful shutdown timeout exceeded, forcing stop",
			zap.Duration("timeout", cfg.GRPC.GrpcShutdownTime),
		)
		server.Stop()
		<-stopped
	}

	return <-serveErr
}

func corsHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "http://localhost:5173"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
