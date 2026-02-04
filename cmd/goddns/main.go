package main

import (
	"flag"
	"fmt"
	"os"

	"goddns/internal/config"
	"goddns/internal/log"
	"goddns/internal/platform/ifaddr"
	"goddns/internal/provider/cloudflare"
)

var (
	// These variables can be set at build time using -ldflags
	version   = "dev"
	commit    = ""
	buildDate = ""
)

func printVersion() {
	fmt.Printf("goddns %s\n", version)
	if commit != "" {
		fmt.Printf("commit: %s\n", commit)
	}
	if buildDate != "" {
		fmt.Printf("built: %s\n", buildDate)
	}
}

// handleRunCommand implements the core DDNS logic
func handleRunCommand(configFile string, ignoreCache bool) {
	cfg, absConfigFile := config.ReadConfig(configFile, false)

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
				log.Fatal("Fallback also failed: %v", err)
			}
		}
	} else {
		infos, err = ifaddr.GetIPv6Fallback(cfg, false)
		if err != nil {
			log.Fatal("Fallback failed: %v", err)
		}
	}

	currentIP, err := ifaddr.SelectBestIPv6(cfg, infos)
	if err != nil {
		log.Fatal("Failed to select best IPv6 address: %v", err)
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
			log.Fatal("Error fetching Zone ID: %v", err)
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
}

func main() {
	// Support -v/--version anywhere on the command line
	for _, a := range os.Args[1:] {
		if a == "-v" || a == "--version" {
			printVersion()
			return
		}
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: goddns <command> -f/--config <path> [-v/--version]")
		fmt.Println("\nCommands:")
		fmt.Println("  run   - Execute the dynamic DNS update (for cron/systemd). Usage: goddns run -f <path> [-i]")
		os.Exit(1)
	}

	command := os.Args[1]

	runCmd := flag.NewFlagSet("run", flag.ExitOnError)

	var runConfigPath string
	runCmd.StringVar(&runConfigPath, "f", "", "Configuration file path (JSON format).")
	runCmd.StringVar(&runConfigPath, "config", "", "Configuration file path (JSON format).")

	var ignoreCache bool
	runCmd.BoolVar(&ignoreCache, "i", false, "Ignore cached IP; force fetch and write new IP to cache")

	switch command {
	case "run":
		runCmd.Parse(os.Args[2:])
		if runConfigPath == "" {
			log.Fatal("Missing required argument: -f or --config")
		}
		handleRunCommand(runConfigPath, ignoreCache)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Usage: goddns run -f/--config <path> [-i]")
		os.Exit(1)
	}
}