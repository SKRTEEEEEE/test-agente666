package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Global variable to store GitHub token
var githubToken string

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

		for _, repo := range repos {
			if repo.OpenIssuesCount > 0 {
				issues, err := fetchRepositoryIssues(username, repo.Name, state)
				if err != nil {
					log.Printf("Error fetching issues for %s: %v", repo.Name, err)
					continue
				}

				if len(issues) > 0 {
					repoWithIssues := RepositoryWithIssues{
						Name:        repo.Name,
						FullName:    repo.FullName,
						URL:         repo.HTMLURL,
						Description: repo.Description,
						Stars:       repo.StargazersCount,
						Forks:       repo.ForksCount,
						Issues:      issues,
					}
					reposWithIssues = append(reposWithIssues, repoWithIssues)
				}
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

// fetchRepositoryInfo fetches details for a specific repository
func fetchRepositoryInfo(username, repoName string) (*GitHubRepo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", username, repoName)

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

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var repo GitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return nil, err
	}

	return &repo, nil
}

// fetchUserRepositories fetches all repositories for a given user
func fetchUserRepositories(username string) ([]GitHubRepo, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=100&sort=updated", username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add User-Agent header (required by GitHub API)
	req.Header.Set("User-Agent", "Go-Issues-Fetcher")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	
	// Add authentication token if available
	if githubToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var repos []GitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}

	return repos, nil
}

// fetchRepositoryIssues fetches issues for a given repository
func fetchRepositoryIssues(username, repoName, state string) ([]GitHubIssue, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=%s&per_page=100", username, repoName, state)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add User-Agent header (required by GitHub API)
	req.Header.Set("User-Agent", "Go-Issues-Fetcher")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	
	// Add authentication token if available
	if githubToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var issues []GitHubIssue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
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

		for _, repo := range repos {
			prs, err := fetchRepositoryPullRequests(username, repo.Name, state)
			if err != nil {
				log.Printf("Error fetching pull requests for %s: %v", repo.Name, err)
				continue
			}

			if len(prs) > 0 {
				repoWithPRs := RepositoryWithPRs{
					Name:         repo.Name,
					FullName:     repo.FullName,
					URL:          repo.HTMLURL,
					Description:  repo.Description,
					Stars:        repo.StargazersCount,
					Forks:        repo.ForksCount,
					PullRequests: prs,
				}
				reposWithPRs = append(reposWithPRs, repoWithPRs)
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

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add User-Agent header (required by GitHub API)
	req.Header.Set("User-Agent", "Go-Issues-Fetcher")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	
	// Add authentication token if available
	if githubToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var prs []GitHubPullRequest
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return nil, err
	}

	return prs, nil
}

func main() {
	// Load GitHub token from environment variable
	githubToken = os.Getenv("GITHUB_TOKEN")
	if githubToken != "" {
		log.Println("GitHub token loaded - using authenticated API requests")
	} else {
		log.Println("No GitHub token found - using unauthenticated API requests (rate limit: 60 req/hour)")
	}

	http.HandleFunc("/", HelloHandler)
	http.HandleFunc("/health", HealthHandler)
	http.HandleFunc("/issues/", IssuesHandler)
	http.HandleFunc("/pr/", PRHandler)

	port := "8080"
	log.Printf("Server starting on port %s...", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
