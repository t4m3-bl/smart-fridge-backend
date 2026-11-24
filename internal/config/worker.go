package config

import (
	"fmt"
	"time"
)

// WorkerConfig intentionally excludes HTTP, authentication, and image storage.
type WorkerConfig struct {
	App      WorkerApp `mapstructure:"app"`
	Database Database  `mapstructure:"database"`
	Push     Push      `mapstructure:"push"`
}
type WorkerApp struct {
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" validate:"required,gt=0"`
}

func LoadWorker() (*WorkerConfig, error) {
	cfg, err := loadConfig[WorkerConfig]()
	if err != nil {
		return nil, err
	}
	if err = cfg.Push.Validate(); err != nil {
		return nil, fmt.Errorf("invalid worker config: %w", err)
	}
	return cfg, nil
}
