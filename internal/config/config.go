package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Mode int

const (
	DEV Mode = iota
	TEST
	PROD
)

func (m Mode) String() string {
	switch m {
	case DEV:
		return "dev"
	case TEST:
		return "test"
	case PROD:
		return "prod"
	default:
		return fmt.Sprintf("Mode(%d)", int(m))
	}
}

func (m *Mode) UnmarshalText(text []byte) error {
	switch strings.ToLower(strings.TrimSpace(string(text))) {
	case "dev":
		*m = DEV
	case "test":
		*m = TEST
	case "prod":
		*m = PROD
	default:
		return fmt.Errorf("unknown mode %q, expected dev, test or prod", text)
	}
	return nil
}

type Config struct {
	Mode     Mode     `mapstructure:"mode"`
	HTTP     HTTP     `mapstructure:"http"`
	Database Database `mapstructure:"database"`
	Auth     Auth     `mapstructure:"auth"`
	Argon2   Argon2   `mapstructure:"argon2"`
}

type Database struct {
	URL string `mapstructure:"url"`
}

type HTTP struct {
	Addr            string        `mapstructure:"addr"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	CORSOrigins     []string      `mapstructure:"cors_origins"`
}

type Auth struct {
	JWTSecret  string        `mapstructure:"jwt_secret"`
	AccessTTL  time.Duration `mapstructure:"access_ttl"`
	RefreshTTL time.Duration `mapstructure:"refresh_ttl"`
	OAuthTTl   time.Duration `mapstructure:"oauth_ttl"`
}

type Argon2 struct {
	MemoryKiB uint32 `mapstructure:"memory_kib"`
	Time      uint32 `mapstructure:"time"`
	Threads   uint8  `mapstructure:"threads"`
	SaltLen   uint32 `mapstructure:"salt_len"`
	KeyLen    uint32 `mapstructure:"key_len"`
}

type Providers struct {
	RedirectURL string   `mapstructure:"redirect_url"`
	Yandex      Provider `mapstructure:"yandex"`
}

type Provider struct {
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	Scopes       []string `mapstructure:"scopes"`
}

const (
	envPrefix       = "SMART_FRIDGE"
	envFileVar      = "SMART_FRIDGE_CONFIG"
	minJWTSecretLen = 32
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

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	var errs []error

	if c.HTTP.Addr == "" {
		errs = append(errs, errors.New("http.addr is required"))
	}
	if c.HTTP.ShutdownTimeout <= 0 {
		errs = append(errs, errors.New("http.shutdown_timeout must be positive"))
	}

	if len(c.Auth.JWTSecret) < minJWTSecretLen {
		errs = append(errs, fmt.Errorf("auth.jwt_secret must be at least %d bytes (set %s_AUTH_JWT_SECRET)", minJWTSecretLen, envPrefix))
	}
	if c.Database.URL == "" {
		errs = append(errs, fmt.Errorf("database.url is required (set %s_DATABASE_URL)", envPrefix))
	}

	if c.Auth.AccessTTL <= 0 {
		errs = append(errs, errors.New("auth.access_ttl must be positive"))
	}
	if c.Auth.RefreshTTL <= c.Auth.AccessTTL {
		errs = append(errs, errors.New("auth.refresh_ttl must be greater than auth.access_ttl"))
	}
	if c.Auth.OAuthTTl <= 0 {
		errs = append(errs, errors.New("auth.oauth_ttl must be positive"))
	}

	if c.Argon2.MemoryKiB == 0 || c.Argon2.Time == 0 || c.Argon2.Threads == 0 ||
		c.Argon2.SaltLen == 0 || c.Argon2.KeyLen == 0 {
		errs = append(errs, errors.New("argon2 parameters must be positive"))
	}

	return errors.Join(errs...)
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("mode", "dev")

	v.SetDefault("http.addr", ":8080")
	v.SetDefault("http.shutdown_timeout", "10s")
	v.SetDefault("http.cors_origins", []string{})

	v.SetDefault("database.url", "")

	v.SetDefault("auth.jwt_secret", "")
	v.SetDefault("auth.access_ttl", "15m")
	v.SetDefault("auth.refresh_ttl", "720h")

	v.SetDefault("argon2.memory_kib", 64*1024)
	v.SetDefault("argon2.time", 3)
	v.SetDefault("argon2.threads", 4)
	v.SetDefault("argon2.salt_len", 16)
	v.SetDefault("argon2.key_len", 32)
}
