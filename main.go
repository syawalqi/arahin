package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/syawalqi/arahin/cmd"
	"github.com/syawalqi/arahin/config"
)

// Version is set at build time via -ldflags.
var version = "dev"

func main() {
	cfg := loadConfig()
	if len(os.Args) < 2 {
		// Default to chat when invoked as just "arahin"
		runChat(cfg)
		return
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "chat":
		runChat(cfg)
	case "fix":
		if err := cmd.Fix(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "fix error: %v\n", err)
			os.Exit(1)
		}
	case "setup":
		if err := cmd.Setup(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "setup error: %v\n", err)
			os.Exit(1)
		}
	case "daemon":
		if err := cmd.Daemon(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "daemon error: %v\n", err)
			os.Exit(1)
		}
	case "alert":
		if err := cmd.AlertCli(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "alert error: %v\n", err)
			os.Exit(1)
		}
	case "route":
		// Parse --mode flag from args
		mode := cfg.Route.Mode
		args := os.Args[2:]
		promptParts := make([]string, 0)
		for i := 0; i < len(args); i++ {
			if args[i] == "--mode" && i+1 < len(args) {
				mode = args[i+1]
				i++
			} else if strings.HasPrefix(args[i], "--mode=") {
				mode = args[i][len("--mode="):]
			} else {
				promptParts = append(promptParts, args[i])
			}
		}
		if len(promptParts) == 0 {
			fmt.Fprintln(os.Stderr, "usage: arahin route [--mode bare|pipeline|agent] <prompt>")
			os.Exit(1)
		}
		prompt := strings.Join(promptParts, " ")
		cfg.Route.Mode = mode
		if err := cmd.Route(cfg, prompt); err != nil {
			fmt.Fprintf(os.Stderr, "route error: %v\n", err)
			os.Exit(1)
		}
	case "serve":
		if err := cmd.Serve(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "serve error: %v\n", err)
			os.Exit(1)
		}
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", subcommand)
		usage()
		os.Exit(1)
	}
}

func runChat(cfg *config.Config) {
	if err := cmd.Chat(cfg, version); err != nil {
		fmt.Fprintf(os.Stderr, "chat error: %v\n", err)
		os.Exit(1)
	}
}

func loadConfig() *config.Config {
	cfg := config.Default()
	configPath := os.ExpandEnv("$HOME/.config/arahin/config.yaml")
	if parsed, err := config.Load(configPath); err == nil {
		cfg = parsed
	}
	// Env overrides for API key
	if key := os.Getenv("LLM_API_KEY"); key != "" {
		cfg.APIKey = key
	}
	if key := os.Getenv("OPENCODE_GO_API_KEY"); key != "" && cfg.APIKey == "" {
		cfg.APIKey = key
	}
	if key := os.Getenv("COUNCIL_API_KEY"); key != "" && cfg.APIKey == "" {
		cfg.APIKey = key
	}
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" && cfg.Provider == "openrouter" && cfg.APIKey == "" {
		cfg.APIKey = key
	}
	return cfg
}

func usage() {
	fmt.Print(`Arahin — Route Planning AI

Usage:
  arahin setup      Interactive first-run configuration
  arahin chat       Interactive chat with LLM agent
  arahin daemon     Background monitoring daemon
  arahin fix        Fix an anomaly (auto-remediate with LLM)
  arahin alert      Send an alert (script hook)
  arahin route      Plan a multi-stop route (CLI)
  arahin serve      Start web server with map UI
  arahin help       Show this help
`)
}
