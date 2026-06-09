package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"

	"bbc-mcp/internal/crypto"
)

type Config struct {
	Server   ServerConfig   `yaml:"Server"`
	Database DatabaseConfig `yaml:"Database"`
	Redis    RedisConfig    `yaml:"Redis"`
	BbcTool  BbcToolConfig  `yaml:"BbcTool"`
	Auth     AuthConfig     `yaml:"Auth"`
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

type AuthConfig struct {
	Tokens []string `yaml:"Tokens"`
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

	if err := cfg.decryptPasswords(); err != nil {
		log.Printf("config: 解密密码失败: %v", err)
		return nil, err
	}

	return cfg, nil
}

func (c *Config) decryptPasswords() error {
	if c.Database.Password != "" {
		pw, err := crypto.DecryptIfNeeded(c.Database.Password)
		if err != nil {
			return fmt.Errorf("解密 Database.Password 失败: %w", err)
		}
		c.Database.Password = pw
	}
	if c.Redis.Password != "" {
		pw, err := crypto.DecryptIfNeeded(c.Redis.Password)
		if err != nil {
			return fmt.Errorf("解密 Redis.Password 失败: %w", err)
		}
		c.Redis.Password = pw
	}
	return nil
}
