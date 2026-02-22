package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"cookit/internal/api"
	"cookit/internal/browser"
	"cookit/internal/history"
	"cookit/internal/navigator"
	"cookit/internal/parser"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			color.Red("\n  💀 Fatal error: %v", r)
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		color.Yellow("\n\n  ⚡ Interrupted. Cleaning up...")
		os.Exit(0)
	}()

	printBanner()

	if err := godotenv.Load(); err != nil {
		color.Red("  ✗  Cannot load .env file: %s", err.Error())
		color.Yellow("  ℹ  Create a .env file with GEMINI_API_KEY=your_key")
		os.Exit(1)
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		color.Red("  ✗  GEMINI_API_KEY is not set in .env")
		os.Exit(1)
	}

	historyStore, err := history.New("data")
	if err != nil {
		color.Red("  ✗  Cannot initialize history: %s", err.Error())
		os.Exit(1)
	}

	srv := api.New(historyStore)
	go func() {
		if err := srv.Run(":8420"); err != nil {
			color.Red("  ✗  API server failed: %s", err.Error())
		}
	}()
	color.HiBlack("  ℹ  API server running on :8420")

	var startPath string

	if len(os.Args) > 1 {
		startPath = os.Args[1]
	} else {
		lastFolder := historyStore.GetLastFolder()
		if lastFolder != "" {
			color.Cyan("  💡 Last opened folder: %s", lastFolder)
		}

		color.HiYellow("\n  ▶ Enter a path to browse: ")
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			color.Red("  ✗  No input received")
			os.Exit(1)
		}
		startPath = strings.TrimSpace(scanner.Text())
	}

	if startPath == "" {
		color.Red("  ✗  Path cannot be empty")
		os.Exit(1)
	}

	if err := browser.EnsureBrowsers(); err != nil {
		color.Red("  ✗  Browser setup failed: %s", err.Error())
		os.Exit(1)
	}

	nav := navigator.New(historyStore)
	cookieParser := parser.New(apiKey)
	launcher := browser.New()

	for {
		selectedFile, err := nav.Navigate(startPath)
		if err != nil {
			color.Red("  ✗  Navigation failed: %s", err.Error())
			continue
		}

		color.Green("\n  ✓  Selected: %s", selectedFile)

		cookies, targetURL, err := cookieParser.ParseCookies(selectedFile)
		if err != nil {
			color.Red("  ✗  Cookie parsing failed: %s", err.Error())
			continue
		}

		if err := historyStore.Record(selectedFile); err != nil {
			color.Yellow("  ⚠  Could not save to history: %s", err.Error())
		}

		if err := launcher.Launch(cookies, targetURL); err != nil {
			color.Red("  ✗  Browser launch failed: %s", err.Error())
		}

		color.HiMagenta("\n  ─────────────────────────────────────────")
		color.HiMagenta("  🍪  Ready for next cookie file!")
		color.HiMagenta("  ─────────────────────────────────────────\n")
	}
}

func printBanner() {
	banner := color.New(color.FgHiMagenta, color.Bold)
	fmt.Println()
	banner.Println("   ╔══════════════════════════════════════╗")
	banner.Println("   ║          🍪  COOKIT  v1.0            ║")
	banner.Println("   ║   Cookie Injection Terminal Tool     ║")
	banner.Println("   ╚══════════════════════════════════════╝")
	fmt.Println()
}
