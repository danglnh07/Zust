package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GitHub implementation
type GitHubProvider struct {
	ClientID     string
	ClientSecret string
}

func (g *GitHubProvider) Name() string {
	return "github"
}

func (g *GitHubProvider) ExchangeToken(code string) (*tokenResponse, error) {
	// Set request parameters
	reqParams := url.Values{}
	reqParams.Set("client_id", g.ClientID)
	reqParams.Set("client_secret", g.ClientSecret)
	reqParams.Set("code", code)
	reqParams.Set("scope", "read:user user:email")

	// Create request to access token endpoint
	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(reqParams.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Make request to access_token endpoint
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check for status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub token exchange failed: %s", string(body))
	}

	// Parse response body
	var githubToken *tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&githubToken); err != nil {
		return nil, err
	}
	return githubToken, nil
}

func (g *GitHubProvider) FetchUser(token string) (*userData, error) {
	// Make request to the userinfo endpoint
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Make request to the userinfo endpoint
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check for status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github user fetch failed: %s", string(body))
	}

	// Parse response
	var data struct {
		ID       int    `json:"id"`
		Username string `json:"login"`
		Avatar   string `json:"avatar_url"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &userData{
		ID:       fmt.Sprintf("%d", data.ID),
		Username: data.Username,
		Avatar:   data.Avatar,
		Email:    data.Email,
	}, nil
}
