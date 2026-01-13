package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	xnet "golang.org/x/net/proxy"

	"netlink_example/internal/config"
	"netlink_example/internal/log"
)

// CloudflareProvider implements Cloudflare-specific logic
type CloudflareProvider struct {
	Config config.Config
}

const (
	cloudflareAPI  = "https://api.cloudflare.com/client/v4"
	zonesEndpoint  = cloudflareAPI + "/zones"
	defaultRetries = 3
	baseDelay      = 1 * time.Second
)

// NewProvider constructor
func NewProvider(cfg config.Config) *CloudflareProvider {
	return &CloudflareProvider{Config: cfg}
}

// cfRequest with retry
func (p *CloudflareProvider) cfRequest(method string, endpoint string, data interface{}) (*http.Response, error) {
	var body io.Reader
	if data != nil {
		jsonBody, _ := json.Marshal(data)
		body = bytes.NewBuffer(jsonBody)
	}

	for attempt := 0; attempt <= defaultRetries; attempt++ {
		req, err := http.NewRequest(method, endpoint, body)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+p.Config.Cloudflare.APIToken)
		req.Header.Set("Content-Type", "application/json")

		transport := &http.Transport{}
		if p.Config.Proxy != "" {
			u, err := url.Parse(p.Config.Proxy)
			if err != nil || u.Scheme == "" {
				return nil, fmt.Errorf("invalid proxy URL '%s': must include scheme (e.g. 'http://', 'https://', 'socks5://')", p.Config.Proxy)
			}

			switch strings.ToLower(u.Scheme) {
			case "http", "https":
				transport.Proxy = http.ProxyURL(u)
			case "socks5", "socks5h":
				var auth *xnet.Auth
				if u.User != nil {
					pw, _ := u.User.Password()
					auth = &xnet.Auth{User: u.User.Username(), Password: pw}
				}
				dialer, err := xnet.SOCKS5("tcp", u.Host, auth, xnet.Direct)
				if err != nil {
					return nil, fmt.Errorf("failed to create socks5 dialer: %w", err)
				}
				transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
					return dialer.Dial(network, addr)
				}
			default:
				return nil, fmt.Errorf("unsupported proxy scheme '%s' in proxy url", u.Scheme)
			}
		}
		client := &http.Client{Timeout: 15 * time.Second, Transport: transport}
		resp, err := client.Do(req)
		if err != nil {
			if attempt == defaultRetries {
				return nil, fmt.Errorf("API request failed after %d retries: %w", defaultRetries, err)
			}
			delay := baseDelay * time.Duration(1<<attempt)
			log.Info(false, "API request failed (attempt %d/%d): %v. Retrying in %v...", attempt+1, defaultRetries, err, delay)
			time.Sleep(delay)
			continue
		}

		if resp.StatusCode >= 500 && attempt < defaultRetries {
			log.Info(false, "Server error (5xx) on attempt %d/%d. Retrying in %v...", attempt+1, defaultRetries, baseDelay*time.Duration(1<<attempt))
			time.Sleep(baseDelay * time.Duration(1<<attempt))
			continue
		}

		return resp, nil
	}
	return nil, fmt.Errorf("max retries exceeded")
}

// GetZoneID returns the Cloudflare Zone ID for the configured zone
func (p *CloudflareProvider) GetZoneID(cfg config.Config) (string, error) {
	reqURL := zonesEndpoint + "?name=" + cfg.Cloudflare.Domain.Zone
	resp, err := p.cfRequest("GET", reqURL, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
		Result  []struct {
			ID string `json:"id"`
		} `json:"result"`
		Errors []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode Zone ID response: %w", err)
	}

	if !result.Success || len(result.Result) == 0 {
		errMsg := "unknown error"
		if len(result.Errors) > 0 {
			errMsg = fmt.Sprintf("Code %d: %s", result.Errors[0].Code, result.Errors[0].Message)
		}
		return "", fmt.Errorf("failed to find zone %s. API error: %s", cfg.Cloudflare.Domain.Zone, errMsg)
	}

	return result.Result[0].ID, nil
}

// UpsertDNSRecord creates or updates the DNS record
func (p *CloudflareProvider) UpsertDNSRecord(cfg config.Config, ip string, zoneID string) bool {
	fqdn := cfg.Cloudflare.Domain.Record + "." + cfg.Cloudflare.Domain.Zone
	recordType := "AAAA"

	searchURL := fmt.Sprintf("%s/%s/dns_records?type=%s&name=%s", zonesEndpoint, zoneID, recordType, fqdn)
	resp, err := p.cfRequest("GET", searchURL, nil)
	if err != nil {
		log.Error(false, "Failed to search existing DNS record: %v", err)
		return false
	}
	defer resp.Body.Close()

	var searchResult struct {
		Success bool `json:"success"`
		Result  []struct {
			ID      string `json:"id"`
			Content string `json:"content"`
			Proxied bool   `json:"proxied"`
			TTL     int    `json:"ttl"`
		} `json:"result"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		log.Error(false, "Failed to decode DNS search response: %v", err)
		return false
	}

	if !searchResult.Success {
		log.Error(false, "DNS search failed. API error: %s", searchResult.Errors[0].Message)
		return false
	}

	newRecordData := map[string]interface{}{
		"type":    recordType,
		"name":    fqdn,
		"content": ip,
		"ttl":     cfg.Cloudflare.TTL,
		"proxied": cfg.Cloudflare.Proxied,
	}

	var method, apiEndpoint string

	if len(searchResult.Result) > 0 {
		existing := searchResult.Result[0]
		if existing.Content == ip && existing.Proxied == cfg.Cloudflare.Proxied && existing.TTL == cfg.Cloudflare.TTL {
			log.Info(false, "DNS record is already up-to-date. No API call needed.")
			return true
		}
		recordID := existing.ID
		method = "PUT"
		apiEndpoint = fmt.Sprintf("%s/%s/dns_records/%s", zonesEndpoint, zoneID, recordID)
		log.Info(false, "Existing record found (ID: %s). Content or configuration changed. Initiating update.", recordID)
	} else {
		method = "POST"
		apiEndpoint = fmt.Sprintf("%s/%s/dns_records", zonesEndpoint, zoneID)
		log.Info(false, "No existing record found. Initiating creation.")
	}

	resp, err = p.cfRequest(method, apiEndpoint, newRecordData)
	if err != nil {
		log.Error(false, "API call failed during %s: %v", method, err)
		return false
	}
	defer resp.Body.Close()

	var updateResult struct {
		Success bool `json:"success"`
		Errors  []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&updateResult); err != nil {
		log.Error(false, "Failed to decode API %s response: %v", method, err)
		return false
	}

	if !updateResult.Success {
		errMsg := updateResult.Errors[0].Message
		log.Error(false, "Cloudflare API %s failed (Code %d): %s", method, updateResult.Errors[0].Code, errMsg)
		return false
	}

	log.Info(false, "DNS record successfully %sed to IP %s. TTL: %ds.", strings.ToLower(method), ip, cfg.Cloudflare.TTL)
	return true
}
