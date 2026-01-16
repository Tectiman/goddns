package main

import (
	"flag"
	"fmt"
	"os"

	"goddns/internal/config"
	"goddns/internal/log"
	"goddns/internal/platform/netlinkutil"
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

	provider := cloudflare.NewProvider(cfg)

	var infos []netlinkutil.IPv6Info
	var err error
	if cfg.GetIP.Interface != "" {
		infos, err = netlinkutil.GetAvailableIPv6(cfg.GetIP.Interface)
		if err != nil {
			log.Info(false, "Local interface failed (%v), trying fallback API...", err)
			infos, err = netlinkutil.GetIPv6Fallback(cfg, false)
			if err != nil {
				log.Fatal(false, "Fallback also failed: %v", err)
			}
		}
	} else {
		infos, err = netlinkutil.GetIPv6Fallback(cfg, false)
		if err != nil {
			log.Fatal(false, "Fallback failed: %v", err)
		}
	}

	currentIP, err := netlinkutil.SelectBestIPv6(cfg, infos)
	if err != nil {
		log.Fatal(false, "Failed to select best IPv6 address: %v", err)
	}

	cacheFilePath := config.GetCacheFilePath(absConfigFile, cfg.WorkDir)
	lastIP := config.ReadLastIP(cacheFilePath)

	if !ignoreCache {
		if lastIP != "" && lastIP == currentIP {
			log.Info(false, "IP has not changed: %s", currentIP)
			return
		}
	}

	if lastIP != "" {
		// IP changed, proceed with update
	} else {
		// First time update
	}

	zoneID := cfg.Cloudflare.ZoneID
	if zoneID == "" {
		fetchedZoneID, err := provider.GetZoneID(cfg)
		if err != nil {
			log.Fatal(false, "Error fetching Zone ID: %v", err)
		}
		cfg.Cloudflare.ZoneID = fetchedZoneID
		zoneID = fetchedZoneID

		if writeErr := config.WriteConfig(absConfigFile, cfg); writeErr != nil {
			log.Warning(false, "Warning: Failed to save Zone ID to config file: %v", writeErr)
		}
	}

	success, err := provider.UpsertDNSRecord(cfg, currentIP, zoneID)

	if success {
		if writeErr := config.WriteLastIP(cacheFilePath, currentIP); writeErr != nil {
			log.Warning(false, "Update succeeded, but failed to write IP to cache: %v", writeErr)
		}
		log.Success(false, "DDNS update successful: %s", currentIP)
	} else {
		log.Error(false, "DDNS update failed: %v", err)
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
		fmt.Println("Usage: goddns <command> -f/--config <path> [-v|--version]")
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
			log.Fatal(false, "Missing required argument: -f or --config")
		}
		handleRunCommand(runConfigPath, ignoreCache)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Usage: goddns run -f/--config <path> [-i]")
		os.Exit(1)
	}
}
