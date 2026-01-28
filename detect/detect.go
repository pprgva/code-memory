// Package detect provides automatic project type detection.
package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ProjectInfo contains detected project information.
type ProjectInfo struct {
	Name      string   `json:"name"`
	Languages []string `json:"languages"`
	Framework string   `json:"framework,omitempty"`
	BuildTool string   `json:"build_tool,omitempty"`
}

// DetectProject detects the project type and metadata from the given root path.
func DetectProject(rootPath string) (*ProjectInfo, error) {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}

	info := &ProjectInfo{
		Name:      filepath.Base(absPath),
		Languages: []string{},
	}

	// Detect languages and build tools based on marker files
	detectLanguagesAndBuildTools(absPath, info)

	// Detect framework
	detectFramework(absPath, info)

	return info, nil
}

// detectLanguagesAndBuildTools detects programming languages and build tools.
func detectLanguagesAndBuildTools(rootPath string, info *ProjectInfo) {
	// Check for language-specific marker files
	markers := []struct {
		file      string
		language  string
		buildTool string
	}{
		// Go
		{"go.mod", "go", "go"},
		{"go.sum", "go", ""},

		// Swift
		{"Package.swift", "swift", "spm"},
		{"project.yml", "swift", "xcodegen"},
		{"*.xcodeproj", "swift", "xcode"},
		{"*.xcworkspace", "swift", "xcode"},

		// Python
		{"pyproject.toml", "python", ""},
		{"setup.py", "python", "setuptools"},
		{"requirements.txt", "python", "pip"},
		{"Pipfile", "python", "pipenv"},
		{"poetry.lock", "python", "poetry"},

		// JavaScript/TypeScript
		{"package.json", "", ""}, // Language detected from contents
		{"tsconfig.json", "typescript", ""},

		// Rust
		{"Cargo.toml", "rust", "cargo"},

		// Java/Kotlin
		{"pom.xml", "java", "maven"},
		{"build.gradle", "", "gradle"},
		{"build.gradle.kts", "kotlin", "gradle"},

		// Ruby
		{"Gemfile", "ruby", "bundler"},
		{"Rakefile", "ruby", "rake"},

		// PHP
		{"composer.json", "php", "composer"},

		// C#/.NET
		{"*.csproj", "csharp", "dotnet"},
		{"*.sln", "csharp", "dotnet"},

		// C/C++
		{"CMakeLists.txt", "", "cmake"},
		{"Makefile", "", "make"},
		{"meson.build", "", "meson"},
	}

	languageSet := make(map[string]bool)

	for _, m := range markers {
		// Handle glob patterns
		if strings.Contains(m.file, "*") {
			matches, err := filepath.Glob(filepath.Join(rootPath, m.file))
			if err == nil && len(matches) > 0 {
				if m.language != "" {
					languageSet[m.language] = true
				}
				if m.buildTool != "" && info.BuildTool == "" {
					info.BuildTool = m.buildTool
				}
			}
		} else {
			if _, err := os.Stat(filepath.Join(rootPath, m.file)); err == nil {
				if m.language != "" {
					languageSet[m.language] = true
				}
				if m.buildTool != "" && info.BuildTool == "" {
					info.BuildTool = m.buildTool
				}
			}
		}
	}

	// Special handling for package.json (detect JS vs TS)
	if _, err := os.Stat(filepath.Join(rootPath, "package.json")); err == nil {
		if _, err := os.Stat(filepath.Join(rootPath, "tsconfig.json")); err == nil {
			languageSet["typescript"] = true
		} else {
			languageSet["javascript"] = true
		}
		if info.BuildTool == "" {
			info.BuildTool = "npm"
		}
	}

	// Detect languages from file extensions if no marker files found
	if len(languageSet) == 0 {
		detectLanguagesFromExtensions(rootPath, languageSet)
	}

	// Convert set to slice
	for lang := range languageSet {
		info.Languages = append(info.Languages, lang)
	}
}

// detectLanguagesFromExtensions scans top-level and one level deep for source files.
func detectLanguagesFromExtensions(rootPath string, languageSet map[string]bool) {
	extensions := map[string]string{
		".go":    "go",
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".tsx":   "typescript",
		".jsx":   "javascript",
		".rs":    "rust",
		".java":  "java",
		".kt":    "kotlin",
		".rb":    "ruby",
		".php":   "php",
		".swift": "swift",
		".c":     "c",
		".cpp":   "cpp",
		".h":     "c",
		".hpp":   "cpp",
		".cs":    "csharp",
	}

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Check one level deep
			subPath := filepath.Join(rootPath, entry.Name())
			subEntries, err := os.ReadDir(subPath)
			if err != nil {
				continue
			}
			for _, subEntry := range subEntries {
				if !subEntry.IsDir() {
					ext := strings.ToLower(filepath.Ext(subEntry.Name()))
					if lang, ok := extensions[ext]; ok {
						languageSet[lang] = true
					}
				}
			}
		} else {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if lang, ok := extensions[ext]; ok {
				languageSet[lang] = true
			}
		}
	}
}

// detectFramework detects the framework used in the project.
func detectFramework(rootPath string, info *ProjectInfo) {
	// Check for Go frameworks
	if containsLang(info.Languages, "go") {
		info.Framework = detectGoFramework(rootPath)
		return
	}

	// Check for Python frameworks
	if containsLang(info.Languages, "python") {
		info.Framework = detectPythonFramework(rootPath)
		return
	}

	// Check for JS/TS frameworks
	if containsLang(info.Languages, "javascript") || containsLang(info.Languages, "typescript") {
		info.Framework = detectJSFramework(rootPath)
		return
	}

	// Check for Swift frameworks
	if containsLang(info.Languages, "swift") {
		info.Framework = detectSwiftFramework(rootPath)
		return
	}

	// Check for Ruby frameworks
	if containsLang(info.Languages, "ruby") {
		info.Framework = detectRubyFramework(rootPath)
		return
	}
}

