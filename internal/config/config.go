package config

import (
	"flag"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/caarlos0/env/v10"
	"gopkg.in/yaml.v3"

	"github.com/gabkaclassic/GitQuest/internal/dto"
)

type (
	Config struct {
		Server Server
		Log    Log
		DB     DB
		Rules  Rules
		Cache  Cache
		Jobs   Jobs
	}
	Server struct {
		Address string `env:"ADDRESS" envDefault:"localhost:8080"`
	}
	DB struct {
		Driver         string `env:"DB_DRIVER" envDefault:"postgres"`
		DSN            string `env:"DATABASE_DSN"`
		MigrationsPath string `env:"DB_MIGRATIONS_PATH" envDefault:"./migrations"`
	}
	Cache struct {
		Address  string `env:"CACHE_ADDRESS" envDefault:"localhost:6379"`
		Username string `env:"CACHE_USER" envDefault:""`
		Password string `env:"CACHE_PASSWORD" envDefault:""`
		DB       int    `env:"CACHE_DB" envDefault:"0"`
	}
	Log struct {
		Level   string `env:"LOG_LEVEL" envDefault:"info"`
		File    string `env:"LOG_FILE"`
		Console bool   `env:"LOG_CONSOLE" envDefault:"false"`
		JSON    bool   `env:"LOG_JSON" envDefault:"true"`
	}
	Rules struct {
		Reeval bool
		Rules  []dto.Rule
	}
	Jobs struct {
		Cleanup Cleanup
	}
	Cleanup struct {
		Timeout  time.Duration `env:"CLEANUP_TIMEOUT" envDefault:"10"`
		Interval time.Duration `env:"CLEANUP_INTERVAL" envDefault:"10"`
	}
)

func defineEnvParsers() map[reflect.Type]env.ParserFunc {
	return map[reflect.Type]env.ParserFunc{
		reflect.TypeOf(time.Duration(0)): func(v string) (any, error) {
			secs, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			return time.Duration(secs) * time.Second, nil
		},
	}
}

func parseRules(path string) (*Rules, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var rules Rules
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&rules); err != nil {
		return nil, err
	}

	return &rules, nil
}

func ParseConfig() (*Config, error) {
	var cfg Config

	parsers := defineEnvParsers()

	if err := env.ParseWithOptions(&cfg, env.Options{FuncMap: parsers}); err != nil {
		return nil, err
	}

	address := flag.String("a", cfg.Server.Address, "HTTP server address")

	rulesFilePath := flag.String("r", "./config/rules.yaml", "File path to rules yaml config")

	logLevel := flag.String("log-level", cfg.Log.Level, "Logging level")
	logFile := flag.String("log-file", cfg.Log.File, "Log file path")
	logConsole := flag.Bool("log-console", cfg.Log.Console, "Enable console logging")
	logJSON := flag.Bool("log-json", cfg.Log.JSON, "Enable JSON output for logs")

	dbDSN := flag.String("d", cfg.DB.DSN, "DSN")
	dbDriver := flag.String("db-driver", cfg.DB.Driver, "Database driver")
	dbMigrationsPath := flag.String("db-migrations-path", cfg.DB.MigrationsPath, "Migrations file path")

	cacheAddress := flag.String("cache-addr", cfg.Cache.Address, "Cache address")
	cacheDB := flag.Int("cache-db", cfg.Cache.DB, "Cache db")
	cacheUsername := flag.String("cache-user", cfg.Cache.Username, "Cache username")
	cachePassword := flag.String("cache-pass", cfg.Cache.Password, "Cache password")

	cleanupJobTimeout := flag.Duration("cleanup-timeout", cfg.Jobs.Cleanup.Timeout, "Cleanup old events background job timeout")
	cleanupJobInterval := flag.Duration("cleanup-interval", cfg.Jobs.Cleanup.Interval, "Cleanup old events background job interval")

	flag.Parse()

	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "a":
			cfg.Server.Address = *address

		case "log-level":
			cfg.Log.Level = *logLevel
		case "log-file":
			cfg.Log.File = *logFile
		case "log-console":
			cfg.Log.Console = *logConsole
		case "log-json":
			cfg.Log.JSON = *logJSON

		case "db-driver":
			cfg.DB.Driver = *dbDriver
		case "d":
			cfg.DB.DSN = *dbDSN
		case "db-migrations-path":
			cfg.DB.MigrationsPath = *dbMigrationsPath

		case "cache-addr":
			cfg.Cache.Address = *cacheAddress
		case "cache-db":
			cfg.Cache.DB = *cacheDB
		case "cache-user":
			cfg.Cache.Username = *cacheUsername
		case "cache-pass":
			cfg.Cache.Password = *cachePassword

		case "cleanup-timeout":
			cfg.Jobs.Cleanup.Timeout = *cleanupJobTimeout
		case "cleanup-interval":
			cfg.Jobs.Cleanup.Interval = *cleanupJobInterval

		}
	})

	rules, err := parseRules(*rulesFilePath)

	if err != nil {
		return nil, err
	}

	cfg.Rules = *rules

	return &cfg, nil
}
