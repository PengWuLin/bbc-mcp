package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"Server"`
	Database DatabaseConfig `yaml:"Database"`
	Redis    RedisConfig    `yaml:"Redis"`
	BbcTool  BbcToolConfig  `yaml:"BbcTool"`
}

type ServerConfig struct {
	Name string `yaml:"Name"`
	Host string `yaml:"Host"`
	Port int    `yaml:"Port"`
}

type DatabaseConfig struct {
	Host         string `yaml:"Host"`
	Port         int    `yaml:"Port"`
	Name         string `yaml:"Name"`
	User         string `yaml:"User"`
	Password     string `yaml:"Password"`
	MaxOpenConns int    `yaml:"MaxOpenConns"`
	MaxIdleConns int    `yaml:"MaxIdleConns"`
}

type RedisConfig struct {
	Host     string `yaml:"Host"`
	Port     int    `yaml:"Port"`
	Password string `yaml:"Password"`
	DB       int    `yaml:"DB"`
}

type BbcToolConfig struct {
	Path    string `yaml:"Path"`
	Timeout int    `yaml:"Timeout"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("config: 读取配置文件失败: %v", err)
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		log.Printf("config: 解析配置失败: %v", err)
		return nil, err
	}
	return cfg, nil
}