func containsLang(languages []string, lang string) bool {
	for _, l := range languages {
		if l == lang {
			return true
		}
	}
	return false
}

// detectGoFramework detects Go web frameworks from go.mod.
func detectGoFramework(rootPath string) string {
	data, err := os.ReadFile(filepath.Join(rootPath, "go.mod"))
	if err != nil {
		return ""
	}
	content := string(data)

	frameworks := []struct {
		indicator string
		name      string
	}{
		{"github.com/gin-gonic/gin", "gin"},
		{"github.com/labstack/echo", "echo"},
		{"github.com/gofiber/fiber", "fiber"},
		{"github.com/gorilla/mux", "gorilla"},
		{"github.com/go-chi/chi", "chi"},
		{"github.com/beego/beego", "beego"},
		{"github.com/revel/revel", "revel"},
	}

	for _, f := range frameworks {
		if strings.Contains(content, f.indicator) {
			return f.name
		}
	}
	return ""
}

// detectPythonFramework detects Python web frameworks.
func detectPythonFramework(rootPath string) string {
	// Check pyproject.toml first
	data, err := os.ReadFile(filepath.Join(rootPath, "pyproject.toml"))
	if err == nil {
		content := string(data)
		frameworks := []struct {
			indicator string
			name      string
		}{
			{"fastapi", "fastapi"},
			{"django", "django"},
			{"flask", "flask"},
			{"starlette", "starlette"},
			{"tornado", "tornado"},
			{"aiohttp", "aiohttp"},
		}
		for _, f := range frameworks {
			if strings.Contains(strings.ToLower(content), f.indicator) {
				return f.name
			}
		}
	}

	// Check requirements.txt
	data, err = os.ReadFile(filepath.Join(rootPath, "requirements.txt"))
	if err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "django") {
			return "django"
		}
		if strings.Contains(content, "flask") {
			return "flask"
		}
		if strings.Contains(content, "fastapi") {
			return "fastapi"
		}
	}

	// Check for manage.py (Django)
	if _, err := os.Stat(filepath.Join(rootPath, "manage.py")); err == nil {
		return "django"
	}

	return ""
}

// detectJSFramework detects JavaScript/TypeScript frameworks from package.json.
func detectJSFramework(rootPath string) string {
	data, err := os.ReadFile(filepath.Join(rootPath, "package.json"))
	if err != nil {
		return ""
	}

	// Parse as JSON to check dependencies properly
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		// Fallback to string matching
		content := string(data)
		return detectJSFrameworkFromString(content)
	}

	// Check dependencies in order of specificity
	allDeps := make(map[string]bool)
	for dep := range pkg.Dependencies {
		allDeps[dep] = true
	}
	for dep := range pkg.DevDependencies {
		allDeps[dep] = true
	}

	// Frameworks in order of specificity (meta-frameworks first)
	frameworks := []struct {
		dep  string
		name string
	}{
		{"next", "next.js"},
		{"nuxt", "nuxt"},
		{"@remix-run/react", "remix"},
		{"gatsby", "gatsby"},
		{"@angular/core", "angular"},
		{"svelte", "svelte"},
		{"vue", "vue"},
		{"react", "react"},
		{"express", "express"},
		{"fastify", "fastify"},
		{"@nestjs/core", "nest.js"},
		{"hono", "hono"},
	}

	for _, f := range frameworks {
		if allDeps[f.dep] {
			return f.name
		}
	}

	return "node"
}

func detectJSFrameworkFromString(content string) string {
	frameworks := []struct {
		indicator string
		name      string
	}{
		{"\"next\"", "next.js"},
		{"\"nuxt\"", "nuxt"},
		{"\"react\"", "react"},
		{"\"vue\"", "vue"},
		{"\"angular\"", "angular"},
		{"\"svelte\"", "svelte"},
		{"\"express\"", "express"},
		{"\"fastify\"", "fastify"},
	}

	for _, f := range frameworks {
		if strings.Contains(content, f.indicator) {
			return f.name
		}
	}
	return "node"
}

// detectSwiftFramework detects Swift frameworks.
func detectSwiftFramework(rootPath string) string {
	// Check for SwiftUI imports in source files
	swiftFiles, err := filepath.Glob(filepath.Join(rootPath, "**", "*.swift"))
	if err != nil || len(swiftFiles) == 0 {
		// Try direct directory
		swiftFiles, _ = filepath.Glob(filepath.Join(rootPath, "*.swift"))
	}

	for _, file := range swiftFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		content := string(data)
		if strings.Contains(content, "import SwiftUI") {
			return "swiftui"
		}
		if strings.Contains(content, "import UIKit") {
			return "uikit"
		}
		if strings.Contains(content, "import AppKit") {
			return "appkit"
		}
	}

	// Check Package.swift for Vapor
	data, err := os.ReadFile(filepath.Join(rootPath, "Package.swift"))
	if err == nil {
		if strings.Contains(string(data), "vapor") {
			return "vapor"
		}
	}

	return ""
}

// detectRubyFramework detects Ruby frameworks.
func detectRubyFramework(rootPath string) string {
	// Check for Rails
	if _, err := os.Stat(filepath.Join(rootPath, "config", "routes.rb")); err == nil {
		return "rails"
	}

	// Check Gemfile for Sinatra
	data, err := os.ReadFile(filepath.Join(rootPath, "Gemfile"))
	if err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "sinatra") {
			return "sinatra"
		}
		if strings.Contains(content, "rails") {
			return "rails"
		}
	}

	return ""
}
