package main

import (
	"bufio"
	"flag"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// skipYAMLFrontMatter skips YAML front matter (--- ... ---) and returns the content lines after it.
func skipYAMLFrontMatter(lines []string) []string {
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				return lines[i+1:]
			}
		}
	}
	return lines
}

// extractSummaryText returns the first n lines (HTML-escaped, joined with <br>) from content lines.
func extractSummaryText(lines []string, n int) string {
	if len(lines) < n {
		n = len(lines)
	}
	first := lines[:n]
	for i, l := range first {
		first[i] = html.EscapeString(l)
	}
	return strings.Join(first, "<br>")
}

// writeDetailsBlock writes a details block to out, using summary and preview lines.
func writeDetailsBlock(out *os.File, summary string, contentLines []string, fileName string) {
	fmt.Fprintf(out, "<details>\n")
	fmt.Fprintf(out, "<summary>%s</summary>\n\n", summary)
	fmt.Fprintf(out, "[リンク](%s)\n\n", fileName)

	isCodeBlock := false
	for _, line := range contentLines {
		// If line is header line, increment header level
		if !isCodeBlock && strings.HasPrefix(line, "#") {
			fmt.Fprint(out, "#")
		}
		if strings.HasPrefix(line, "```") {
			// Toggle code block state
			isCodeBlock = !isCodeBlock
		}
		fmt.Fprintln(out, line)
	}
	fmt.Fprintf(out, "</details>\n\n")
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"Usage: %s <category-dir>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	catDir := flag.Arg(0)

	// 1. Capitalize category name
	catName := filepath.Base(catDir)
	title := toTitle(catName)

	// 2. List and sort Markdown files
	entries, err := os.ReadDir(catDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading dir: %v\n", err)
		os.Exit(1)
	}
	var mdFiles []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".md") && e.Name() != "index.md" {
			mdFiles = append(mdFiles, e)
		}
	}
	sort.Slice(mdFiles, func(i, j int) bool {
		return mdFiles[i].Name() < mdFiles[j].Name()
	})

	// 3. Open index.md for writing
	outPath := filepath.Join(catDir, "index.md")
	out, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot create index.md: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	// 4. Write header
	fmt.Fprintf(out, "# %s\n\n", title)

	// 5. Process each file
	for _, fi := range mdFiles {
		path := filepath.Join(catDir, fi.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read %s: %v\n", path, err)
			continue
		}
		lines := splitLines(string(content))
		contentLines := skipYAMLFrontMatter(lines)
		summary := extractSummaryText(contentLines, 1)
		writeDetailsBlock(out, summary, contentLines, fi.Name())
	}

	fmt.Printf("Generated %s\n", outPath)
}

// toTitle splits on non‐letters, Title‐cases each token, and joins with spaces.
func toTitle(s string) string {
	var parts []string
	var buf strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) {
			buf.WriteRune(r)
		} else {
			if buf.Len() > 0 {
				parts = append(parts, buf.String())
				buf.Reset()
			}
		}
	}
	if buf.Len() > 0 {
		parts = append(parts, buf.String())
	}
	for i, w := range parts {
		parts[i] = strings.ToUpper(string(w[0])) + strings.ToLower(w[1:])
	}
	return strings.Join(parts, " ")
}

// splitLines handles both Unix and Windows line endings.
func splitLines(content string) []string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}
