package config

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"goddns/internal/log"
)

// CloudflareConfig Cloudflare specific settings
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

// IPSource source for obtaining IP
type IPSource struct {
	Interface string   `json:"interface,omitempty"`
	URL       string   `json:"url,omitempty"`       // 保持原有字段兼容性
	URLs      []string `json:"urls,omitempty"`      // 新增数组字段支持多个URL
}

// Config main configuration structure
type Config struct {
	Provider   string           `json:"provider"`
	GetIP      IPSource         `json:"get_ip"`
	WorkDir    string           `json:"work_dir"`
	Proxy      string           `json:"proxy,omitempty"`
	LogOutput  string           `json:"log_output,omitempty"`   // 日志输出配置: shell或文件路径
	Cloudflare CloudflareConfig `json:"provider_options"`
}

// ReadConfig reads and validates config, writes back standardized JSON if needed
func ReadConfig(path string, quiet bool) (Config, string) {
	config := Config{}
	configFile, err := filepath.Abs(path)
	if err != nil {
		return config, ""
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return config, ""
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return config, ""
	}

	// 直接明文处理，无需解密

	if config.Provider == "" {
		return config, ""
	}
	if config.Provider != "cloudflare" {
		return config, ""
	}

	// 检查IP源配置，同时支持interface和urls/url字段
	hasInterface := config.GetIP.Interface != ""
	hasURL := config.GetIP.URL != "" || len(config.GetIP.URLs) > 0

	if !hasInterface && !hasURL {
		return config, ""
	}

	if config.Cloudflare.APIToken == "" {
		return config, ""
	}
	if config.Cloudflare.Domain.Zone == "" || config.Cloudflare.Domain.Record == "" {
		return config, ""
	}

	changed := false

	if config.Proxy != "" {
		pu, err := url.Parse(config.Proxy)
		if err != nil || pu.Scheme == "" {
			log.Fatal("Config 'proxy' must include scheme, e.g., 'socks5://127.0.0.1:1080' or 'http://127.0.0.1:8080'")
		}
		scheme := strings.ToLower(pu.Scheme)
		if scheme != "http" && scheme != "https" && scheme != "socks5" && scheme != "socks5h" {
			log.Fatal("Unsupported proxy scheme '%s' in config.proxy. Supported: http, https, socks5, socks5h", pu.Scheme)
		}
		changed = true
	}
	if config.Cloudflare.TTL == 0 {
		config.Cloudflare.TTL = 180
		changed = true
	}
	if config.WorkDir == "" {
		config.WorkDir = ""
		changed = true
	}
	if !config.Cloudflare.Proxied {
		changed = true
	}

	if changed {
		if writeErr := WriteConfig(configFile, config); writeErr != nil {
			log.Error("Warning: Failed to standardize config file %s. Error: %v", configFile, writeErr)
		}
	}

	return config, configFile
}

// WriteConfig writes config to the given path
func WriteConfig(path string, config Config) error {
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

// 已移除加密/解密逻辑，API 字段直接明文存储

// WriteLastIP writes the ip to cache file
func WriteLastIP(path string, ip string) error {
	return os.WriteFile(path, []byte(ip), 0644)
}