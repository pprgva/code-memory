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

// DefaultModelName est le modèle HuggingFace téléchargé par défaut.
const DefaultModelName = "intfloat/multilingual-e5-large"

// DefaultModelDir retourne le chemin global du modèle : ~/.local/share/grepai/models/<nom>
func DefaultModelDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	// Nom du dossier = partie après le "/" du repo HuggingFace
	return filepath.Join(home, ".local", "share", "grepai", "models", "multilingual-e5-large")
}

// ModelExists vérifie si le modèle est déjà téléchargé.
func ModelExists(modelPath string) bool {
	_, err := os.Stat(filepath.Join(modelPath, "config.json"))
	return err == nil
}

// DownloadModel télécharge le modèle depuis HuggingFace via le venv Python.
func DownloadModel(venvDir, modelPath, modelName string) error {
	if modelName == "" {
		modelName = DefaultModelName
	}

	if err := os.MkdirAll(modelPath, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	pythonPath := VenvPythonPath(venvDir)
	script := fmt.Sprintf(
		`from huggingface_hub import snapshot_download; snapshot_download(repo_id="%s", local_dir="%s")`,
		modelName, modelPath,
	)

	cmd := exec.Command(pythonPath, "-c", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download model %s: %w", modelName, err)
	}
	return nil
}
