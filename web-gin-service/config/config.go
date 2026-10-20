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
	Server ServerConfig `yaml:"server"`
	GRPC   GRPCConfig   `yaml:"grpc"`
	JWT    JWTConfig    `yaml:"jwt"`
	CORS   CORSConfig   `yaml:"cors"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type GRPCConfig struct {
	LogicAddr string `yaml:"logic_addr"`
}

type JWTConfig struct {
	Secret string `yaml:"secret"`
	Expire int    `yaml:"expire"`
}

type CORSConfig struct {
	AllowedOrigins string `yaml:"allowed_origins"`
}

var GlobalConfig *Config

func LoadConfig(path string) (*Config, error) {
	// Process environment wins, then service-local .env, then repository .env.
	for _, envPath := range []string{filepath.Join(filepath.Dir(path), ".env"), filepath.Join(filepath.Dir(path), "..", ".env")} {
		if err := godotenv.Load(envPath); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("could not load environment file %s", envPath)
		}
	}
	cfg := Config{Server: ServerConfig{Port: 8080}, GRPC: GRPCConfig{LogicAddr: "127.0.0.1:50051"}, JWT: JWTConfig{Expire: 24}}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	for key, dest := range map[string]*string{"LOGIC_GRPC_ADDR": &cfg.GRPC.LogicAddr, "JWT_SECRET": &cfg.JWT.Secret, "CORS_ALLOWED_ORIGINS": &cfg.CORS.AllowedOrigins} {
		if value, ok := os.LookupEnv(key); ok {
			*dest = value
		}
	}
	for key, dest := range map[string]*int{"WEB_PORT": &cfg.Server.Port, "JWT_EXPIRE": &cfg.JWT.Expire} {
		if value, ok := os.LookupEnv(key); ok {
			n, err := strconv.Atoi(value)
			if err != nil || n <= 0 || (key == "WEB_PORT" && n > 65535) {
				return nil, fmt.Errorf("invalid integer setting %s", key)
			}
			*dest = n
		}
	}
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	GlobalConfig = &cfg
	return &cfg, nil
}
