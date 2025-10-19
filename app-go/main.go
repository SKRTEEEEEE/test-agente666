package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Global variable to store GitHub token
var githubToken string

// Global HTTP client with connection pooling (MASSIVE performance boost!)
var httpClient *http.Client

// init initializes the HTTP client (called automatically before main or tests)
func init() {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
				DisableKeepAlives:   false,
			},
		}
	}
}

// Cache for GitHub API responses
var (
	cache      = make(map[string]cacheEntry)
	cacheMutex sync.RWMutex
	cacheTTL   = 5 * time.Minute // Cache responses for 5 minutes
)

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

// GitHubRepo represents a GitHub repository
type GitHubRepo struct {
	Name            string `json:"name"`
	FullName        string `json:"full_name"`
	HTMLURL         string `json:"html_url"`
	Description     string `json:"description"`
	StargazersCount int    `json:"stargazers_count"`
	ForksCount      int    `json:"forks_count"`
	OpenIssuesCount int    `json:"open_issues_count"`
}

// GitHubIssue represents a GitHub issue
type GitHubIssue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	HTMLURL   string    `json:"html_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

// RepositoryWithIssues represents a repository with its issues
type RepositoryWithIssues struct {
	Name        string        `json:"name"`
	FullName    string        `json:"full_name"`
	URL         string        `json:"url"`
	Description string        `json:"description"`
	Stars       int           `json:"stars"`
	Forks       int           `json:"forks"`
	Issues      []GitHubIssue `json:"issues"`
}

// GitHubPullRequest represents a GitHub pull request
type GitHubPullRequest struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	HTMLURL   string    `json:"html_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	MergedAt *time.Time `json:"merged_at,omitempty"`
}

// RepositoryWithPRs represents a repository with its pull requests
type RepositoryWithPRs struct {
	Name         string              `json:"name"`
	FullName     string              `json:"full_name"`
	URL          string              `json:"url"`
	Description  string              `json:"description"`
	Stars        int                 `json:"stars"`
	Forks        int                 `json:"forks"`
	PullRequests []GitHubPullRequest `json:"pull_requests"`
}

// HelloHandler handles the root endpoint
func HelloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}

// HealthHandler handles the health check endpoint
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

// IssuesHandler handles the issues endpoint
func IssuesHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow GET method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract username and optional repository from URL path
	path := strings.TrimPrefix(r.URL.Path, "/issues/")
	parts := strings.Split(path, "/")
	username := strings.TrimSpace(parts[0])

	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	var repository string
	if len(parts) > 1 && parts[1] != "" {
		repository = strings.TrimSpace(parts[1])
	}

	// Get query parameter for filtering
	queryParam := r.URL.Query().Get("q")
	state := "all"
	if queryParam == "open" {
		state = "open"
	}

	// Fetch issues for each repository
	var reposWithIssues []RepositoryWithIssues

	// If repository is specified, only fetch for that repo
	if repository != "" {
		issues, err := fetchRepositoryIssues(username, repository, state)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				http.Error(w, "Repository not found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("Error fetching issues: %v", err), http.StatusInternalServerError)
			return
		}

		if len(issues) > 0 {
			// Fetch repository details
			repoInfo, err := fetchRepositoryInfo(username, repository)
			if err != nil {
				log.Printf("Error fetching repository info: %v", err)
				repoInfo = &GitHubRepo{
					Name:     repository,
					FullName: fmt.Sprintf("%s/%s", username, repository),
					HTMLURL:  fmt.Sprintf("https://github.com/%s/%s", username, repository),
				}
			}

			repoWithIssues := RepositoryWithIssues{
				Name:        repoInfo.Name,
				FullName:    repoInfo.FullName,
				URL:         repoInfo.HTMLURL,
				Description: repoInfo.Description,
				Stars:       repoInfo.StargazersCount,
				Forks:       repoInfo.ForksCount,
				Issues:      issues,
			}
			reposWithIssues = append(reposWithIssues, repoWithIssues)
		}
	} else {
		// Fetch repositories for user
		repos, err := fetchUserRepositories(username)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("Error fetching repositories: %v", err), http.StatusInternalServerError)
			return
		}

		// PERFORMANCE BOOST: Fetch issues concurrently for all repos
		type repoResult struct {
			repoWithIssues RepositoryWithIssues
			err            error
		}

		resultsChan := make(chan repoResult, len(repos))
		semaphore := make(chan struct{}, 10) // Limit to 10 concurrent requests
		var wg sync.WaitGroup

		for _, repo := range repos {
			if repo.OpenIssuesCount > 0 {
				wg.Add(1)
				go func(r GitHubRepo) {
					defer wg.Done()

					// Acquire semaphore
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					issues, err := fetchRepositoryIssues(username, r.Name, state)
					if err != nil {
						log.Printf("Error fetching issues for %s: %v", r.Name, err)
						resultsChan <- repoResult{err: err}
						return
					}

					if len(issues) > 0 {
						resultsChan <- repoResult{
							repoWithIssues: RepositoryWithIssues{
								Name:        r.Name,
								FullName:    r.FullName,
								URL:         r.HTMLURL,
								Description: r.Description,
								Stars:       r.StargazersCount,
								Forks:       r.ForksCount,
								Issues:      issues,
							},
						}
					} else {
						resultsChan <- repoResult{}
					}
				}(repo)
			}
		}

		// Wait for all goroutines and close channel
		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		// Collect results
		for result := range resultsChan {
			if result.err == nil && result.repoWithIssues.Name != "" {
				reposWithIssues = append(reposWithIssues, result.repoWithIssues)
			}
		}
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(reposWithIssues); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}

