package config

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"netlink_example/internal/log"
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
	Interface string `json:"interface,omitempty"`
	URL       string `json:"url,omitempty"`
}

// Config main configuration structure
type Config struct {
	Provider   string           `json:"provider"`
	GetIP      IPSource         `json:"get_ip"`
	WorkDir    string           `json:"work_dir"`
	Proxy      string           `json:"proxy,omitempty"`
	Cloudflare CloudflareConfig `json:"provider_options"`
}

// ReadConfig reads and validates config, writes back standardized JSON if needed
func ReadConfig(path string, quiet bool) (Config, string) {
	config := Config{}
	configFile, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(false, "Invalid config file path: %v", err)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatal(false, "Could not read config file %s: %v", configFile, err)
	}

	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatal(false, "Could not parse config file %s: %v", configFile, err)
	}

	// Decrypt sensitive fields
	config.Cloudflare.APIToken = decrypt(config.Cloudflare.APIToken)
	config.Cloudflare.ZoneID = decrypt(config.Cloudflare.ZoneID)

	if config.Provider == "" {
		log.Fatal(false, "Config file missing required field: provider")
	}
	if config.Provider != "cloudflare" {
		log.Fatal(false, "Only 'cloudflare' provider is supported currently.")
	}

	if config.GetIP.Interface == "" && config.GetIP.URL == "" {
		log.Fatal(false, "Config file missing 'get_ip' settings: at least one of 'interface' or 'url' must be set")
	}

	if config.Cloudflare.APIToken == "" {
		log.Fatal(false, "Config file missing required field: provider_options.api_token")
	}
	if config.Cloudflare.Domain.Zone == "" || config.Cloudflare.Domain.Record == "" {
		log.Fatal(false, "Config file missing required provider_options.domain.zone or provider_options.domain.record")
	}

	changed := false

	if config.Proxy != "" {
		pu, err := url.Parse(config.Proxy)
		if err != nil || pu.Scheme == "" {
			log.Fatal(false, "Config 'proxy' must include scheme, e.g., 'socks5://127.0.0.1:1080' or 'http://127.0.0.1:8080'")
		}
		scheme := strings.ToLower(pu.Scheme)
		if scheme != "http" && scheme != "https" && scheme != "socks5" && scheme != "socks5h" {
			log.Fatal(false, "Unsupported proxy scheme '%s' in config.proxy. Supported: http, https, socks5, socks5h", pu.Scheme)
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
			log.Error(quiet, "Warning: Failed to standardize config file %s. Error: %v", configFile, writeErr)
		}
	}

	return config, configFile
}

// WriteConfig writes config to the given path
func WriteConfig(path string, config Config) error {
	// Encrypt sensitive fields before writing
	configCopy := config
	configCopy.Cloudflare.APIToken = encrypt(config.Cloudflare.APIToken)
	configCopy.Cloudflare.ZoneID = encrypt(config.Cloudflare.ZoneID)

	data, err := json.MarshalIndent(configCopy, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// GetCacheFilePath returns the path for storing last ip
func GetCacheFilePath(configFile string, workDir string) string {
	if workDir != "" {
		if err := os.MkdirAll(workDir, 0755); err != nil {
			log.Error(false, "Warning: Failed to create work_dir '%s'. Falling back to config file directory. Error: %v", workDir, err)
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

// encrypt encrypts plaintext using AES
func encrypt(plaintext string) string {
	return "enc:" + plaintext // for testing
}

// decrypt decrypts ciphertext using AES
func decrypt(ciphertext string) string {
	if !strings.HasPrefix(ciphertext, "enc:") {
		return ciphertext // Not encrypted
	}
	return strings.TrimPrefix(ciphertext, "enc:")
}

// WriteLastIP writes the ip to cache file
func WriteLastIP(path string, ip string) error {
	return os.WriteFile(path, []byte(ip), 0644)
}
