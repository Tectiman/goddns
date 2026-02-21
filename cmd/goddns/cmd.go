package main

import (
	"fmt"
	"os"
	"sync"

	"goddns/internal/config"
	"goddns/internal/log"
	"goddns/internal/platform/ifaddr"
	"goddns/internal/provider/aliyun"
	"goddns/internal/provider/cloudflare"
	"github.com/spf13/cobra"
)

var version = "dev"
var commit = ""
var buildDate = ""

func printVersion() {
	fmt.Printf("goddns %s\n", version)
	if commit != "" {
		fmt.Printf("commit: %s\n", commit)
	}
	if buildDate != "" {
		fmt.Printf("built: %s\n", buildDate)
	}
}

var rootCmd = &cobra.Command{
	Use:   "goddns",
	Short: "强大的动态 DNS 客户端 - 支持多域名多服务商",
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "运行 DDNS 更新",
	Run: func(cmd *cobra.Command, args []string) {
		// 解析参数
		configPath, _ := cmd.Flags().GetString("config")
		ignoreCache, _ := cmd.Flags().GetBool("ignore-cache")
		if configPath == "" {
			fmt.Fprintln(os.Stderr, "缺少配置文件参数：--config")
			os.Exit(1)
		}

		// 读取配置
		cfg, absConfigFile := config.ReadConfig(configPath, false)
		if cfg == nil {
			fmt.Fprintln(os.Stderr, "Failed to load configuration")
			os.Exit(1)
		}

		// 初始化日志系统
		if err := log.Init(cfg.General.LogOutput); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize logging: %v\n", err)
			os.Exit(1)
		}

		log.Info("GodDNS starting with %d record(s)", len(cfg.Records))

		// 获取当前 IP 地址（get_ip 始终直连，不使用代理）
		var infos []ifaddr.IPv6Info
		var err error
		if cfg.General.GetIP.Interface != "" {
			infos, err = ifaddr.GetAvailableIPv6(cfg.General.GetIP.Interface)
			if err != nil {
				log.Info("Interface %s failed: %v", cfg.General.GetIP.Interface, err)
				log.Info("Trying fallback API...")
				infos, err = ifaddr.GetIPv6FromAPIs(cfg.General.GetIP.URLs, false)
				if err != nil {
					log.Error("Fallback also failed: %v", err)
					os.Exit(1)
				}
			}
		} else {
			infos, err = ifaddr.GetIPv6FromAPIs(cfg.General.GetIP.URLs, false)
			if err != nil {
				log.Error("Failed to get IP from APIs: %v", err)
				os.Exit(1)
			}
		}

		currentIP, err := ifaddr.SelectBestIPv6(infos)
		if err != nil {
			log.Error("Failed to select best IPv6 address: %v", err)
			os.Exit(1)
		}

		log.Info("Current IPv6 address: %s", currentIP)

		// 检查缓存
		cacheFilePath := config.GetCacheFilePath(absConfigFile, cfg.General.WorkDir)
		lastIP := config.ReadLastIP(cacheFilePath)

		if !ignoreCache {
			if lastIP != "" && lastIP == currentIP {
				log.Info("IP has not changed since last run: %s", currentIP)
			} else if lastIP != "" {
				log.Info("IP changed from %s to %s", lastIP, currentIP)
			}
		}

		// 批量更新所有记录
		updateRecords(cfg, currentIP, cacheFilePath, ignoreCache)
	},
}

// updateRecords updates all DNS records in parallel
func updateRecords(cfg *config.Config, currentIP string, cacheFilePath string, ignoreCache bool) {
	var wg sync.WaitGroup
	results := make([]updateResult, len(cfg.Records))

	for i, record := range cfg.Records {
		wg.Add(1)
		go func(idx int, rec config.RecordConfig) {
			defer wg.Done()
			results[idx] = updateSingleRecord(cfg, &rec, currentIP, cacheFilePath, ignoreCache)
		}(i, record)
	}

	wg.Wait()

	// 汇总结果
	successCount := 0
	failCount := 0
	for _, result := range results {
		if result.success {
			successCount++
		} else {
			failCount++
		}
	}

	log.Info("Update completed: %d succeeded, %d failed", successCount, failCount)

	if failCount > 0 {
		os.Exit(1)
	}
}

type updateResult struct {
	success bool
	err     error
	record  string
}

