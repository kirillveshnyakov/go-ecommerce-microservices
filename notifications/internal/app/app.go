package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	clientAdapter "github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/adapter/client"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/adapter/kafka"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/config"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/controller"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/inbox"
	inboxRepo "github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/repository/inbox"
	notificationsUC "github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/usecase/notifier"
	db "github.com/igoroutine-courses/microservices.ecommerce.notifications/migrations"
	"github.com/igoroutine-courses/microservices.ecommerce.pkg/transactor"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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
	inboxRepository := inboxRepo.NewInboxRepository(dbPool)

	callbackAddr := strings.TrimSpace(cfg.Clients.CallbackAddr)
	client := clientAdapter.NewClient(callbackAddr, logger)

	notifier := notificationsUC.NewRestNotifier(client, logger)

	inboxCore := inbox.New(inboxRepository, notifier, transactor, logger)
	inboxWait := inboxCore.Start(ctx, inbox.Config{
		Workers:       cfg.Inbox.Workers,
		MaxAttempts:   cfg.Inbox.MaxAttempts,
		BatchSize:     cfg.Inbox.BatchSize,
		FetchPeriod:   cfg.Inbox.FetchPeriod,
		RetryDelay:    cfg.Inbox.RetryDelay,
		InProgressTTL: cfg.Inbox.InProgressTTL,
	})

	wg := &sync.WaitGroup{}
	wg.Go(func() {
		if err = kafka.RunConsumer(
			ctx, cfg.KafkaBrokerAddrs(), cfg.Kafka.Topic, cfg.Kafka.ConsumerGroup, inboxRepository, logger,
		); err != nil {
			logger.Error("kafka run failed", zap.Error(err))
			stop()
		}
	})

	api := controller.New(notifier, logger)

	err = runGrpc(ctx, logger, cfg, api)
	if err != nil {
		logger.Error("failed to run grpc server", zap.Error(err))
		stop()
	}

	inboxWait()
	wg.Wait()
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
