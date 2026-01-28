package embedder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// VenvExists vérifie si le venv Python existe déjà.
func VenvExists(venvDir string) bool {
	_, err := os.Stat(VenvPythonPath(venvDir))
	return err == nil
}

// VenvPythonPath retourne le chemin vers le binaire python3 du venv.
func VenvPythonPath(venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "python.exe")
	}
	return filepath.Join(venvDir, "bin", "python3")
}

// SetupVenv crée un venv Python dans le répertoire spécifié.
func SetupVenv(venvDir, pythonPath string) error {
	if pythonPath == "" {
		pythonPath = "python3"
	}

	cmd := exec.Command(pythonPath, "-m", "venv", venvDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create venv: %w", err)
	}
	return nil
}

// InstallDeps installe torch et transformers dans le venv.
func InstallDeps(venvDir string) error {
	pipPath := filepath.Join(venvDir, "bin", "pip")
	if runtime.GOOS == "windows" {
		pipPath = filepath.Join(venvDir, "Scripts", "pip.exe")
	}

	cmd := exec.Command(pipPath, "install", "torch", "transformers")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}
	return nil
}
