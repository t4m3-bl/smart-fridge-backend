package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Mode int

const (
	DEV Mode = iota
	TEST
	PROD
)

type Config struct {
	HTTP   HTTP   `mapstructure:"http"`
	Auth   Auth   `mapstructure:"auth"`
	Argon2 Argon2 `mapstructure:"argon2"`
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
}

type Argon2 struct {
	MemoryKiB uint32 `mapstructure:"memory_kib"`
	Time      uint32 `mapstructure:"time"`
	Threads   uint8  `mapstructure:"threads"`
	SaltLen   uint32 `mapstructure:"salt_len"`
	KeyLen    uint32 `mapstructure:"key_len"`
}

const (
	envPrefix  = ""
	envFileVar = ""
)

func Load() (*Config, error) {
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
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("http.addr", ":8080")
	v.SetDefault("http.shutdown_timeout", "10s")
	v.SetDefault("http.cors_origins", []string{})

	v.SetDefault("auth.jwt_secret", "")
	v.SetDefault("auth.access_ttl", "15m")
	v.SetDefault("auth.refresh_ttl", "720h")

	v.SetDefault("argon2.memory_kib", 64*1024)
	v.SetDefault("argon2.time", 3)
	v.SetDefault("argon2.threads", 4)
	v.SetDefault("argon2.salt_len", 16)
	v.SetDefault("argon2.key_len", 32)
}
