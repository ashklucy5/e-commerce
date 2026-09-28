package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv            string
	HTTPAddr          string
	ServerlessRuntime bool

	TrustedProxies []string

	PostgresHost     string
	PostgresPort     uint16
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string
	PostgresSSLMode  string

	// RedisURL is used only for production managed Redis.
	//
	// Production:
	//   APP_ENV=production
	//   APP_REDIS_URL=rediss://...
	//
	// Local/Docker:
	//   APP_ENV=development
	//   REDIS_ADDR=redis:6379
	//   REDIS_PASSWORD=
	//   REDIS_DB=0
	RedisURL      string
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	ShutdownTimeout time.Duration

	PaymentExpiryInterval  time.Duration
	PaymentExpiryBatchSize int

	CODEnabled          bool
	BKashEnabled        bool
	NagadEnabled        bool
	RocketEnabled       bool
	BankTransferEnabled bool
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil &&
		!os.IsNotExist(err) {

		return Config{},
			fmt.Errorf(
				"load .env: %w",
				err,
			)
	}

	appEnv :=
		strings.ToLower(
			strings.TrimSpace(
				getEnv(
					"APP_ENV",
					"development",
				),
			),
		)

	/*
		Vercel and other managed runtimes provide the HTTP listener
		port through PORT.

		Validate it explicitly here so an invalid deployment value
		fails during configuration loading rather than later inside
		the HTTP server.
	*/
	if port :=
		strings.TrimSpace(
			os.Getenv(
				"PORT",
			),
		); port != "" {

		parsedPort, err :=
			strconv.ParseUint(
				port,
				10,
				16,
			)

		if err != nil ||
			parsedPort == 0 {

			return Config{},
				fmt.Errorf(
					"PORT must be an integer between 1 and 65535",
				)
		}
	}

	/*
		PostgreSQL has two application runtime modes.

		Local / Docker / tests:
			POSTGRES_HOST
			POSTGRES_PORT
			POSTGRES_DB
			POSTGRES_USER
			POSTGRES_PASSWORD
			POSTGRES_SSLMODE

		Production runtime:
			APP_POSTGRES_HOST
			APP_POSTGRES_PORT
			APP_POSTGRES_DB
			APP_POSTGRES_USER
			APP_POSTGRES_PASSWORD
			APP_POSTGRES_SSLMODE

		PROD_POSTGRES_* is intentionally NOT read here.

		PROD_POSTGRES_* belongs exclusively to Atlas and other
		explicit production database administration/migration
		operations.

		This separation allows the application runtime to use a
		Neon pooled endpoint while Atlas uses the direct Neon
		endpoint.
	*/

	postgresPrefix :=
		"POSTGRES_"

	postgresDefaultSSLMode :=
		"disable"

	if appEnv == "production" {
		postgresPrefix =
			"APP_POSTGRES_"

		postgresDefaultSSLMode =
			"require"
	}

	postgresHost, err :=
		requireEnv(
			postgresPrefix +
				"HOST",
		)
	if err != nil {
		return Config{}, err
	}

	postgresDB, err :=
		requireEnv(
			postgresPrefix +
				"DB",
		)
	if err != nil {
		return Config{}, err
	}

	postgresUser, err :=
		requireEnv(
			postgresPrefix +
				"USER",
		)
	if err != nil {
		return Config{}, err
	}

	postgresPassword, err :=
		requireEnv(
			postgresPrefix +
				"PASSWORD",
		)
	if err != nil {
		return Config{}, err
	}

	postgresPortValue, err :=
		strconv.ParseUint(
			getEnv(
				postgresPrefix+
					"PORT",
				"5432",
			),
			10,
			16,
		)
	if err != nil {
		return Config{},
			fmt.Errorf(
				"invalid %sPORT: %w",
				postgresPrefix,
				err,
			)
	}

	postgresSSLMode :=
		strings.TrimSpace(
			getEnv(
				postgresPrefix+
					"SSLMODE",
				postgresDefaultSSLMode,
			),
		)

	if postgresSSLMode == "" {
		return Config{},
			fmt.Errorf(
				"%sSSLMODE must not be empty",
				postgresPrefix,
			)
	}

	if appEnv == "production" {
		switch strings.ToLower(
			postgresSSLMode,
		) {
		case "require",
			"verify-ca",
			"verify-full":
			// Accepted secure production modes.

		default:
			return Config{},
				fmt.Errorf(
					"APP_POSTGRES_SSLMODE must be require, verify-ca, or verify-full when APP_ENV=production",
				)
		}
	}

	/*
		Redis has two intentional operating modes.

		Production:
			APP_ENV=production
			APP_REDIS_URL=rediss://...

		Local/Docker and tests:
			REDIS_ADDR=redis:6379
			REDIS_PASSWORD=
			REDIS_DB=0

		APP_REDIS_URL is intentionally used only in production.

		There is intentionally no REDIS_URL fallback.
	*/

	redisURL :=
		""

	redisDB :=
		0

	if appEnv == "production" {
		redisURL =
			strings.TrimSpace(
				os.Getenv(
					"APP_REDIS_URL",
				),
			)

		if redisURL == "" {
			return Config{},
				fmt.Errorf(
					"APP_REDIS_URL is required when APP_ENV=production",
				)
		}

		if !strings.HasPrefix(
			strings.ToLower(
				redisURL,
			),
			"rediss://",
		) {
			return Config{},
				fmt.Errorf(
					"APP_REDIS_URL must use rediss:// when APP_ENV=production",
				)
		}
	} else {
		parsedRedisDB, err :=
			strconv.Atoi(
				getEnv(
					"REDIS_DB",
					"0",
				),
			)
		if err != nil {
			return Config{},
				fmt.Errorf(
					"invalid REDIS_DB: %w",
					err,
				)
		}

		if parsedRedisDB < 0 {
			return Config{},
				fmt.Errorf(
					"REDIS_DB must not be negative",
				)
		}

		redisDB =
			parsedRedisDB
	}

	shutdownTimeout, err :=
		time.ParseDuration(
			getEnv(
				"SHUTDOWN_TIMEOUT",
				"10s",
			),
		)
	if err != nil {
		return Config{},
			fmt.Errorf(
				"invalid SHUTDOWN_TIMEOUT: %w",
				err,
			)
	}

	serverlessRuntime, err :=
		getBoolEnv(
			"SERVERLESS_RUNTIME",
			false,
		)
	if err != nil {
		return Config{}, err
	}
	paymentExpiryInterval, err :=
		time.ParseDuration(
			getEnv(
				"PAYMENT_EXPIRY_INTERVAL",
				"30s",
			),
		)
	if err != nil {
		return Config{},
			fmt.Errorf(
				"invalid PAYMENT_EXPIRY_INTERVAL: %w",
				err,
			)
	}

	if paymentExpiryInterval <= 0 {
		return Config{},
			fmt.Errorf(
				"PAYMENT_EXPIRY_INTERVAL must be greater than zero",
			)
	}

	paymentExpiryBatchSize, err :=
		strconv.Atoi(
			getEnv(
				"PAYMENT_EXPIRY_BATCH_SIZE",
				"100",
			),
		)
	if err != nil {
		return Config{},
			fmt.Errorf(
				"invalid PAYMENT_EXPIRY_BATCH_SIZE: %w",
				err,
			)
	}

	if paymentExpiryBatchSize <= 0 ||
		paymentExpiryBatchSize > 1000 {

		return Config{},
			fmt.Errorf(
				"PAYMENT_EXPIRY_BATCH_SIZE must be between 1 and 1000",
			)
	}

	codEnabled, err :=
		getBoolEnv(
			"COD_ENABLED",
			true,
		)
	if err != nil {
		return Config{}, err
	}

	bkashEnabled, err :=
		getBoolEnv(
			"BKASH_ENABLED",
			false,
		)
	if err != nil {
		return Config{}, err
	}

	nagadEnabled, err :=
		getBoolEnv(
			"NAGAD_ENABLED",
			false,
		)
	if err != nil {
		return Config{}, err
	}

	rocketEnabled, err :=
		getBoolEnv(
			"ROCKET_ENABLED",
			false,
		)
	if err != nil {
		return Config{}, err
	}

	bankTransferEnabled, err :=
		getBoolEnv(
			"BANK_TRANSFER_ENABLED",
			false,
		)
	if err != nil {
		return Config{}, err
	}

	cfg :=
		Config{
			AppEnv: appEnv,

			HTTPAddr:          runtimeHTTPAddr(),
			ServerlessRuntime: serverlessRuntime,

			TrustedProxies: splitCSV(
				os.Getenv(
					"TRUSTED_PROXIES",
				),
			),

			PostgresHost: postgresHost,

			PostgresPort: uint16(
				postgresPortValue,
			),

			PostgresDB: postgresDB,

			PostgresUser: postgresUser,

			PostgresPassword: postgresPassword,

			PostgresSSLMode: postgresSSLMode,

			RedisURL: redisURL,

			RedisAddr: getEnv(
				"REDIS_ADDR",
				"localhost:6379",
			),

			RedisPassword: os.Getenv(
				"REDIS_PASSWORD",
			),

			RedisDB: redisDB,

			ShutdownTimeout: shutdownTimeout,

			PaymentExpiryInterval: paymentExpiryInterval,

			PaymentExpiryBatchSize: paymentExpiryBatchSize,

			CODEnabled: codEnabled,

			BKashEnabled: bkashEnabled,

			NagadEnabled: nagadEnabled,

			RocketEnabled: rocketEnabled,

			BankTransferEnabled: bankTransferEnabled,
		}

	return cfg, nil
}

