package main

import (
	"fmt"
	"os"

	"goddns/internal/config"
	"goddns/internal/log"
	"goddns/internal/platform/ifaddr"
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
	Short: "强大的动态 DNS 客户端",
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "运行 DDNS 更新",
	Run: func(cmd *cobra.Command, args []string) {
		// 解析参数
		configPath, _ := cmd.Flags().GetString("config")
		ignoreCache, _ := cmd.Flags().GetBool("ignore-cache")
		if configPath == "" {
			fmt.Fprintln(os.Stderr, "缺少配置文件参数: --config")
			os.Exit(1)
		}

		// 读取配置
		importConfig, absConfigFile := config.ReadConfig(configPath, false)
		cfg := importConfig

		// 初始化日志系统
		if err := log.Init(cfg.LogOutput); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize logging: %v\n", err)
			os.Exit(1)
		}

		provider := cloudflare.NewProvider(cfg)

		var infos []ifaddr.IPv6Info
		var err error
		if cfg.GetIP.Interface != "" {
			infos, err = ifaddr.GetAvailableIPv6(cfg.GetIP.Interface)
			if err != nil {
				log.Info("Interface %s failed: %v", cfg.GetIP.Interface, err)
				log.Info("Trying fallback API...")
				infos, err = ifaddr.GetIPv6Fallback(cfg, false)
				if err != nil {
					log.Error("Fallback also failed: %v", err)
					os.Exit(1)
				}
			}
		} else {
			infos, err = ifaddr.GetIPv6Fallback(cfg, false)
			if err != nil {
				log.Error("Fallback failed: %v", err)
				os.Exit(1)
			}
		}

		currentIP, err := ifaddr.SelectBestIPv6(cfg, infos)
		if err != nil {
			log.Error("Failed to select best IPv6 address: %v", err)
			os.Exit(1)
		}

		cacheFilePath := config.GetCacheFilePath(absConfigFile, cfg.WorkDir)
		lastIP := config.ReadLastIP(cacheFilePath)

		if !ignoreCache {
			if lastIP != "" && lastIP == currentIP {
				log.Info("IP has not changed: %s", currentIP)
				return
			}
		}

		zoneID := cfg.Cloudflare.ZoneID
		if zoneID == "" {
			fetchedZoneID, err := provider.GetZoneID(cfg)
			if err != nil {
				log.Error("Error fetching Zone ID: %v", err)
				os.Exit(1)
			}
			cfg.Cloudflare.ZoneID = fetchedZoneID
			zoneID = fetchedZoneID

			if writeErr := config.WriteConfig(absConfigFile, cfg); writeErr != nil {
				log.Warning("Warning: Failed to save Zone ID to config file: %v", writeErr)
			}
		}

		success, err := provider.UpsertDNSRecord(cfg, currentIP, zoneID)

		if success {
			if writeErr := config.WriteLastIP(cacheFilePath, currentIP); writeErr != nil {
				log.Warning("Update succeeded, but failed to write IP to cache: %v", writeErr)
			}
			log.Success("DDNS update successful: %s", currentIP)
		} else {
			log.Error("DDNS update failed: %v", err)
		}
	},
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
		fmt.Fprintf(os.Stderr, "命令执行失败: %v\n", err)
		os.Exit(1)
	}
}
