package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"goddns/internal/log"
)

// IPSource source for obtaining IP
type IPSource struct {
	Interface string   `json:"interface,omitempty"`
	URL       string   `json:"url,omitempty"`  // 保持向后兼容性
	URLs      []string `json:"urls,omitempty"` // 支持多个 URL
}

// GeneralConfig global configuration settings
type GeneralConfig struct {
	GetIP     IPSource `json:"get_ip"`
	WorkDir   string   `json:"work_dir,omitempty"`
	LogOutput string   `json:"log_output,omitempty"`
	Proxy     string   `json:"proxy,omitempty"` // 全局代理配置
}

// CloudflareRecord Cloudflare provider specific settings
type CloudflareRecord struct {
	APIToken string `json:"api_token"`
	ZoneID   string `json:"zone_id,omitempty"`
	Proxied  bool   `json:"proxied,omitempty"`
	TTL      int    `json:"ttl,omitempty"`
}

// AliyunRecord Aliyun provider specific settings (demo purpose)
type AliyunRecord struct {
	AccessKey    string `json:"access_key"`
	AccessSecret string `json:"access_secret"`
}

// RecordConfig single DNS record configuration
type RecordConfig struct {
	Provider string `json:"provider"`
	Zone     string `json:"zone"`
	Record   string `json:"record"`
	TTL      int    `json:"ttl,omitempty"`
	Proxied  bool   `json:"proxied,omitempty"` // Cloudflare only
	UseProxy bool   `json:"use_proxy,omitempty"`

	// Provider specific credentials
	Cloudflare *CloudflareRecord `json:"cloudflare,omitempty"`
	Aliyun     *AliyunRecord     `json:"aliyun,omitempty"`

	// Legacy fields for backward compatibility
	APIToken     string `json:"api_token,omitempty"`     // Cloudflare
	AccessKey    string `json:"access_key,omitempty"`    // Aliyun
	AccessSecret string `json:"access_secret,omitempty"` // Aliyun
}

// Config main configuration structure
type Config struct {
	General GeneralConfig  `json:"general"`
	Records []RecordConfig `json:"records"`
}

// LegacyConfig old configuration structure for migration
type LegacyConfig struct {
	Provider        string `json:"provider"`
	GetIP           IPSource `json:"get_ip"`
	WorkDir         string `json:"work_dir"`
	Proxy           string           `json:"proxy,omitempty"`
	LogOutput       string           `json:"log_output,omitempty"`
	Cloudflare      CloudflareConfig `json:"provider_options,omitempty"`
	ProviderOptions json.RawMessage  `json:"provider_options,omitempty"`
}

// CloudflareConfig legacy Cloudflare config structure
type CloudflareConfig struct {
	APIToken string `json:"api_token"`
	ZoneID   string `json:"zone_id,omitempty"`
	Proxied  bool   `json:"proxied"`
	TTL      int    `json:"ttl"`
	Domain   struct {
		Zone   string `json:"zone"`
		Record string `json:"record"`
	} `json:"domain"`
}

// ReadConfig reads and validates config, supports both new and legacy formats
func ReadConfig(path string, quiet bool) (*Config, string) {
	configFile, err := filepath.Abs(path)
	if err != nil {
		log.Error("Failed to resolve config path: %v", err)
		return nil, ""
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Error("Failed to read config file: %v", err)
		return nil, ""
	}

	// 尝试解析为新格式
	var config Config
	if err := json.Unmarshal(data, &config); err == nil && config.Records != nil {
		// 新格式，验证配置
		if err := validateConfig(&config); err != nil {
			log.Error("Invalid config: %v", err)
			return nil, ""
		}
		return &config, configFile
	}

	// 回退到旧格式迁移
	legacyConfig, err := migrateLegacyConfig(data)
	if err != nil {
		log.Error("Failed to parse config (neither new nor legacy format): %v", err)
		return nil, ""
	}

	// 自动迁移旧配置到新格式
	config = *legacyConfigToNew(legacyConfig)
	if err := validateConfig(&config); err != nil {
		log.Error("Migrated config validation failed: %v", err)
		return nil, ""
	}

	// 询问用户是否要迁移配置文件（静默模式下自动迁移）
	if !quiet {
		log.Info("Legacy config detected. Consider migrating to new format.")
	}

	return &config, configFile
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	if len(cfg.Records) == 0 {
		return fmt.Errorf("at least one record must be configured")
	}

	// 检查 IP 源配置
	hasInterface := cfg.General.GetIP.Interface != ""
	hasURL := cfg.General.GetIP.URL != "" || len(cfg.General.GetIP.URLs) > 0
	if !hasInterface && !hasURL {
		return fmt.Errorf("either 'get_ip.interface' or 'get_ip.urls' must be configured")
	}

	// 验证全局代理配置
	if cfg.General.Proxy != "" {
		if err := validateProxyURL(cfg.General.Proxy); err != nil {
			return fmt.Errorf("invalid global proxy: %w", err)
		}
	}

	// 验证每个记录
	for i, record := range cfg.Records {
		if record.Provider == "" {
			return fmt.Errorf("record[%d]: provider is required", i)
		}
		if record.Zone == "" {
			return fmt.Errorf("record[%d]: zone is required", i)
		}
		if record.Record == "" {
			return fmt.Errorf("record[%d]: record name is required", i)
		}

		// 验证记录级代理配置
		if record.UseProxy && cfg.General.Proxy == "" {
			return fmt.Errorf("record[%d]: use_proxy is true but no global proxy configured", i)
		}

		// 验证凭证
		switch record.Provider {
		case "cloudflare":
			token := record.APIToken
			if record.Cloudflare != nil && record.Cloudflare.APIToken != "" {
				token = record.Cloudflare.APIToken
			}
			if token == "" {
				return fmt.Errorf("record[%d]: cloudflare api_token is required", i)
			}
		case "aliyun":
			key := record.AccessKey
			secret := record.AccessSecret
			if record.Aliyun != nil {
				if record.Aliyun.AccessKey != "" {
					key = record.Aliyun.AccessKey
				}
				if record.Aliyun.AccessSecret != "" {
					secret = record.Aliyun.AccessSecret
				}
			}
			if key == "" || secret == "" {
				return fmt.Errorf("record[%d]: aliyun access_key and access_secret are required", i)
			}
		default:
			return fmt.Errorf("record[%d]: unsupported provider '%s'", i, record.Provider)
		}
	}

	return nil
}

