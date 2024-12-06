package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

type Release struct {
	Version     string  `json:"version"`
	Platform    string  `json:"platform"`
	Channel     string  `json:"channel"`
	Milestone   int     `json:"milestone"`
	ReleaseDate float64 `json:"time"`
}

func getLatestCanaryRelease() (*Release, error) {
	url := "https://chromiumdash.appspot.com/fetch_releases?channel=Canary&platform=Win64,Linux,Mac"

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var releases []Release
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found")
	}

	// Find the latest release across all platforms
	latest := releases[0]
	for _, release := range releases[1:] {
		if release.ReleaseDate > latest.ReleaseDate {
			latest = release
		}
	}

	return &latest, nil
}

const nativeTemplateText = `// This file is generated by html_to_wrapper.py


export const getTemplateHtml = function(): HTMLTemplateElement {
    return getTemplate({{ .Content }});
};`

type HTMLWrapper struct {
	Content string
}

func downloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func downloadGitiles(tag, path string) ([]byte, error) {
	url := fmt.Sprintf("https://chromium.googlesource.com/chromium/src/+/refs/tags/%s/%s?format=TEXT", tag, path)
	data, err := downloadFile(url)
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(string(data))
}

func extractTarGz(data []byte, destPath string) error {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar reading error: %w", err)
		}

		target := filepath.Join(destPath, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		case tar.TypeReg:
			dir := filepath.Dir(target)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to write to file %s: %w", target, err)
			}
			f.Close()
		}
	}
	return nil
}