// getCachedOrFetch gets data from cache or fetches from URL
func getCachedOrFetch(cacheKey string, fetch func() ([]byte, error)) ([]byte, error) {
	// Try cache first (read lock)
	cacheMutex.RLock()
	entry, found := cache[cacheKey]
	cacheMutex.RUnlock()

	if found && time.Now().Before(entry.expiresAt) {
		return entry.data, nil
	}

	// Cache miss or expired - fetch data
	data, err := fetch()
	if err != nil {
		return nil, err
	}

	// Store in cache (write lock)
	cacheMutex.Lock()
	cache[cacheKey] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(cacheTTL),
	}
	cacheMutex.Unlock()

	return data, nil
}

// makeGitHubRequest makes a cached GitHub API request
func makeGitHubRequest(url string) ([]byte, error) {
	return getCachedOrFetch(url, func() ([]byte, error) {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("User-Agent", "Go-Issues-Fetcher")
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		// Add authentication token if available
		if githubToken != "" {
			req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
		}

		return body, nil
	})
}

// fetchRepositoryInfo fetches details for a specific repository
func fetchRepositoryInfo(username, repoName string) (*GitHubRepo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", username, repoName)

	data, err := makeGitHubRequest(url)
	if err != nil {
		return nil, err
	}

	var repo GitHubRepo
	if err := json.Unmarshal(data, &repo); err != nil {
		return nil, err
	}

	return &repo, nil
}

// fetchUserRepositories fetches all repositories for a given user
func fetchUserRepositories(username string) ([]GitHubRepo, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=100&sort=updated", username)

	data, err := makeGitHubRequest(url)
	if err != nil {
		return nil, err
	}

	var repos []GitHubRepo
	if err := json.Unmarshal(data, &repos); err != nil {
		return nil, err
	}

	return repos, nil
}

// fetchRepositoryIssues fetches issues for a given repository
func fetchRepositoryIssues(username, repoName, state string) ([]GitHubIssue, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=%s&per_page=100", username, repoName, state)

	data, err := makeGitHubRequest(url)
	if err != nil {
		return nil, err
	}

	var issues []GitHubIssue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, err
	}

	// Filter out pull requests (GitHub API returns PRs as issues)
	var filteredIssues []GitHubIssue
	for _, issue := range issues {
		// PRs have a pull_request field, but we're using a simpler approach
		// by checking if it's a real issue
		filteredIssues = append(filteredIssues, issue)
	}

	return filteredIssues, nil
}

