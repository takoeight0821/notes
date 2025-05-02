package main

import (
	"bufio"
	"flag"
	"fmt"
	"html"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

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
	entries, err := ioutil.ReadDir(catDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading dir: %v\n", err)
		os.Exit(1)
	}
	var mdFiles []fs.FileInfo
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
		content, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read %s: %v\n", path, err)
			continue
		}
		lines := splitLines(string(content))

		// --- skip YAML front matter ---
		contentStart := 0
		if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
			// Find the closing '---'
			for i := 1; i < len(lines); i++ {
				if strings.TrimSpace(lines[i]) == "---" {
					contentStart = i + 1
					break
				}
			}
		}
		contentLines := lines[contentStart:]

		// --- summary: first 5 lines of content, HTML-escaped, joined with <br> ---
		previewCount := 10
		if len(contentLines) < previewCount {
			previewCount = len(contentLines)
		}
		// For summary, use first 5 lines
		summaryLines := 5
		if len(contentLines) < summaryLines {
			summaryLines = len(contentLines)
		}
		first5 := contentLines[:summaryLines]
		for i, l := range first5 {
			first5[i] = html.EscapeString(l)
		}
		summaryText := strings.Join(first5, "<br>")

		// 6. Write details block
		fmt.Fprintf(out, "<details>\n")
		fmt.Fprintf(out, "<summary>%s</summary>\n\n", summaryText)
		for i := 0; i < previewCount; i++ {
			fmt.Fprintln(out, contentLines[i])
		}
		// If longer, append rest un‐collapsed
		if len(contentLines) > previewCount {
			for i := previewCount; i < len(contentLines); i++ {
				fmt.Fprintln(out, contentLines[i])
			}
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "</details>\n\n")
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
