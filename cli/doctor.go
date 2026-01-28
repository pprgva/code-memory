package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pprgva/code-memory/config"
	"github.com/pprgva/code-memory/embedder"
	"github.com/pprgva/code-memory/store"
	"github.com/spf13/cobra"
)

var doctorJSON bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the health of your grepai installation",
	Long:  "Runs diagnostic checks on config, Python, venv, model, embedder, store, and index.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output results as JSON")
}

type checkResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // OK, FAIL, SKIP
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

func runDoctor() error {
	var results []checkResult

	// 1. Config
	projectRoot, cfg, configResult := checkConfig()
	results = append(results, configResult)
	configOK := configResult.Status == "OK"

	// 2. Python
	pythonResult := checkPython(configOK)
	results = append(results, pythonResult)
	pythonOK := pythonResult.Status == "OK"

	// 3. Venv
	venvResult := checkVenv(pythonOK)
	results = append(results, venvResult)
	venvOK := venvResult.Status == "OK"

	// 4. Dependencies
	depsResult := checkDependencies(venvOK)
	results = append(results, depsResult)

	// 5. Model
	modelResult := checkModel(configOK, cfg)
	results = append(results, modelResult)

	// 6. Embedder
	embedderResult := checkEmbedder(venvOK && modelResult.Status == "OK", cfg)
	results = append(results, embedderResult)

	// 7. Store
	storeInstance, storeResult := checkStore(configOK, projectRoot, cfg)
	results = append(results, storeResult)
	storeOK := storeResult.Status == "OK"

	// 8. Index
	indexResult := checkIndex(storeOK, storeInstance)
	results = append(results, indexResult)

	// Fermer le store si ouvert
	if storeInstance != nil {
		storeInstance.Close()
	}

	// Affichage
	if doctorJSON {
		return outputDoctorJSON(results)
	}
	return outputDoctorText(results)
}

func checkConfig() (string, *config.Config, checkResult) {
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		cwd, _ := os.Getwd()
		return "", nil, checkResult{
			Name:    "Config",
			Status:  "FAIL",
			Message: fmt.Sprintf("no .grepai/config.yaml in %s (or parents)", cwd),
			Hint:    "Run 'grepai init' in your project to initialize it",
		}
	}
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return projectRoot, nil, checkResult{
			Name:    "Config",
			Status:  "FAIL",
			Message: fmt.Sprintf("failed to load: %v", err),
		}
	}
	return projectRoot, cfg, checkResult{
		Name:    "Config",
		Status:  "OK",
		Message: ".grepai/config.yaml found",
	}
}

func checkPython(configOK bool) checkResult {
	if !configOK {
		return checkResult{Name: "Python", Status: "SKIP", Message: "(requires config)"}
	}
	out, err := exec.Command("python3", "--version").Output()
	if err != nil {
		return checkResult{
			Name:    "Python",
			Status:  "FAIL",
			Message: "python3 not found",
			Hint:    "Install Python 3.9+",
		}
	}
	version := strings.TrimSpace(string(out))
	return checkResult{Name: "Python", Status: "OK", Message: version}
}

func checkVenv(pythonOK bool) checkResult {
	if !pythonOK {
		return checkResult{Name: "Venv", Status: "SKIP", Message: "(requires Python)"}
	}
	venvDir := config.GetVenvDir()
	if !embedder.VenvExists(venvDir) {
		return checkResult{
			Name:    "Venv",
			Status:  "FAIL",
			Message: fmt.Sprintf("%s not found", venvDir),
			Hint:    "Run 'grepai init --yes' to create it",
		}
	}
	return checkResult{
		Name:    "Venv",
		Status:  "OK",
		Message: embedder.VenvPythonPath(venvDir),
	}
}

func checkDependencies(venvOK bool) checkResult {
	if !venvOK {
		return checkResult{Name: "Dependencies", Status: "SKIP", Message: "(requires venv)"}
	}
	venvDir := config.GetVenvDir()
	pythonPath := embedder.VenvPythonPath(venvDir)
	script := `import torch; import transformers; print(f"torch {torch.__version__}, transformers {transformers.__version__}")`
	out, err := exec.Command(pythonPath, "-c", script).Output()
	if err != nil {
		return checkResult{
			Name:    "Dependencies",
			Status:  "FAIL",
			Message: "torch or transformers not importable",
			Hint:    "Run 'grepai init --yes' to install dependencies",
		}
	}
	return checkResult{
		Name:    "Dependencies",
		Status:  "OK",
		Message: strings.TrimSpace(string(out)),
	}
}

