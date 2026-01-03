package config

import (
	"os"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	ServiceHost string
	ServicePort int
	MinIOHost   string
	MinIOPort   string
}

func NewConfig() (*Config, error) {
	var err error

	configName := "config"
	_ = godotenv.Load()
	if os.Getenv("CONFIG_NAME") != "" {
		configName = os.Getenv("CONFIG_NAME")
	}

	viper.SetConfigName(configName)
	viper.SetConfigType("toml")
	viper.AddConfigPath("config")
	viper.AddConfigPath(".")
	viper.WatchConfig()

	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = viper.Unmarshal(cfg)
	if err != nil {
		return nil, err
	}

	// MinIO configuration from environment
	cfg.MinIOHost = os.Getenv("MINIO_HOST")
	if cfg.MinIOHost == "" {
		cfg.MinIOHost = "127.0.0.1"
	}
	cfg.MinIOPort = os.Getenv("MINIO_PORT")
	if cfg.MinIOPort == "" {
		cfg.MinIOPort = "9000"
	}

	log.Info("config parsed")

	return cfg, nil
}
