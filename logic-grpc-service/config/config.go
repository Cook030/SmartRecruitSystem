package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	MySQL MySQLConfig `yaml:"mysql"`
	GRPC  GRPCConfig  `yaml:"grpc"`
	OSS   OSSConfig   `yaml:"oss"`
	AI    AIConfig    `yaml:"ai"`
	Redis RedisConfig `yaml:"redis"`
}

type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type GRPCConfig struct {
	Port int `yaml:"port"`
}

type OSSConfig struct {
	Endpoint        string `yaml:"endpoint"`
	AccessKeyID     string `yaml:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret"`
	BucketName      string `yaml:"bucket_name"`
}

type AIConfig struct {
	TokenBudget int    `yaml:"token_budget"` // Per request, across all model calls.
	APIKey      string `yaml:"api_key"`
	BaseURL     string `yaml:"base_url"`
	Model       string `yaml:"model"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

var GlobalConfig *Config

func LoadConfig(path string) (*Config, error) {
	// Process environment wins, then service-local .env, then repository .env.
	for _, envPath := range []string{filepath.Join(filepath.Dir(path), ".env"), filepath.Join(filepath.Dir(path), "..", ".env")} {
		if err := godotenv.Load(envPath); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("could not load environment file %s", envPath)
		}
	}
	cfg := Config{
		MySQL: MySQLConfig{Host: "127.0.0.1", Port: 3306, User: "root", Database: "smart_recruit"},
		GRPC:  GRPCConfig{Port: 50051},
		AI:    AIConfig{BaseURL: "https://api.deepseek.com/v1", Model: "deepseek-chat"},
		Redis: RedisConfig{Host: "127.0.0.1", Port: 6379},
	}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	for key, dest := range map[string]*string{
		"MYSQL_HOST": &cfg.MySQL.Host, "MYSQL_USER": &cfg.MySQL.User,
		"MYSQL_PASSWORD": &cfg.MySQL.Password, "MYSQL_DATABASE": &cfg.MySQL.Database,
		"OSS_ENDPOINT": &cfg.OSS.Endpoint, "OSS_ACCESS_KEY_ID": &cfg.OSS.AccessKeyID,
		"OSS_ACCESS_KEY_SECRET": &cfg.OSS.AccessKeySecret, "OSS_BUCKET_NAME": &cfg.OSS.BucketName,
		"AI_API_KEY": &cfg.AI.APIKey, "AI_BASE_URL": &cfg.AI.BaseURL, "AI_MODEL": &cfg.AI.Model,
		"REDIS_HOST": &cfg.Redis.Host, "REDIS_PASSWORD": &cfg.Redis.Password,
	} {
		if value, ok := os.LookupEnv(key); ok {
			*dest = value
		}
	}
	for key, dest := range map[string]*int{"MYSQL_PORT": &cfg.MySQL.Port, "GRPC_PORT": &cfg.GRPC.Port, "REDIS_PORT": &cfg.Redis.Port, "REDIS_DB": &cfg.Redis.DB} {
		if value, ok := os.LookupEnv(key); ok {
			n, err := strconv.Atoi(value)
			if err != nil || n < 0 || (key != "REDIS_DB" && (n == 0 || n > 65535)) {
				return nil, fmt.Errorf("invalid integer setting %s", key)
			}
			*dest = n
		}
	}
	if value, ok := os.LookupEnv("AI_TOKEN_BUDGET"); ok {
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid integer setting AI_TOKEN_BUDGET")
		}
		cfg.AI.TokenBudget = n
	}
	if cfg.AI.TokenBudget < 0 {
		return nil, fmt.Errorf("invalid ai.token_budget")
	}
	GlobalConfig = &cfg
	return &cfg, nil
}
