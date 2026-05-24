package config

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/caarlos0/env/v10"
)

type (
	Config struct {
		GRPC struct {
			Port             string        `env:"GRPC_PORT" envDefault:"50052"`
			GatewayPort      string        `env:"GRPC_GATEWAY_PORT" envDefault:"8081"`
			GrpcShutdownTime time.Duration `env:"GRPC_SHUTDOWN_TIME" envDefault:"5s"`
			HTTPShutdownTime time.Duration `env:"HTTP_SHUTDOWN_TIME" envDefault:"30s"`
		}

		PG struct {
			Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
			Port     string `env:"POSTGRES_PORT" envDefault:"5432"`
			DB       string `env:"POSTGRES_DB" envDefault:"ecommerce"`
			User     string `env:"POSTGRES_USER" envDefault:"ecommerce_user"`
			Password string `env:"POSTGRES_PASSWORD" envDefault:"12345"`
		}

		Clients struct {
			NotificationsGrpcAddr string `env:"NOTIFICATIONS_GRPC_ADDR" envDefault:"localhost:50053"`
		}

		Outbox struct {
			Workers     int           `env:"OUTBOX_WORKERS" envDefault:"3"`
			BatchSize   int           `env:"OUTBOX_BATCH_SIZE" envDefault:"5"`
			FetchPeriod time.Duration `env:"OUTBOX_FETCH_PERIOD" envDefault:"5s"`
			TTL         time.Duration `env:"OUTBOX_IN_PROGRESS_TTL" envDefault:"60s"`
		}

		Kafka struct {
			Brokers string `env:"KAFKA_BROKERS" envDefault:"localhost:9092"`
			Topic   string `env:"KAFKA_NOTIFICATIONS_TOPIC" envDefault:"order_status_notifications"`
		}
	}
)

func (c *Config) KafkaBrokerAddrs() []string {
	parts := strings.Split(c.Kafka.Brokers, ",")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}

	return out
}

func (c *Config) ConstructPostgresURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		c.PG.User,
		c.PG.Password,
		net.JoinHostPort(c.PG.Host, c.PG.Port),
		c.PG.DB,
	)
}

func New() (*Config, error) {
	var cfg Config
	err := env.Parse(&cfg)
	return &cfg, err
}
