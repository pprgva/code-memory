package search

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/pprgva/code-memory/config"
	"github.com/pprgva/code-memory/embedder"
	"github.com/pprgva/code-memory/store"
)

// FederatedResult extends SearchResult with project information.
type FederatedResult struct {
	store.SearchResult
	ProjectName string `json:"project_name"`
	ProjectPath string `json:"project_path"`
}

// FederatedSearcher searches across multiple projects and merges results.
type FederatedSearcher struct {
	embedder embedder.Embedder
}

// NewFederatedSearcher creates a new federated searcher.
func NewFederatedSearcher(emb embedder.Embedder) *FederatedSearcher {
	return &FederatedSearcher{
		embedder: emb,
	}
}

// projectSearchResult holds results from a single project search.
type projectSearchResult struct {
	ProjectName string
	ProjectPath string
	Results     []store.SearchResult
	Error       error
}

// Search searches across multiple projects and merges results using RRF.
// If projectNames is nil or empty, it searches all registered projects.
func (f *FederatedSearcher) Search(ctx context.Context, projectNames []string, query string, limit int) ([]FederatedResult, error) {
	// Load projects config
	projectsCfg, err := config.LoadProjectsConfig()
	if err != nil {
		return nil, err
	}

	// Determine which projects to search
	var projectsToSearch []string
	if len(projectNames) == 0 {
		// Search all projects
		projectsToSearch = projectsCfg.ListProjects()
	} else {
		projectsToSearch = projectNames
	}

	if len(projectsToSearch) == 0 {
		return []FederatedResult{}, nil
	}

	// Embed the query once
	queryVector, err := f.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	// Search projects in parallel
	resultsChan := make(chan projectSearchResult, len(projectsToSearch))
	var wg sync.WaitGroup

	for _, projectName := range projectsToSearch {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			result := f.searchProject(ctx, projectsCfg, name, queryVector, limit)
			resultsChan <- result
		}(projectName)
	}

	// Wait for all searches to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	var allResultSets [][]FederatedResult
	for result := range resultsChan {
		if result.Error != nil {
			// Log error but continue with other projects
			continue
		}
		if len(result.Results) > 0 {
			// Convert to FederatedResult
			fedResults := make([]FederatedResult, len(result.Results))
			for i, r := range result.Results {
				fedResults[i] = FederatedResult{
					SearchResult: r,
					ProjectName:  result.ProjectName,
					ProjectPath:  result.ProjectPath,
				}
			}
			allResultSets = append(allResultSets, fedResults)
		}
	}

	if len(allResultSets) == 0 {
		return []FederatedResult{}, nil
	}

	// Merge results using RRF
	return MergeResultsRRF(allResultSets, 60, limit), nil
}

// searchProject searches a single project and returns results.
func (f *FederatedSearcher) searchProject(ctx context.Context, projectsCfg *config.ProjectsConfig, projectName string, queryVector []float32, limit int) projectSearchResult {
	result := projectSearchResult{
		ProjectName: projectName,
	}

	// Get project from registry
	project, err := projectsCfg.GetProject(projectName)
	if err != nil {
		result.Error = err
		return result
	}

	// Expand path (convert ~ to absolute)
	absPath := config.ExpandPath(project.Path)
	result.ProjectPath = project.Path

	// Check if project has an index
	indexPath := filepath.Join(absPath, config.ConfigDir, config.IndexFileName)

	// Load the GOB store for this project
	gobStore := store.NewGOBStore(indexPath)
	if err := gobStore.Load(ctx); err != nil {
		result.Error = err
		return result
	}
	defer gobStore.Close()

	// Search in the store
	results, err := gobStore.Search(ctx, queryVector, limit)
	if err != nil {
		result.Error = err
		return result
	}

	// File paths in GOB store are relative to project root
	// The FederatedResult will provide project context

	result.Results = results
	return result
}