func checkModel(configOK bool, cfg *config.Config) checkResult {
	if !configOK || cfg == nil {
		return checkResult{Name: "Model", Status: "SKIP", Message: "(requires config)"}
	}
	modelPath := cfg.Embedder.ModelPath
	if modelPath == "" {
		modelPath = embedder.DefaultModelDir()
	}
	if !embedder.ModelExists(modelPath) {
		return checkResult{
			Name:    "Model",
			Status:  "FAIL",
			Message: fmt.Sprintf("%s not found or missing config.json", modelPath),
			Hint:    "Run 'grepai init --yes' to download the model",
		}
	}
	dims := cfg.Embedder.Dimensions
	if dims == 0 {
		dims = 1024
	}
	return checkResult{
		Name:    "Model",
		Status:  "OK",
		Message: fmt.Sprintf("multilingual-e5-large (%d dims)", dims),
	}
}

func checkEmbedder(ready bool, cfg *config.Config) checkResult {
	if !ready || cfg == nil {
		return checkResult{Name: "Embedder", Status: "SKIP", Message: "(requires venv + model)"}
	}

	// Essayer le socket daemon d'abord
	socketEmb, socketErr := embedder.NewSocketEmbedder()
	if socketErr == nil {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		vec, err := socketEmb.Embed(ctx, "doctor test")
		socketEmb.Close()
		if err != nil {
			return checkResult{Name: "Embedder", Status: "FAIL", Message: fmt.Sprintf("socket embed failed: %v", err)}
		}
		elapsed := time.Since(start)
		expectedDims := cfg.Embedder.Dimensions
		if expectedDims == 0 {
			expectedDims = 1024
		}
		if len(vec) != expectedDims {
			return checkResult{Name: "Embedder", Status: "FAIL", Message: fmt.Sprintf("expected %d dims, got %d", expectedDims, len(vec))}
		}
		return checkResult{Name: "Embedder", Status: "OK", Message: fmt.Sprintf("daemon socket (%dms)", elapsed.Milliseconds())}
	}

	modelPath := cfg.Embedder.ModelPath
	if modelPath == "" {
		modelPath = embedder.DefaultModelDir()
	}
	venvDir := config.GetVenvDir()

	start := time.Now()
	e, err := embedder.NewE5Embedder(modelPath, venvDir)
	if err != nil {
		return checkResult{
			Name:    "Embedder",
			Status:  "FAIL",
			Message: fmt.Sprintf("worker failed to start: %v", err),
		}
	}
	defer e.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	vec, err := e.Embed(ctx, "doctor test")
	if err != nil {
		return checkResult{
			Name:    "Embedder",
			Status:  "FAIL",
			Message: fmt.Sprintf("embed failed: %v", err),
		}
	}
	elapsed := time.Since(start)

	expectedDims := cfg.Embedder.Dimensions
	if expectedDims == 0 {
		expectedDims = 1024
	}
	if len(vec) != expectedDims {
		return checkResult{
			Name:    "Embedder",
			Status:  "FAIL",
			Message: fmt.Sprintf("expected %d dims, got %d", expectedDims, len(vec)),
		}
	}

	return checkResult{
		Name:    "Embedder",
		Status:  "OK",
		Message: fmt.Sprintf("E5 worker responds (%dms)", elapsed.Milliseconds()),
	}
}

