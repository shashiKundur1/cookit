package browser

import (
	"fmt"
	"os"
	"strings"
	"time"

	"cookit/internal/parser"

	"github.com/fatih/color"
	"github.com/playwright-community/playwright-go"
)

type Launcher struct{}

func New() *Launcher {
	return &Launcher{}
}

func EnsureBrowsers() error {
	if os.Getenv("PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH") != "" {
		color.Green("  ✓  Using system Chromium")
		return nil
	}

	color.Cyan("  🔧 Checking Playwright browsers...")
	if err := playwright.Install(); err != nil {
		return fmt.Errorf("cannot install playwright browsers: %w", err)
	}
	color.Green("  ✓  Browsers ready")
	return nil
}

func (l *Launcher) Launch(cookies []parser.Cookie, targetURL string) error {
	color.Cyan("\n  🚀 Launching browser...")

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("playwright failed to start: %w", err)
	}
	defer func() {
		if stopErr := pw.Stop(); stopErr != nil {
			color.Red("  ✗  Failed to stop playwright: %s", stopErr.Error())
		}
	}()

	launchOpts := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
		Channel:  playwright.String("chrome"),
	}

	if execPath := os.Getenv("PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH"); execPath != "" {
		launchOpts.ExecutablePath = playwright.String(execPath)
		launchOpts.Channel = nil
	}

	browser, err := pw.Chromium.Launch(launchOpts)
	if err != nil {
		return fmt.Errorf("cannot launch chromium: %w", err)
	}

	var playwrightCookies []playwright.OptionalCookie
	var skipped int
	domainCount := make(map[string]bool)

	for _, c := range cookies {
		if c.Name == "" || c.Domain == "" {
			skipped++
			continue
		}

		cookiePath := c.Path
		if cookiePath == "" {
			cookiePath = "/"
		}

		secure := c.Secure
		if strings.HasPrefix(c.Name, "__Secure-") || strings.HasPrefix(c.Name, "__Host-") {
			secure = true
		}

		pc := playwright.OptionalCookie{
			Name:   c.Name,
			Value:  c.Value,
			Domain: playwright.String(c.Domain),
			Path:   playwright.String(cookiePath),
		}

		if c.HttpOnly {
			pc.HttpOnly = playwright.Bool(true)
		}
		if secure {
			pc.Secure = playwright.Bool(true)
		}

		playwrightCookies = append(playwrightCookies, pc)
		domainCount[strings.TrimPrefix(c.Domain, ".")] = true
	}

	if skipped > 0 {
		color.Yellow("  ⚠  Skipped %d invalid cookies (missing name or domain)", skipped)
	}

	if len(playwrightCookies) == 0 {
		return fmt.Errorf("no valid cookies to inject")
	}

	context, err := browser.NewContext()
	if err != nil {
		return fmt.Errorf("cannot create browser context: %w", err)
	}

	var injected int
	var failed int
	for _, pc := range playwrightCookies {
		if err := context.AddCookies([]playwright.OptionalCookie{pc}); err != nil {
			failed++
			continue
		}
		injected++
	}

	if failed > 0 {
		color.Yellow("  ⚠  Skipped %d cookies with invalid fields", failed)
	}

	if injected == 0 {
		browser.Close()
		return fmt.Errorf("no cookies could be injected")
	}

	color.Green("  ✓  Injected %d/%d cookies across %d domains", injected, len(playwrightCookies), len(domainCount))

	page, err := context.NewPage()
	if err != nil {
		browser.Close()
		return fmt.Errorf("cannot create page: %w", err)
	}

	color.Cyan("  🌐 Navigating to %s", targetURL)

	if _, err := page.Goto(targetURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(30000),
	}); err != nil {
		color.Yellow("  ⚠  Initial navigation issue: %s", err.Error())
	}

	color.Green("  ✓  Browser is open at %s", targetURL)
	color.HiYellow("\n  ⏳ Browser is open. Use it freely — close the window when done.\n")

	for {
		time.Sleep(500 * time.Millisecond)
		pages := context.Pages()
		if len(pages) == 0 {
			break
		}
		allClosed := true
		for _, p := range pages {
			if !p.IsClosed() {
				allClosed = false
				break
			}
		}
		if allClosed {
			break
		}
	}

	browser.Close()
	color.Green("  ✓  Browser closed. Done!\n")
	return nil
}
