package config

import (
	"fmt"
	"net"
	"time"

	"github.com/caarlos0/env/v10"
)

type (
	Config struct {
		Clients struct {
			LOMSGrpcAddr string `env:"LOMS_GRPC_ADDR" envDefault:"localhost:50052"`
		}

		GRPC struct {
			Port             string        `env:"GRPC_PORT" envDefault:"50051"`
			GatewayPort      string        `env:"GRPC_GATEWAY_PORT" envDefault:"8080"`
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
	}
)

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
