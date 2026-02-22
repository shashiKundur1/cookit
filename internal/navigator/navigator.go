package navigator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"cookit/internal/history"

	"github.com/fatih/color"
)

type Navigator struct {
	history *history.Store
	scanner *bufio.Scanner
}

func New(h *history.Store) *Navigator {
	return &Navigator{
		history: h,
		scanner: bufio.NewScanner(os.Stdin),
	}
}

func (n *Navigator) Navigate(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("cannot access path: %w", err)
	}

	if !info.IsDir() {
		return absPath, nil
	}

	currentPath := absPath

	for {
		entries, err := n.listDir(currentPath)
		if err != nil {
			return "", fmt.Errorf("cannot read directory: %w", err)
		}

		if len(entries) == 0 {
			color.Yellow("\n  ⚠  Empty directory: %s", currentPath)
			parent := filepath.Dir(currentPath)
			if parent == currentPath {
				return "", fmt.Errorf("reached filesystem root with no files")
			}
			color.Cyan("  ↩  Going back...\n")
			currentPath = parent
			continue
		}

		n.printEntries(currentPath, entries)

		choice, err := n.prompt(len(entries))
		if err != nil {
			color.Red("  ✗  %s", err.Error())
			continue
		}

		if choice == 0 {
			parent := filepath.Dir(currentPath)
			if parent == currentPath {
				color.Yellow("  ⚠  Already at filesystem root")
				continue
			}
			currentPath = parent
			continue
		}

		selected := entries[choice-1]
		selectedPath := filepath.Join(currentPath, selected.Name())

		if selected.IsDir() {
			currentPath = selectedPath
			continue
		}

		return selectedPath, nil
	}
}

type byTypeAndName []os.DirEntry

func (b byTypeAndName) Len() int      { return len(b) }
func (b byTypeAndName) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byTypeAndName) Less(i, j int) bool {
	if b[i].IsDir() != b[j].IsDir() {
		return b[i].IsDir()
	}
	return strings.ToLower(b[i].Name()) < strings.ToLower(b[j].Name())
}

func (n *Navigator) listDir(path string) ([]os.DirEntry, error) {
	raw, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var filtered []os.DirEntry
	for _, e := range raw {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		filtered = append(filtered, e)
	}

	sort.Sort(byTypeAndName(filtered))
	return filtered, nil
}

func (n *Navigator) printEntries(currentPath string, entries []os.DirEntry) {
	header := color.New(color.FgHiCyan, color.Bold)
	fmt.Println()
	header.Printf("  📂 %s\n", currentPath)
	fmt.Println(strings.Repeat("─", 60))

	dim := color.New(color.FgHiBlack)
	folderColor := color.New(color.FgHiBlue, color.Bold)
	fileColor := color.New(color.FgWhite)
	badge := color.New(color.FgHiGreen)

	dim.Printf("  [0]  ↩  ..\n")

	for i, e := range entries {
		fullPath := filepath.Join(currentPath, e.Name())
		opened := ""
		if n.history.IsOpened(fullPath) {
			opened = badge.Sprintf(" ✓ opened")
		}

		if e.IsDir() {
			folderColor.Printf("  [%d]  📁  %s/%s\n", i+1, e.Name(), opened)
		} else {
			info, _ := e.Info()
			size := ""
			if info != nil {
				size = dim.Sprintf(" (%s)", formatSize(info.Size()))
			}
			fileColor.Printf("  [%d]  📄  %s%s%s\n", i+1, e.Name(), size, opened)
		}
	}

	fmt.Println(strings.Repeat("─", 60))
}

func (n *Navigator) prompt(max int) (int, error) {
	promptColor := color.New(color.FgHiYellow, color.Bold)
	promptColor.Printf("  ▶ Pick [0-%d]: ", max)

	if !n.scanner.Scan() {
		return 0, fmt.Errorf("input closed")
	}

	input := strings.TrimSpace(n.scanner.Text())

	if input == ".." || input == "b" || input == "back" {
		return 0, nil
	}

	num, err := strconv.Atoi(input)
	if err != nil || num < 0 || num > max {
		return 0, fmt.Errorf("invalid choice, enter a number between 0 and %d", max)
	}

	return num, nil
}

func formatSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)

	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
