package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"zust/util"
)

// Google provider implementation
type GoogleProvider struct {
	ClientID     string
	ClientSecret string
}

func (g *GoogleProvider) Name() string {
	return "google"
}

func (g *GoogleProvider) ExchangeToken(code string) (*tokenResponse, error) {
	// Set request parameters
	reqParams := url.Values{}
	reqParams.Set("client_id", g.ClientID)
	reqParams.Set("client_secret", g.ClientSecret)
	reqParams.Set("code", code)
	reqParams.Set("grant_type", "authorization_code")
	reqParams.Set("redirect_uri", fmt.Sprintf("http://%s:%s/oauth2/callback", util.GetConfig().Domain, util.GetConfig().Port))

	// Create request to access token endpoint
	req, err := http.NewRequest("POST", "https://oauth2.googleapis.com/token", strings.NewReader(reqParams.Encode()))
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
		return nil, fmt.Errorf("google token exchange failed: %s", string(body))
	}

	// Parse response body
	var githubToken *tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&githubToken); err != nil {
		return nil, err
	}
	return githubToken, nil
}

func (g *GoogleProvider) FetchUser(token string) (*userData, error) {
	// Create request to the userinfo endpoint
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
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
		return nil, fmt.Errorf("google user fetch failed: %s", string(body))
	}

	// Parse response
	var data struct {
		ID       string `json:"id"`
		Username string `json:"name"`
		Avatar   string `json:"picture"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &userData{
		ID:       data.ID,
		Username: data.Username,
		Avatar:   data.Avatar,
		Email:    data.Email,
	}, nil
}