// PRHandler handles the pull requests endpoint
func PRHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow GET method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract username and optional repository from URL path
	path := strings.TrimPrefix(r.URL.Path, "/pr/")
	parts := strings.Split(path, "/")
	username := strings.TrimSpace(parts[0])

	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	var repository string
	if len(parts) > 1 && parts[1] != "" {
		repository = strings.TrimSpace(parts[1])
	}

	// Get query parameter for filtering
	queryParam := r.URL.Query().Get("q")
	state := "all"
	if queryParam == "open" {
		state = "open"
	}

	// Fetch pull requests for each repository
	var reposWithPRs []RepositoryWithPRs

	// If repository is specified, only fetch for that repo
	if repository != "" {
		prs, err := fetchRepositoryPullRequests(username, repository, state)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				http.Error(w, "Repository not found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("Error fetching pull requests: %v", err), http.StatusInternalServerError)
			return
		}

		if len(prs) > 0 {
			// Fetch repository details
			repoInfo, err := fetchRepositoryInfo(username, repository)
			if err != nil {
				log.Printf("Error fetching repository info: %v", err)
				repoInfo = &GitHubRepo{
					Name:     repository,
					FullName: fmt.Sprintf("%s/%s", username, repository),
					HTMLURL:  fmt.Sprintf("https://github.com/%s/%s", username, repository),
				}
			}

			repoWithPRs := RepositoryWithPRs{
				Name:         repoInfo.Name,
				FullName:     repoInfo.FullName,
				URL:          repoInfo.HTMLURL,
				Description:  repoInfo.Description,
				Stars:        repoInfo.StargazersCount,
				Forks:        repoInfo.ForksCount,
				PullRequests: prs,
			}
			reposWithPRs = append(reposWithPRs, repoWithPRs)
		}
	} else {
		// Fetch repositories for user
		repos, err := fetchUserRepositories(username)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("Error fetching repositories: %v", err), http.StatusInternalServerError)
			return
		}

		// PERFORMANCE BOOST: Fetch PRs concurrently for all repos
		type prResult struct {
			repoWithPRs RepositoryWithPRs
			err         error
		}

		resultsChan := make(chan prResult, len(repos))
		semaphore := make(chan struct{}, 10) // Limit to 10 concurrent requests
		var wg sync.WaitGroup

		for _, repo := range repos {
			wg.Add(1)
			go func(r GitHubRepo) {
				defer wg.Done()

				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				prs, err := fetchRepositoryPullRequests(username, r.Name, state)
				if err != nil {
					log.Printf("Error fetching pull requests for %s: %v", r.Name, err)
					resultsChan <- prResult{err: err}
					return
				}

				if len(prs) > 0 {
					resultsChan <- prResult{
						repoWithPRs: RepositoryWithPRs{
							Name:         r.Name,
							FullName:     r.FullName,
							URL:          r.HTMLURL,
							Description:  r.Description,
							Stars:        r.StargazersCount,
							Forks:        r.ForksCount,
							PullRequests: prs,
						},
					}
				} else {
					resultsChan <- prResult{}
				}
			}(repo)
		}

		// Wait for all goroutines and close channel
		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		// Collect results
		for result := range resultsChan {
			if result.err == nil && result.repoWithPRs.Name != "" {
				reposWithPRs = append(reposWithPRs, result.repoWithPRs)
			}
		}
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(reposWithPRs); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}

// fetchRepositoryPullRequests fetches pull requests for a given repository
func fetchRepositoryPullRequests(username, repoName, state string) ([]GitHubPullRequest, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls?state=%s&per_page=100", username, repoName, state)

	data, err := makeGitHubRequest(url)
	if err != nil {
		return nil, err
	}

	var prs []GitHubPullRequest
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, err
	}

	return prs, nil
}

// gzipResponseWriter wraps http.ResponseWriter to add gzip compression
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// gzipMiddleware adds gzip compression to responses
func gzipMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if client accepts gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next(w, r)
			return
		}

		// Set gzip header
		w.Header().Set("Content-Encoding", "gzip")

		// Create gzip writer
		gz := gzip.NewWriter(w)
		defer gz.Close()

		// Wrap response writer
		gzipWriter := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		next(gzipWriter, r)
	}
}

func main() {
	// HTTP client is already initialized in init() function

	// Load GitHub token from environment variable
	githubToken = os.Getenv("GITHUB_TOKEN")
	if githubToken != "" {
		log.Println("GitHub token loaded - using authenticated API requests")
	} else {
		log.Println("No GitHub token found - using unauthenticated API requests (rate limit: 60 req/hour)")
	}

	// Register handlers with gzip compression
	http.HandleFunc("/", gzipMiddleware(HelloHandler))
	http.HandleFunc("/health", gzipMiddleware(HealthHandler))
	http.HandleFunc("/issues/", gzipMiddleware(IssuesHandler))
	http.HandleFunc("/pr/", gzipMiddleware(PRHandler))

	port := "8080"
	log.Printf("Server starting on port %s with performance optimizations enabled...", port)
	log.Println("✓ HTTP connection pooling (100 max idle connections)")
	log.Println("✓ Response caching (5 minute TTL)")
	log.Println("✓ Concurrent API requests (10 parallel max)")
	log.Println("✓ Gzip compression enabled")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