// updateSingleRecord updates a single DNS record
func updateSingleRecord(cfg *config.Config, record *config.RecordConfig, currentIP string, cacheFilePath string, ignoreCache bool) updateResult {
	result := updateResult{record: fmt.Sprintf("%s.%s", record.Record, record.Zone)}

	log.Info("Processing record: %s (%s)", result.record, record.Provider)

	// 获取记录级代理配置
	proxyURL := config.GetRecordProxy(cfg, record)

	switch record.Provider {
	case "cloudflare":
		// 获取 API Token
		apiToken := record.Cloudflare.APIToken

		// 获取 ZoneID
		zoneID := record.Cloudflare.ZoneID

		// 创建 Cloudflare Provider
		providerConfig := &cloudflare.SimpleConfig{
			Proxy: proxyURL,
		}
		provider := cloudflare.NewProvider(providerConfig, apiToken)

		// 如果 ZoneID 为空，自动获取
		if zoneID == "" {
			log.Info("Zone ID not configured, fetching for zone: %s", record.Zone)
			fetchedZoneID, err := provider.GetZoneID(record.Zone)
			if err != nil {
				log.Error("Failed to fetch Zone ID: %v", err)
				result.err = fmt.Errorf("failed to get Zone ID: %w", err)
				result.success = false
				return result
			}
			zoneID = fetchedZoneID
			log.Info("Zone ID fetched: %s", zoneID)

			// 保存 ZoneID 到配置
			record.Cloudflare.ZoneID = zoneID
			if writeErr := config.WriteConfig(cacheFilePath+".config", cfg); writeErr != nil {
				log.Warning("Warning: Failed to save Zone ID to config: %v", writeErr)
			}
		}

		// 获取 TTL 和 Proxied 设置
		ttl := config.GetRecordTTL(record)
		proxied := record.Proxied
		if record.Cloudflare.TTL > 0 {
			ttl = record.Cloudflare.TTL
		}
		if record.Cloudflare.Proxied {
			proxied = true
		}

		// 更新 DNS 记录
		success, err := provider.UpsertDNSRecord(record.Zone, record.Record, currentIP, zoneID, ttl, proxied)
		if err != nil {
			log.Error("Failed to update %s: %v", result.record, err)
			result.err = err
			result.success = false
			return result
		}

		if success {
			log.Success("Record %s updated successfully", result.record)
			result.success = true
		} else {
			log.Error("Record %s update returned false", result.record)
			result.success = false
		}

	case "aliyun":
		// 获取阿里云凭证
		accessKeyID := record.Aliyun.AccessKeyID
		accessKeySecret := record.Aliyun.AccessKeySecret

		// 创建阿里云 Provider
		provider := aliyun.NewProvider(accessKeyID, accessKeySecret)

		// 获取 TTL 设置
		ttl := config.GetRecordTTL(record)
		if record.Aliyun.TTL > 0 {
			ttl = record.Aliyun.TTL
		}

		// 阿里云不支持代理，忽略 use_proxy 设置
		if proxyURL != "" {
			log.Warning("Aliyun provider does not support proxy, ignoring use_proxy setting")
		}

		// 更新 DNS 记录
		success, err := provider.UpsertDNSRecord(record.Zone, record.Record, currentIP, ttl)
		if err != nil {
			log.Error("Failed to update %s: %v", result.record, err)
			result.err = err
			result.success = false
			return result
		}

		if success {
			log.Success("Record %s updated successfully", result.record)
			result.success = true
		} else {
			log.Error("Record %s update returned false", result.record)
			result.success = false
		}

	default:
		log.Error("Unsupported provider: %s", record.Provider)
		result.err = fmt.Errorf("unsupported provider: %s", record.Provider)
		result.success = false
	}

	// 如果成功，更新缓存
	if result.success {
		if writeErr := config.WriteLastIP(cacheFilePath, currentIP); writeErr != nil {
			log.Warning("Warning: Failed to write IP to cache: %v", writeErr)
		}
	}

	return result
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

func Execute() {
	runCmd.Flags().StringP("config", "f", "", "配置文件路径 (JSON 格式)")
	runCmd.Flags().BoolP("ignore-cache", "i", false, "忽略缓存 IP，强制更新")
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "命令执行失败：%v\n", err)
		os.Exit(1)
	}
}
