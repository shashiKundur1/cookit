package browser

import (
	"fmt"
	"time"

	"cookit/internal/parser"

	"github.com/fatih/color"
	"github.com/playwright-community/playwright-go"
)

type Launcher struct{}

func New() *Launcher {
	return &Launcher{}
}

func (l *Launcher) Launch(cookies []parser.Cookie, targetURL string) error {
	color.Cyan("\n  🚀 Launching browser...")

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("playwright failed to start (run 'go run github.com/playwright-community/playwright-go/cmd/playwright install --with-deps'): %w", err)
	}
	defer func() {
		if stopErr := pw.Stop(); stopErr != nil {
			color.Red("  ✗  Failed to stop playwright: %s", stopErr.Error())
		}
	}()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	if err != nil {
		return fmt.Errorf("cannot launch chromium: %w", err)
	}
	defer func() {
		if closeErr := browser.Close(); closeErr != nil {
			color.Red("  ✗  Failed to close browser: %s", closeErr.Error())
		}
	}()

	var playwrightCookies []playwright.OptionalCookie
	var skipped int

	for _, c := range cookies {
		if c.Name == "" || c.Domain == "" {
			skipped++
			continue
		}

		pc := playwright.OptionalCookie{
			Name:   c.Name,
			Value:  c.Value,
			Domain: playwright.String(c.Domain),
			Path:   playwright.String(c.Path),
		}

		if c.HttpOnly {
			pc.HttpOnly = playwright.Bool(true)
		}
		if c.Secure {
			pc.Secure = playwright.Bool(true)
		}

		playwrightCookies = append(playwrightCookies, pc)
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

	if err := context.AddCookies(playwrightCookies); err != nil {
		return fmt.Errorf("cannot inject cookies: %w", err)
	}

	color.Green("  ✓  Injected %d cookies", len(playwrightCookies))

	page, err := context.NewPage()
	if err != nil {
		return fmt.Errorf("cannot create page: %w", err)
	}

	color.Cyan("  🌐 Navigating to %s", targetURL)

	var navigated bool
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := page.Goto(targetURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			Timeout:   playwright.Float(30000),
		}); err != nil {
			if attempt == 0 {
				color.Yellow("  ⚠  Navigation timeout, retrying...")
				time.Sleep(2 * time.Second)
				continue
			}
			return fmt.Errorf("navigation failed: %w", err)
		}
		navigated = true
		break
	}

	if !navigated {
		return fmt.Errorf("failed to navigate after retries")
	}

	color.Green("  ✓  Browser is open at %s", targetURL)
	color.HiYellow("\n  ⏳ Waiting for you to close the browser window...\n")

	page.WaitForEvent("close")

	color.Green("  ✓  Browser closed. Done!\n")
	return nil
}
