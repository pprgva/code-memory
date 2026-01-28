package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProject_GoProject(t *testing.T) {
	// Create temp directory
	dir := t.TempDir()

	// Create go.mod
	goMod := `module example.com/myproject

go 1.21

require github.com/gin-gonic/gin v1.9.0
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectProject(dir)
	if err != nil {
		t.Fatalf("DetectProject failed: %v", err)
	}

	if info.Name != filepath.Base(dir) {
		t.Errorf("expected name %q, got %q", filepath.Base(dir), info.Name)
	}

	if !contains(info.Languages, "go") {
		t.Errorf("expected languages to contain 'go', got %v", info.Languages)
	}

	if info.BuildTool != "go" {
		t.Errorf("expected build tool 'go', got %q", info.BuildTool)
	}

	if info.Framework != "gin" {
		t.Errorf("expected framework 'gin', got %q", info.Framework)
	}
}

func TestDetectProject_PythonProject(t *testing.T) {
	dir := t.TempDir()

	// Create pyproject.toml with FastAPI
	pyproject := `[project]
name = "myapi"
dependencies = [
    "fastapi>=0.100.0",
    "uvicorn",
]
`
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(pyproject), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectProject(dir)
	if err != nil {
		t.Fatalf("DetectProject failed: %v", err)
	}

	if !contains(info.Languages, "python") {
		t.Errorf("expected languages to contain 'python', got %v", info.Languages)
	}

	if info.Framework != "fastapi" {
		t.Errorf("expected framework 'fastapi', got %q", info.Framework)
	}
}

func TestDetectProject_TypeScriptReactProject(t *testing.T) {
	dir := t.TempDir()

	// Create package.json with React
	packageJSON := `{
  "name": "my-react-app",
  "dependencies": {
    "react": "^18.0.0",
    "react-dom": "^18.0.0"
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Create tsconfig.json
	tsconfig := `{
  "compilerOptions": {
    "target": "ES2020"
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(tsconfig), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectProject(dir)
	if err != nil {
		t.Fatalf("DetectProject failed: %v", err)
	}

	if !contains(info.Languages, "typescript") {
		t.Errorf("expected languages to contain 'typescript', got %v", info.Languages)
	}

	if info.Framework != "react" {
		t.Errorf("expected framework 'react', got %q", info.Framework)
	}
}

func TestDetectProject_NextJSProject(t *testing.T) {
	dir := t.TempDir()

	// Create package.json with Next.js
	packageJSON := `{
  "name": "my-next-app",
  "dependencies": {
    "next": "^14.0.0",
    "react": "^18.0.0"
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Create tsconfig.json
	if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectProject(dir)
	if err != nil {
		t.Fatalf("DetectProject failed: %v", err)
	}

	// Next.js should be detected over React (more specific)
	if info.Framework != "next.js" {
		t.Errorf("expected framework 'next.js', got %q", info.Framework)
	}
}

func TestDetectProject_RustProject(t *testing.T) {
	dir := t.TempDir()

	// Create Cargo.toml
	cargoToml := `[package]
name = "myproject"
version = "0.1.0"
`
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargoToml), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectProject(dir)
	if err != nil {
		t.Fatalf("DetectProject failed: %v", err)
	}

	if !contains(info.Languages, "rust") {
		t.Errorf("expected languages to contain 'rust', got %v", info.Languages)
	}

	if info.BuildTool != "cargo" {
		t.Errorf("expected build tool 'cargo', got %q", info.BuildTool)
	}
}

func TestDetectProject_SwiftProject(t *testing.T) {
	dir := t.TempDir()

	// Create Package.swift
	packageSwift := `// swift-tools-version: 5.9
import PackageDescription
let package = Package(name: "MyApp")
`
	if err := os.WriteFile(filepath.Join(dir, "Package.swift"), []byte(packageSwift), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectProject(dir)
	if err != nil {
		t.Fatalf("DetectProject failed: %v", err)
	}

	if !contains(info.Languages, "swift") {
		t.Errorf("expected languages to contain 'swift', got %v", info.Languages)
	}

	if info.BuildTool != "spm" {
		t.Errorf("expected build tool 'spm', got %q", info.BuildTool)
	}
}

func TestDetectProject_SwiftUIProject(t *testing.T) {
	dir := t.TempDir()

	// Create a Swift file with SwiftUI import
	swiftFile := `import SwiftUI

struct ContentView: View {
    var body: some View {
        Text("Hello")
    }
}
`
	if err := os.WriteFile(filepath.Join(dir, "ContentView.swift"), []byte(swiftFile), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectProject(dir)
	if err != nil {
		t.Fatalf("DetectProject failed: %v", err)
	}

	if !contains(info.Languages, "swift") {
		t.Errorf("expected languages to contain 'swift', got %v", info.Languages)
	}

	if info.Framework != "swiftui" {
		t.Errorf("expected framework 'swiftui', got %q", info.Framework)
	}
}

func TestDetectProject_DjangoProject(t *testing.T) {
	dir := t.TempDir()

	// Create manage.py (Django indicator)
	managePy := `#!/usr/bin/env python
import django
`
	if err := os.WriteFile(filepath.Join(dir, "manage.py"), []byte(managePy), 0644); err != nil {
		t.Fatal(err)
	}

	// Create requirements.txt
	requirements := `django>=4.0
`
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(requirements), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectProject(dir)
	if err != nil {
		t.Fatalf("DetectProject failed: %v", err)
	}

	if !contains(info.Languages, "python") {
		t.Errorf("expected languages to contain 'python', got %v", info.Languages)
	}

	if info.Framework != "django" {
		t.Errorf("expected framework 'django', got %q", info.Framework)
	}
}

func TestDetectProject_RailsProject(t *testing.T) {
	dir := t.TempDir()

	// Create config/routes.rb (Rails indicator)
	configDir := filepath.Join(dir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	routesRb := `Rails.application.routes.draw do
end
`
	if err := os.WriteFile(filepath.Join(configDir, "routes.rb"), []byte(routesRb), 0644); err != nil {
		t.Fatal(err)
	}

	// Create Gemfile
	gemfile := `source 'https://rubygems.org'
gem 'rails'
`
	if err := os.WriteFile(filepath.Join(dir, "Gemfile"), []byte(gemfile), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectProject(dir)
	if err != nil {
		t.Fatalf("DetectProject failed: %v", err)
	}

	if !contains(info.Languages, "ruby") {
		t.Errorf("expected languages to contain 'ruby', got %v", info.Languages)
	}

	if info.Framework != "rails" {
		t.Errorf("expected framework 'rails', got %q", info.Framework)
	}
}

func TestDetectProject_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	info, err := DetectProject(dir)
	if err != nil {
		t.Fatalf("DetectProject failed: %v", err)
	}

	if info.Name != filepath.Base(dir) {
		t.Errorf("expected name %q, got %q", filepath.Base(dir), info.Name)
	}

	if len(info.Languages) != 0 {
		t.Errorf("expected no languages, got %v", info.Languages)
	}
}

func TestDetectProject_LanguageFromExtensions(t *testing.T) {
	dir := t.TempDir()

	// Create a Go file without go.mod
	goFile := `package main

func main() {
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(goFile), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectProject(dir)
	if err != nil {
		t.Fatalf("DetectProject failed: %v", err)
	}

	if !contains(info.Languages, "go") {
		t.Errorf("expected languages to contain 'go', got %v", info.Languages)
	}
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
