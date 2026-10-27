package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"smart-fridge-backend/internal/domain"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	App      App      `mapstructure:"app"`
	HTTP     HTTP     `mapstructure:"http"`
	Database Database `mapstructure:"database"`
	Auth     Auth     `mapstructure:"auth"`
	Crypto   Crypto   `mapstructure:"crypto"`
}

type FrontendURL string

type App struct {
	Env             Env           `mapstructure:"env"`
	FrontendURL     FrontendURL   `mapstructure:"frontend_url" validate:"required,url"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" validate:"required,gt=0"`
	Version         string        `mapstructure:"version" validate:"required"`
}

type HTTP struct {
	Addr            string        `mapstructure:"addr" validate:"required"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" validate:"required,gt=0"`
	CORSOrigins     []string      `mapstructure:"cors_origins"`
}

type Database struct {
	URL string `mapstructure:"url" validate:"required"`
}

type Auth struct {
	JWTSecret string `mapstructure:"jwt_secret" validate:"required,min=32"`
	JWT       JWT    `mapstructure:"jwt"`
	OAuth     OAuth  `mapstructure:"oauth"`
}

type JWT struct {
	AccessTTL  time.Duration `mapstructure:"access_ttl" validate:"required,gt=0"`
	RefreshTTL time.Duration `mapstructure:"refresh_ttl" validate:"required,gt=0"`
}

type OAuth struct {
	TokensTTL time.Duration                     `mapstructure:"tokens_ttl" validate:"required,gt=0"`
	Providers map[domain.Provider]OAuthProvider `mapstructure:"providers"`
}

type OAuthProvider struct {
	ClientID     string   `mapstructure:"client_id" validate:"required"`
	ClientSecret string   `mapstructure:"client_secret" validate:"required"`
	RedirectURL  string   `mapstructure:"redirect_url" validate:"omitempty,url"`
	Scopes       []string `mapstructure:"scopes" validate:"required,gt=0"`
}

func (p OAuthProvider) Enabled() bool {
	return p.ClientID != "" && p.ClientSecret != ""
}

type Crypto struct {
	Argon2 Argon2 `mapstructure:"argon2"`
}

type Argon2 struct {
	MemoryKiB uint32 `mapstructure:"memory_kib" validate:"required,gt=0"`
	Time      uint32 `mapstructure:"time" validate:"required,gt=0"`
	Threads   uint8  `mapstructure:"threads" validate:"required,gt=0"`
	SaltLen   uint32 `mapstructure:"salt_len" validate:"required,gt=0"`
	KeyLen    uint32 `mapstructure:"key_len" validate:"required,gt=0"`
}

const (
	envPrefix  = "SMART_FRIDGE"
	envFileVar = "SMART_FRIDGE_CONFIG"
)

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	v := viper.New()
	setDefaults(v)

	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := bindEnvAliases(v); err != nil {
		return nil, err
	}

	if path := os.Getenv(envFileVar); path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	err := v.Unmarshal(&cfg, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
		mapstructure.TextUnmarshallerHookFunc(),
	)))
	if err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	validate := validator.New()
	if err := validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if err := cfg.customValidate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) customValidate() error {
	var errs []error

	if c.App.ShutdownTimeout <= c.HTTP.ShutdownTimeout {
		errs = append(errs, errors.New("app.shutdown_timeout must be greater than http.shutdown_timeout"))
	}

	if c.Auth.JWT.RefreshTTL <= c.Auth.JWT.AccessTTL {
		errs = append(errs, errors.New("auth.jwt.refresh_ttl must be greater than auth.jwt.access_ttl"))
	}

	return errors.Join(errs...)
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.env", "dev")
	v.SetDefault("app.frontend_url", "http://localhost:3000")
	v.SetDefault("app.shutdown_timeout", "15s")
	v.SetDefault("app.version", "1.0.0")

	v.SetDefault("http.addr", ":8080")
	v.SetDefault("http.shutdown_timeout", "10s")
	v.SetDefault("http.cors_origins", []string{})

	v.SetDefault("database.url", "")

	v.SetDefault("auth.jwt_secret", "")
	v.SetDefault("auth.jwt.access_ttl", "15m")
	v.SetDefault("auth.jwt.refresh_ttl", "720h")

	v.SetDefault("auth.oauth.tokens_ttl", "10m")

	v.SetDefault("auth.oauth.providers.yandex.client_id", "")
	v.SetDefault("auth.oauth.providers.yandex.client_secret", "")
	v.SetDefault("auth.oauth.providers.yandex.redirect_url", "http://localhost:8080/api/v1/auth/yandex/callback")
	v.SetDefault("auth.oauth.providers.yandex.scopes", []string{"login:email"})

	v.SetDefault("crypto.argon2.memory_kib", 65536)
	v.SetDefault("crypto.argon2.time", 3)
	v.SetDefault("crypto.argon2.threads", 4)
	v.SetDefault("crypto.argon2.salt_len", 16)
	v.SetDefault("crypto.argon2.key_len", 32)
}

func bindEnvAliases(v *viper.Viper) error {
	aliases := map[string]string{
		"auth.oauth.providers.yandex.client_id":     envPrefix + "_OAUTH_YANDEX_CLIENT_ID",
		"auth.oauth.providers.yandex.client_secret": envPrefix + "_OAUTH_YANDEX_CLIENT_SECRET",
	}

	for key, env := range aliases {
		if err := v.BindEnv(key, env); err != nil {
			return fmt.Errorf("bind env %s: %w", env, err)
		}
	}

	return nil
}
