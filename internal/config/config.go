package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"

	"bbc-mcp/internal/crypto"
)

type Config struct {
	Server      ServerConfig              `yaml:"Server"`
	Database    DatabaseConfig            `yaml:"Database"`
	Redis       RedisConfig               `yaml:"Redis"`
	BbcTool     BbcToolConfig             `yaml:"BbcTool"`
	Auth        AuthConfig                `yaml:"Auth"`
	Gateway     GatewayConfig             `yaml:"Gateway"`
	K8sClusters map[string]K8sClusterConfig `yaml:"K8sClusters"`
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

type GatewayConfig struct {
	Mode   string              `yaml:"Mode"`
	Native GatewayNativeConfig `yaml:"Native"`
}

type GatewayNativeConfig struct {
	Namespace   string `yaml:"Namespace"`
	StatefulSet string `yaml:"StatefulSet"`
	Container   string `yaml:"Container"`
	Port        int    `yaml:"Port"`
}

type K8sClusterConfig struct {
	Server   string `yaml:"Server"`   // K8s API server URL, e.g. https://192.168.1.100:6443
	Token    string `yaml:"Token"`    // Bearer token
	CAData   string `yaml:"CAData"`   // CA certificate (base64, optional)
	Insecure bool   `yaml:"Insecure"` // Skip TLS verification
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

	cfg.applyDefaults()

	if err := cfg.decryptSecrets(); err != nil {
		log.Printf("config: 解密敏感数据失败: %v", err)
		return nil, err
	}

	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Gateway.Mode == "" {
		c.Gateway.Mode = "cli"
	}
	if c.Gateway.Native.Namespace == "" {
		c.Gateway.Native.Namespace = "xcentral"
	}
	if c.Gateway.Native.StatefulSet == "" {
		c.Gateway.Native.StatefulSet = "cloudbbc-gateway"
	}
	if c.Gateway.Native.Container == "" {
		c.Gateway.Native.Container = "cloudbbc-goproxy-container"
	}
	if c.Gateway.Native.Port == 0 {
		c.Gateway.Native.Port = 5000
	}
}

func (c *Config) decryptSecrets() error {
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

	for i, token := range c.Auth.Tokens {
		if token != "" {
			decrypted, err := crypto.DecryptIfNeeded(token)
			if err != nil {
				return fmt.Errorf("解密 Auth.Tokens[%d] 失败: %w", i, err)
			}
			c.Auth.Tokens[i] = decrypted
		}
	}

	for name, cluster := range c.K8sClusters {
		if cluster.Token != "" {
			decrypted, err := crypto.DecryptIfNeeded(cluster.Token)
			if err != nil {
				return fmt.Errorf("解密 K8sClusters.%s.Token 失败: %w", name, err)
			}
			cluster.Token = decrypted
			c.K8sClusters[name] = cluster
		}
	}

	return nil
}