// validateProxyURL validates proxy URL format
func validateProxyURL(proxyURL string) error {
	if proxyURL == "" {
		return nil
	}
	u, err := url.Parse(proxyURL)
	if err != nil || u.Scheme == "" {
		return fmt.Errorf("proxy must include scheme (e.g., 'socks5://', 'http://')")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" && scheme != "socks5" && scheme != "socks5h" {
		return fmt.Errorf("unsupported proxy scheme '%s'", scheme)
	}
	return nil
}

// migrateLegacyConfig migrates legacy config to new format
func migrateLegacyConfig(data []byte) (*LegacyConfig, error) {
	var legacy LegacyConfig
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, err
	}

	if legacy.Provider == "" || legacy.Provider != "cloudflare" {
		return nil, fmt.Errorf("only 'cloudflare' provider is supported in legacy format")
	}

	return &legacy, nil
}

// legacyConfigToNew converts legacy config to new format
func legacyConfigToNew(legacy *LegacyConfig) *Config {
	ttl := legacy.Cloudflare.TTL
	if ttl == 0 {
		ttl = 180
	}

	return &Config{
		General: GeneralConfig{
			GetIP:     legacy.GetIP,
			WorkDir:   legacy.WorkDir,
			LogOutput: legacy.LogOutput,
			Proxy:     legacy.Proxy,
		},
		Records: []RecordConfig{
			{
				Provider: "cloudflare",
				Zone:     legacy.Cloudflare.Domain.Zone,
				Record:   legacy.Cloudflare.Domain.Record,
				TTL:      ttl,
				Proxied:  legacy.Cloudflare.Proxied,
				UseProxy: legacy.Proxy != "",
				Cloudflare: &CloudflareRecord{
					APIToken: legacy.Cloudflare.APIToken,
					ZoneID:   legacy.Cloudflare.ZoneID,
				},
			},
		},
	}
}

// WriteConfig writes config to the given path
func WriteConfig(path string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// GetCacheFilePath returns the path for storing last ip
func GetCacheFilePath(configFile string, workDir string) string {
	if workDir != "" {
		if err := os.MkdirAll(workDir, 0755); err != nil {
			log.Error("Warning: Failed to create work_dir '%s'. Falling back to config file directory. Error: %v", workDir, err)
			return filepath.Join(filepath.Dir(configFile), "cache.lastip")
		}
		return filepath.Join(workDir, "cache.lastip")
	}
	return filepath.Join(filepath.Dir(configFile), "cache.lastip")
}

// ReadLastIP reads the last IP from cache file
func ReadLastIP(path string) string {
	ip, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(ip))
}

// WriteLastIP writes the ip to cache file
func WriteLastIP(path string, ip string) error {
	return os.WriteFile(path, []byte(ip), 0644)
}

// GetRecordProxy returns the proxy URL for a specific record
// Returns empty string if proxy should not be used
func GetRecordProxy(cfg *Config, record *RecordConfig) string {
	if !record.UseProxy {
		return ""
	}
	return cfg.General.Proxy
}

// GetRecordTTL returns the effective TTL for a record
func GetRecordTTL(record *RecordConfig) int {
	if record.TTL > 0 {
		return record.TTL
	}
	// Default TTL
	if record.Provider == "cloudflare" {
		return 180
	}
	return 600
}