func copyFiles() error {
	files := []string{
		"icon.ts",
		"tab_strip.mojom-webui.ts",
		"tabs.mojom-webui.ts",
		"tabs_api_proxy.ts",
	}

	for _, file := range files {
		srcPath := filepath.Join("..", "src", file)
		destPath := filepath.Join("out", file)
		if err := copyFile(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", file, err)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0644)
}

func processTypeScriptFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Apply sed-like replacements
	replaced := string(content)
	replaced = strings.ReplaceAll(replaced, "chrome://resources/js/", "./")
	replaced = strings.ReplaceAll(replaced, "//resources/js/", "./")
	replaced = regexp.MustCompile(`(?m).*\.\/strings\.m\.js.*\n`).ReplaceAllString(replaced, "")

	// Remove ColorChangeUpdater lines from tab_list.ts
	if strings.HasSuffix(path, "tab_list.ts") {
		replaced = regexp.MustCompile(`(?m).*ColorChangeUpdater.*\n`).ReplaceAllString(replaced, "")
	}

	return os.WriteFile(path, []byte(replaced), 0644)
}

func convertHTMLToWrapper(inPath, outPath string) error {
	content, err := os.ReadFile(inPath)
	if err != nil {
		return err
	}

	// Escape the HTML content for JS template string
	escaped := fmt.Sprintf("`%s`", strings.ReplaceAll(string(content), "`", "\\`"))

	wrapper := HTMLWrapper{Content: escaped}
	tmpl, err := template.New("wrapper").Parse(nativeTemplateText)
	if err != nil {
		return err
	}

	outFile, err := os.Create(outPath + ".ts")
	if err != nil {
		return err
	}
	defer outFile.Close()

	return tmpl.Execute(outFile, wrapper)
}

func embedImages(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	replaced := string(content)

	// Embed sad favicon
	sadFaviconData, err := os.ReadFile("../strip/IDR_CRASH_SAD_FAVICON@2x.png")
	if err == nil {
		sadFaviconBase64 := base64.StdEncoding.EncodeToString(sadFaviconData)
		replaced = strings.ReplaceAll(replaced,
			"chrome://theme/IDR_CRASH_SAD_FAVICON@2x",
			"data:image/png;base64,"+sadFaviconBase64)
	}

	// Embed clear icon
	clearIconData, err := os.ReadFile("../strip/icon_clear.svg")
	if err == nil {
		clearIconBase64 := base64.StdEncoding.EncodeToString(clearIconData)
		replaced = strings.ReplaceAll(replaced,
			"chrome://resources/images/icon_clear.svg",
			"data:image/svg+xml;base64,"+clearIconBase64)
	}

	return os.WriteFile(path, []byte(replaced), 0644)
}

func main() {
	// Get the latest canary release version
	release, err := getLatestCanaryRelease()
	if err != nil {
		fmt.Printf("Error getting latest release: %v\n", err)
		os.Exit(1)
	}
	tag := release.Version

	// Check if copyonly flag is provided
	if len(os.Args) > 1 && os.Args[1] == "copyonly" {
		if err := copyFiles(); err != nil {
			fmt.Printf("Error copying files: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Clean and create directories
	dirs := []string{"in", "in2", "grit", "out"}
	for _, dir := range dirs {
		os.RemoveAll(dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	// Download and extract tab_strip.tar.gz
	tabStripURL := fmt.Sprintf("https://chromium.googlesource.com/chromium/src/+archive/refs/tags/%s/chrome/browser/resources/tab_strip.tar.gz", tag)
	tabStripData, err := downloadFile(tabStripURL)
	if err != nil {
		fmt.Printf("Error downloading tab_strip: %v\n", err)
		os.Exit(1)
	}
	if err := extractTarGz(tabStripData, "in"); err != nil {
		fmt.Printf("Error extracting tab_strip: %v\n", err)
		os.Exit(1)
	}

	// Download and extract grit.tar.gz
	gritURL := fmt.Sprintf("https://chromium.googlesource.com/chromium/src/+archive/refs/tags/%s/tools/grit.tar.gz", tag)
	gritData, err := downloadFile(gritURL)
	if err != nil {
		fmt.Printf("Error downloading grit: %v\n", err)
		os.Exit(1)
	}
	if err := extractTarGz(gritData, "grit"); err != nil {
		fmt.Printf("Error extracting grit: %v\n", err)
		os.Exit(1)
	}

	// Download util.ts to in folder
	utilData, err := downloadGitiles(tag, "ui/webui/resources/js/util.ts")
	if err != nil {
		fmt.Printf("Error downloading util.ts: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile("in/util.ts", utilData, 0644); err != nil {
		fmt.Printf("Error writing util.ts: %v\n", err)
		os.Exit(1)
	}

	// Copy files from in to in2
	if err := exec.Command("cp", "-r", "in/.", "in2/").Run(); err != nil {
		fmt.Printf("Error copying files to in2: %v\n", err)
		os.Exit(1)
	}

	// Run Python preprocessing script
	pythonCmd := exec.Command("python3", "grit/preprocess_if_expr.py",
		"--in-folder", "in",
		"--out-folder", "in2",
		"--in-files", "drag_manager.ts", "tab_list.html", "util.ts",
		"-D", "linux=true",
		"-D", "chromeos_ash=false",
		"-D", "macosx=false")
	if err := pythonCmd.Run(); err != nil {
		fmt.Printf("Error running preprocess_if_expr.py: %v\n", err)
		os.Exit(1)
	}

	// Convert HTML files to wrappers
	htmlFiles := []string{
		"alert_indicator.html",
		"alert_indicators.html",
		"tab_group.html",
		"tab_list.html",
		"tab.html",
	}
	for _, file := range htmlFiles {
		inPath := filepath.Join("in2", file)
		outPath := filepath.Join("out", file)
		if err := convertHTMLToWrapper(inPath, outPath); err != nil {
			fmt.Printf("Error converting %s to wrapper: %v\n", file, err)
			os.Exit(1)
		}
	}

	// Copy TypeScript files
	tsFiles := []string{
		"alert_indicator.ts",
		"alert_indicators.ts",
		"tab_group.ts",
		"tab_list.ts",
		"tab.ts",
		"drag_manager.ts",
		"tab_swiper.ts",
		"tab_strip.html",
		"util.ts",
	}
	for _, file := range tsFiles {
		srcPath := filepath.Join("in2", file)
		destPath := filepath.Join("out", file)
		if err := copyFile(srcPath, destPath); err != nil {
			fmt.Printf("Error copying %s: %v\n", file, err)
			os.Exit(1)
		}
	}

	// Copy alert_indicators directory
	if err := exec.Command("cp", "-r", "in/alert_indicators", "out/").Run(); err != nil {
		fmt.Printf("Error copying alert_indicators directory: %v\n", err)
		os.Exit(1)
	}

	// Process all TypeScript files in out directory
	outFiles, err := filepath.Glob("out/*.ts")
	if err != nil {
		fmt.Printf("Error listing output files: %v\n", err)
		os.Exit(1)
	}
	for _, file := range outFiles {
		if err := processTypeScriptFile(file); err != nil {
			fmt.Printf("Error processing %s: %v\n", file, err)
			os.Exit(1)
		}
	}

	// Download and process additional files
	additionalFiles := map[string]bool{
		"ui/webui/resources/js/assert.ts":                false,
		"ui/webui/resources/js/custom_element.ts":        true,
		"ui/webui/resources/js/event_tracker.ts":         false,
		"ui/webui/resources/js/focus_outline_manager.ts": false,
		"ui/webui/resources/js/load_time_data.ts":        false,
		"ui/webui/resources/js/static_types.ts":          true,
	}

	for file, needsNoCheck := range additionalFiles {
		data, err := downloadGitiles(tag, file)
		if err != nil {
			fmt.Printf("Error downloading %s: %v\n", file, err)
			os.Exit(1)
		}

		outPath := filepath.Join("out", filepath.Base(file))
		if needsNoCheck {
			data = append([]byte("// @ts-nocheck\n"), data...)
		}
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			fmt.Printf("Error writing %s: %v\n", outPath, err)
			os.Exit(1)
		}
	}

	// Embed images in tab.html.ts
	if err := embedImages("out/tab.html.ts"); err != nil {
		fmt.Printf("Error embedding images: %v\n", err)
		os.Exit(1)
	}

	// Final copy of files
	if err := copyFiles(); err != nil {
		fmt.Printf("Error in final file copy: %v\n", err)
		os.Exit(1)
	}

	// Clean up
	for _, dir := range []string{"in", "in2", "grit"} {
		os.RemoveAll(dir)
	}

	fmt.Printf("Successfully processed files for Chromium version %s\n", tag)
}
