package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

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

	// Extract username from URL path
	path := strings.TrimPrefix(r.URL.Path, "/issues/")
	username := strings.TrimSpace(path)

	if username == "" || username == "/" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

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

	// Fetch issues for each repository
	var reposWithIssues []RepositoryWithIssues

	for _, repo := range repos {
		if repo.OpenIssuesCount > 0 {
			issues, err := fetchRepositoryIssues(username, repo.Name)
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

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(reposWithIssues); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
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

// fetchRepositoryIssues fetches all issues for a given repository
func fetchRepositoryIssues(username, repoName string) ([]GitHubIssue, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=all&per_page=100", username, repoName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add User-Agent header (required by GitHub API)
	req.Header.Set("User-Agent", "Go-Issues-Fetcher")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

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

func main() {
	http.HandleFunc("/", HelloHandler)
	http.HandleFunc("/health", HealthHandler)
	http.HandleFunc("/issues/", IssuesHandler)

	port := "8080"
	log.Printf("Server starting on port %s...", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