func checkStore(configOK bool, projectRoot string, cfg *config.Config) (store.VectorStore, checkResult) {
	if !configOK || cfg == nil {
		return nil, checkResult{Name: "Store", Status: "SKIP", Message: "(requires config)"}
	}

	ctx := context.Background()
	backend := cfg.Store.Backend

	switch backend {
	case "gob":
		indexPath := config.GetIndexPath(projectRoot)
		info, err := os.Stat(indexPath)
		if err != nil {
			return nil, checkResult{
				Name:    "Store",
				Status:  "FAIL",
				Message: "index.gob not found",
				Hint:    "Run 'grepai watch' to create the index",
			}
		}
		sizeMB := float64(info.Size()) / (1024 * 1024)
		s := store.NewGOBStore(indexPath)
		if err := s.Load(ctx); err != nil {
			return nil, checkResult{
				Name:    "Store",
				Status:  "FAIL",
				Message: fmt.Sprintf("failed to load gob: %v", err),
			}
		}
		return s, checkResult{
			Name:    "Store",
			Status:  "OK",
			Message: fmt.Sprintf("gob (index.gob, %.1f MB)", sizeMB),
		}

	case "postgres":
		dims := cfg.Embedder.Dimensions
		if dims == 0 {
			dims = 1024
		}
		s, err := store.NewPostgresStore(ctx, cfg.Store.Postgres.DSN, projectRoot, dims)
		if err != nil {
			return nil, checkResult{
				Name:    "Store",
				Status:  "FAIL",
				Message: fmt.Sprintf("postgres connection failed: %v", err),
			}
		}
		// Extraire host:port du DSN pour l'affichage
		dsn := cfg.Store.Postgres.DSN
		return s, checkResult{
			Name:    "Store",
			Status:  "OK",
			Message: fmt.Sprintf("postgres (%s)", dsn),
		}

	case "qdrant":
		endpoint := cfg.Store.Qdrant.Endpoint
		port := cfg.Store.Qdrant.Port
		if port <= 0 {
			port = 6334
		}
		dims := cfg.Embedder.Dimensions
		if dims == 0 {
			dims = 1024
		}
		collection := cfg.Store.Qdrant.Collection
		if collection == "" {
			collection = store.SanitizeCollectionName(projectRoot)
		}
		s, err := store.NewQdrantStore(ctx, endpoint, port, cfg.Store.Qdrant.UseTLS, collection, cfg.Store.Qdrant.APIKey, dims)
		if err != nil {
			return nil, checkResult{
				Name:    "Store",
				Status:  "FAIL",
				Message: fmt.Sprintf("qdrant connection failed: %v", err),
			}
		}
		return s, checkResult{
			Name:    "Store",
			Status:  "OK",
			Message: fmt.Sprintf("qdrant (%s:%d)", endpoint, port),
		}

	default:
		return nil, checkResult{
			Name:    "Store",
			Status:  "FAIL",
			Message: fmt.Sprintf("unknown backend: %s", backend),
		}
	}
}

func checkIndex(storeOK bool, s store.VectorStore) checkResult {
	if !storeOK || s == nil {
		return checkResult{Name: "Index", Status: "SKIP", Message: "(requires store)"}
	}
	ctx := context.Background()
	stats, err := s.GetStats(ctx)
	if err != nil {
		return checkResult{
			Name:    "Index",
			Status:  "FAIL",
			Message: fmt.Sprintf("failed to get stats: %v", err),
		}
	}
	return checkResult{
		Name:    "Index",
		Status:  "OK",
		Message: fmt.Sprintf("%d files, %d chunks", stats.TotalFiles, stats.TotalChunks),
	}
}

func outputDoctorJSON(results []checkResult) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func outputDoctorText(results []checkResult) error {
	fmt.Println()
	fmt.Println("grepai doctor")
	fmt.Println()

	okCount, failCount, skipCount := 0, 0, 0
	for _, r := range results {
		var tag string
		switch r.Status {
		case "OK":
			tag = "[OK]"
			okCount++
		case "FAIL":
			tag = "[FAIL]"
			failCount++
		case "SKIP":
			tag = "[SKIP]"
			skipCount++
		}
		fmt.Printf("%-6s %-16s %s\n", tag, r.Name, r.Message)
		if r.Hint != "" {
			fmt.Printf("%-6s %-16s → %s\n", "", "", r.Hint)
		}
	}

	fmt.Println()
	if failCount == 0 && skipCount == 0 {
		fmt.Println("All checks passed!")
	} else {
		parts := []string{}
		if okCount > 0 {
			parts = append(parts, fmt.Sprintf("%d passed", okCount))
		}
		if skipCount > 0 {
			parts = append(parts, fmt.Sprintf("%d skipped", skipCount))
		}
		if failCount > 0 {
			parts = append(parts, fmt.Sprintf("%d failed", failCount))
		}
		fmt.Println(strings.Join(parts, ", ") + ".")
	}

	return nil
}
