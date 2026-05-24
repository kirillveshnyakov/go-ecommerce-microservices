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
		Clients struct {
			CallbackAddr string `env:"CALLBACK_ADDR" envDefault:""`
		}

		GRPC struct {
			Port             string        `env:"GRPC_PORT" envDefault:"50053"`
			GrpcShutdownTime time.Duration `env:"GRPC_SHUTDOWN_TIME" envDefault:"5s"`
		}

		Kafka struct {
			Brokers       string `env:"KAFKA_BROKERS" envDefault:"localhost:9092"`
			Topic         string `env:"KAFKA_NOTIFICATIONS_TOPIC" envDefault:"order_status_notifications"`
			ConsumerGroup string `env:"KAFKA_CONSUMER_GROUP" envDefault:"notifications"`
		}

		TelegramNotifier struct {
			Token    string `env:"TELEGRAM_TOKEN" envDefault:""`
			ClientID int64  `env:"TELEGRAM_CHAT_ID" envDefault:""`
		}

		Inbox struct {
			Workers       int           `env:"INBOX_WORKERS" envDefault:"2"`
			MaxAttempts   int           `env:"INBOX_MAX_ATTEMPTS" envDefault:"20"`
			BatchSize     int           `env:"INBOX_BATCH_SIZE" envDefault:"5"`
			FetchPeriod   time.Duration `env:"INBOX_FETCH_PERIOD" envDefault:"200ms"`
			RetryDelay    time.Duration `env:"INBOX_RETRY_DELAY" envDefault:"1s"`
			InProgressTTL time.Duration `env:"INBOX_IN_PROGRESS_TTL" envDefault:"30s"`
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
