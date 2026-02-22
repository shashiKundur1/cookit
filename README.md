<div align="center">

# 🍪 Cookit

**Cookie Injection Terminal Tool**

Navigate to cookie files interactively, parse them instantly, and launch a headed Chrome browser with cookies pre-injected.

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Playwright](https://img.shields.io/badge/Playwright-Go-2EAD33?style=flat-square&logo=playwright&logoColor=white)](https://playwright-community.github.io/playwright-go/)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)

</div>

---

## ✨ Features

- **Interactive file navigator** — browse folders with arrow-style navigation, file sizes, and type icons
- **Smart cookie parsing** — auto-detects Netscape/tab-separated cookies instantly (no API needed)
- **AI fallback** — uses Gemini Vision API for unusual formats and image-based cookie files
- **Headed Chrome** — opens your real Google Chrome with full video codec support
- **Cookie injection** — handles `__Secure-`, `__Host-` prefixed cookies, skips invalid ones gracefully
- **History tracking** — marks previously opened files with `✓ opened`
- **Loop mode** — close browser → pick next file instantly, no restart needed
- **REST API** — Gin-based API server on `:8420` for history access
- **Graceful error handling** — retries, fallbacks, panic recovery, SIGINT handling

## 📦 Installation

```bash
git clone https://github.com/your-org/cookit.git
cd cookit
go mod download
go build -o cookit .
```

Install Playwright browsers (first time only):

```bash
go run github.com/playwright-community/playwright-go/cmd/playwright install --with-deps
```

## ⚙️ Configuration

Create a `.env` file:

```env
GEMINI_API_KEY=your_gemini_api_key_here
```

> The Gemini API key is only needed for image-based cookie files or unusual formats. Standard `.txt` cookie files are parsed directly without any API call.

## 🚀 Usage

```bash
# Navigate from a specific path
./cookit /path/to/cookie/folder

# Or start with an interactive prompt
./cookit
```

### How it works

```
   ╔══════════════════════════════════════╗
   ║          🍪  COOKIT  v1.0            ║
   ║   Cookie Injection Terminal Tool     ║
   ╚══════════════════════════════════════╝

  📂 /path/to/cookies
────────────────────────────────────────────────────
  [0]  ↩  ..
  [1]  📁  Grok/
  [2]  📁  Instagram/
  [3]  📁  ChatGPT/
────────────────────────────────────────────────────
  ▶ Pick [0-3]: 1

  ✓  Selected: cookies/Grok/session.txt
  ✓  Direct parse: 11 cookies → https://grok.com
  ✓  Injected 11/11 cookies across 2 domains
  🌐 Navigating to https://grok.com
  ✓  Browser is open at https://grok.com

  ⏳ Browser is open. Use it freely — close the window when done.
```

1. **Navigate** — browse folders, pick a cookie file
2. **Parse** — cookies are extracted instantly (Netscape format) or via Gemini AI
3. **Launch** — Chrome opens with cookies injected, navigates to the target site
4. **Loop** — close the browser → pick the next file immediately

## 📁 Supported Cookie Formats

| Format | Method | Speed |
|--------|--------|-------|
| Netscape (tab-separated `.txt`) | Direct parser | ⚡ Instant |
| JSON cookie files | Direct parser | ⚡ Instant |
| Screenshot/image of cookies | Gemini Vision API | ~3s |
| Unknown text formats | Gemini API | ~3s |

## 🌐 Auto-Detected Sites

Cookit automatically maps cookie domains to URLs:

| Domain | Opens |
|--------|-------|
| `.grok.com` / `.x.ai` | `https://grok.com` |
| `.chatgpt.com` / `.openai.com` | `https://chatgpt.com` |
| `.instagram.com` | `https://www.instagram.com` |
| `.twitter.com` / `.x.com` | `https://x.com` |
| `.kick.com` | `https://kick.com` |
| `.twitch.tv` | `https://www.twitch.tv` |
| Other domains | `https://<domain>` |

## 🏗️ Project Structure

```
cookit/
├── main.go                    # Entry point, loop, signal handling
├── .env                       # Gemini API key
├── Dockerfile                 # Multi-stage Alpine build
├── docker-compose.yml         # Docker compose config
├── run.sh                     # Docker run helper
└── internal/
    ├── api/api.go             # Gin REST API (history endpoints)
    ├── browser/browser.go     # Playwright Chrome launcher + cookie injection
    ├── history/history.go     # JSON-based file open history
    ├── navigator/navigator.go # Interactive terminal file browser
    └── parser/parser.go       # Netscape parser + Gemini AI fallback
```

## 🐳 Docker

```bash
# Build
docker build -t cookit .

# Run (requires X11/XQuartz on macOS for headed browser)
docker run -it --rm \
  -e GEMINI_API_KEY=your_key \
  -v /path/to/cookies:/cookies:ro \
  cookit:latest /cookies
```

## 📜 License

MIT
