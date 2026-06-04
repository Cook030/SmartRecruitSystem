package config

import (
	"os"

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
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	GlobalConfig = &cfg
	return &cfg, nil
}