func (c Config) PostgresURL() string {
	u :=
		&url.URL{
			Scheme: "postgres",

			User: url.UserPassword(
				c.PostgresUser,
				c.PostgresPassword,
			),

			Host: net.JoinHostPort(
				c.PostgresHost,
				strconv.Itoa(
					int(
						c.PostgresPort,
					),
				),
			),

			Path: "/" +
				c.PostgresDB,
		}

	query :=
		u.Query()

	query.Set(
		"sslmode",
		c.PostgresSSLMode,
	)

	u.RawQuery =
		query.Encode()

	return u.String()
}

func runtimeHTTPAddr() string {
	port :=
		strings.TrimSpace(
			os.Getenv(
				"PORT",
			),
		)

	if port == "" {
		return getEnv(
			"HTTP_ADDR",
			":8080",
		)
	}

	/*
		PORT has already been validated by Load.

		Parsing it again here keeps this helper safe if it is exercised
		directly by tests or another caller in the config package.
	*/
	parsedPort, err :=
		strconv.ParseUint(
			port,
			10,
			16,
		)
	if err != nil ||
		parsedPort == 0 {

		return ":" + port
	}

	return ":" +
		strconv.FormatUint(
			parsedPort,
			10,
		)
}

func getEnv(
	key string,
	fallback string,
) string {
	value :=
		os.Getenv(
			key,
		)

	if value == "" {
		return fallback
	}

	return value
}

func getBoolEnv(
	key string,
	fallback bool,
) (bool, error) {
	value :=
		os.Getenv(
			key,
		)

	if value == "" {
		return fallback, nil
	}

	parsed, err :=
		strconv.ParseBool(
			value,
		)
	if err != nil {
		return false,
			fmt.Errorf(
				"invalid %s: %w",
				key,
				err,
			)
	}

	return parsed, nil
}

func requireEnv(
	key string,
) (string, error) {
	value :=
		strings.TrimSpace(
			os.Getenv(
				key,
			),
		)

	if value == "" {
		return "",
			fmt.Errorf(
				"required environment variable %s is not set",
				key,
			)
	}

	return value, nil
}
