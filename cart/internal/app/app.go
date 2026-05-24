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
	lomsAdapter "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/adapter/loms/grpc"
	productAdapter "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/adapter/product/grpc"
	stocksAdapter "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/adapter/stocks/grpc"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/config"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/controller"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/repository/cart"
	cartUsecase "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/usecase/cart"
	itemUsecase "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/usecase/item"
	db "github.com/igoroutine-courses/microservices.ecommerce.cart/migrations"
	"github.com/igoroutine-courses/microservices.ecommerce.pkg/transactor"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
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

	cartRepository := cart.NewCartRepository(dbPool)

	lomsConn, err := grpc.NewClient(
		cfg.Clients.LOMSGrpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error("failed to connect to loms server", zap.Error(err))
		return
	}
	defer func() {
		if err = lomsConn.Close(); err != nil {
			logger.Warn("failed to close loms connection", zap.Error(err))
		}
	}()

	productClient := productAdapter.NewProductClient(lomsConn)
	stocksClient := stocksAdapter.NewStocksClient(lomsConn)
	lomsClient := lomsAdapter.NewLOMSClient(lomsConn)

	itemUC := itemUsecase.NewItemService(cartRepository, productClient, stocksClient, logger)
	cartUC := cartUsecase.NewCartService(cartRepository, productClient, lomsClient, logger, transactor)

	api := controller.New(itemUC, cartUC, logger)

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
