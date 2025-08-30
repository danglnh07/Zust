package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	db "zust/db/sqlc"
	"zust/service"
	"zust/util"

	"github.com/google/uuid"
)

/*=== PASSWORD AUTH HANDLERS ===*/

// Request body for login
type loginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// Response body for login
type loginResponse struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Avatar       string `json:"avatar"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// HandleLogin handles the login with username and password
func (server *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	/*
	 * POST auth/login
	 * Success: 200 OK
	 * Error: 400 Bad Request, 403 Forbidden, 500 Internal Server Error
	 */

	// Extract the request body
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.logger.Error("POST /login: failed to decode request body", "error", err)
		server.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate the request body
	if err := server.validate.Struct(&req); err != nil {
		server.logger.Error("POST /login: invalid request body", "error", err)
		server.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get account by username
	account, err := server.query.GetAccountByUsername(r.Context(), req.Username)
	if err != nil {
		// If no account found with the username
		if errors.Is(err, sql.ErrNoRows) {
			server.WriteError(w, http.StatusBadRequest, "Invalid username or password")
			return
		}

		// Other database error
		server.logger.Error("POST /login: failed to get account by username", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// If the account status is not active
	if account.Status != db.AccountStatusActive {
		server.WriteError(w, http.StatusForbidden, "Account is not active")
		return
	}

	// If the account does not have a password (OAuth account)
	if !account.Password.Valid {
		server.WriteError(w, http.StatusBadRequest, "Account does not have a password, please login with OAuth provider")
		return
	}

	// Check if the password is correct
	if !util.BcryptCompare(account.Password.String, req.Password) {
		server.WriteError(w, http.StatusBadRequest, "Invalid username or password")
		return
	}

	// If success, create JWT tokens (access token and refresh token)
	accessToken, err := server.jwtService.CreateToken(account.AccountID.String(), "access-token",
		int(account.TokenVersion), server.jwtService.TokenExpirationTime)
	if err != nil {
		server.logger.Error("POST /login: failed to create JWT access token", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	refreshToken, err := server.jwtService.CreateToken(account.AccountID.String(), "refresh-token",
		int(account.TokenVersion), server.jwtService.RefreshTokenExpirationTime)
	if err != nil {
		server.logger.Error("POST /login: failed to create JWT refresh token", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Return user info and tokens
	var resp = loginResponse{
		ID:           account.AccountID.String(),
		Email:        account.Email,
		Username:     account.Username,
		Avatar:       service.GenerateMediaLink(account.AccountID.String(), "avatar", "avatar.png"),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	server.WriteJSON(w, http.StatusOK, resp)
}

// Request body for register
type registerRequest struct {
	Email    string `json:"email" validate:"required,email,max=40"`
	Username string `json:"username" validate:"required,max=20"`
	Password string `json:"password" validate:"required"`
}

// HandleRegister handles the register with email, username and password
func (server *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	/*
	 * POST auth/register
	 * Success: 200 OK
	 * Error: 400 Bad Request, 500 Internal Server Error
	 */

	// Extract the request body
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.logger.Error("POST /register: failed to decode request body", "error", err)
		server.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate the request body
	if err := server.validate.Struct(&req); err != nil {
		server.logger.Error("POST /register: invalid request body", "error", err)
		server.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Hash the password
	hashedPassword, err := util.BcryptHash(req.Password)
	if err != nil {
		server.logger.Error("POST /register: failed to hash password", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Create account
	account, err := server.query.CreateAccountWithPassword(r.Context(), db.CreateAccountWithPasswordParams{
		Email:    req.Email,
		Username: req.Username,
		Password: sql.NullString{String: hashedPassword, Valid: true},
	})
	if err != nil {
		// If the email is already taken
		if strings.Contains(err.Error(), "accounts_email_key") {
			server.WriteError(w, http.StatusBadRequest, "Email is already taken")
			return
		}

		// If the username is already taken
		if strings.Contains(err.Error(), "accounts_username_key") {
			server.WriteError(w, http.StatusBadRequest, "Username is already taken")
			return
		}

		// Other database error
		server.logger.Error("POST /register: failed to create account", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Failed to create account")
		return
	}

	// Create user repository with default avatar and cover
	err = server.storage.CreateUserRepo(account.AccountID.String())
	if err != nil {
		server.logger.Error("POST /auth/register: failed to create user repository", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Send verification email
	if err := server.sendVerificationEmail(account.AccountID.String(), account.Username, account.Email); err != nil {
		server.logger.Error("POST /register: failed to send verification email", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Account created successfully, but failed to send verification email")
		return
	}

	server.WriteJSON(w, http.StatusOK, "Account created successfully")
}

func (server *Server) sendVerificationEmail(id, username, email string) error {
	// Get configurations
	config := util.GetConfig()

	// Generate token: userID|timestamp and encode it with base64
	token := util.Encode(fmt.Sprintf("%s|%d", id, time.Now().UnixNano()))

	// Prepare email body
	body, err := server.mailService.PrepareEmail(service.VerificationEmailData{
		Username: username,
		Link:     fmt.Sprintf("http://%s:%s/auth/verification?token=%s", config.Domain, config.Port, token),
	})
	if err != nil {
		return err
	}

	// Send email
	return server.mailService.SendEmail(email, "Zust - Verify your email", body)
}

func (server *Server) HandleVerify(w http.ResponseWriter, r *http.Request) {
	/*
	 * GET auth/verification?token=...
	 * Success: 200 OK
	 * Error: 400 Bad Request, 500 Internal Server Error
	 */

	// Get the token from query params
	token := r.URL.Query().Get("token")
	if token == "" {
		server.WriteError(w, http.StatusBadRequest, "Missing token")
		return
	}

	// Decode the token to get the account ID
	decodeToken := util.Decode(token)

	// Split the decoded string to get the account ID and timestamp
	parts := strings.Split(decodeToken, "|")
	if len(parts) != 2 {
		server.WriteError(w, http.StatusBadRequest, "Invalid token")
		return
	}
	accountID := parts[0]

	// Check if the token is expired (valid for 24 hours)
	timestamp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		server.WriteError(w, http.StatusBadRequest, "Invalid token")
		return
	}
	// Since the timestamp is generated by UnixNano(), the sec parameter should be in 0 to get the correct time
	if time.Since(time.Unix(0, timestamp)) > 24*time.Hour {
		server.WriteError(w, http.StatusBadRequest, "Token has expired")
		return
	}

	// Activate the account
	var uuid uuid.UUID
	if err := uuid.Scan(accountID); err != nil {
		server.WriteError(w, http.StatusBadRequest, "Invalid account ID")
		return
	}

	err = server.query.ActivateAccount(r.Context(), uuid)
	if err != nil {
		// If no account found with the account ID
		if errors.Is(err, sql.ErrNoRows) {
			server.WriteError(w, http.StatusBadRequest, "Account does not exist")
			return
		}

		// Other database error
		server.logger.Error("GET /verification: failed to activate account", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Failed to verify account")
		return
	}

	server.WriteJSON(w, http.StatusOK, "Account verified successfully")
}

func (server *Server) HandleResendVerification(w http.ResponseWriter, r *http.Request) {
	/*
	 * POST auth/verification/resend?email=...
	 * Success: 200 OK
	 * Error: 400 Bad Request, 500 Internal Server Error
	 */

	// Get the email from query params
	email := r.URL.Query().Get("email")
	if email == "" {
		server.WriteError(w, http.StatusBadRequest, "Missing email")
		return
	}

	// Get account by email
	account, err := server.query.GetAccountByEmail(r.Context(), email)
	if err != nil {
		// If no account found with the email
		if errors.Is(err, sql.ErrNoRows) {
			server.WriteError(w, http.StatusBadRequest, "Account with this email does not exist")
			return
		}

		// Other database error
		server.logger.Error("POST /verification/resend: failed to get account by email", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Check for account status
	if account.Status != db.AccountStatusInactive {
		server.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Account is %s", account.Status))
		return
	}

	// Send verification email
	if err := server.sendVerificationEmail(account.AccountID.String(), account.Username, account.Email); err != nil {
		server.logger.Error("POST /verification/resend: failed to send verification email", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Failed to send verification email")
		return
	}

	server.WriteJSON(w, http.StatusOK, "Verification email sent successfully")
}

/*=== OAUTH2 AUTH HANDLERS ===*/

// Response of when exchange the code for access token return by OAuth provider
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// User data needed that we fetch from OAuth provider
type userData struct {
	ID       string
	Username string
	Avatar   string
	Email    string
}

// Interface for each OAuth provider
type OAuthProvider interface {
	Name() string
	ExchangeToken(code string) (*tokenResponse, error)
	FetchUser(token string) (*userData, error)
}

// HandleCallback handles the OAuth callback from provider
func (server *Server) HandleCallback(w http.ResponseWriter, r *http.Request) {
	/*
	 * GET /oauth2/callback?code=...&state=...
	 * Success: 200 OK
	 * Error: 400 Bad Request, 500 Internal Server Error
	 */

	// Get the OAuth provider
	providerName := r.URL.Query().Get("state")
	var provider OAuthProvider

	// For each provider, fecth the client ID and client secret from the config
	switch providerName {
	case "github":
		cfg := util.GetConfig()
		provider = &GitHubProvider{
			ClientID:     cfg.GithubClientID,
			ClientSecret: cfg.GithubClientSecret,
		}
	case "google":
		cfg := util.GetConfig()
		provider = &GoogleProvider{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
		}
	default:
		server.WriteError(w, http.StatusBadRequest, "Unknown provider")
		return
	}

	// Get the code return by OAuth provider
	code := r.URL.Query().Get("code")
	if code == "" {
		server.WriteError(w, http.StatusBadRequest, "Missing authorization code")
		return
	}

	// Exchange code for access token
	token, err := provider.ExchangeToken(code)
	if err != nil {
		server.logger.Error("GET: oauth2/callback: failed to get access token", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Failed to exchange token")
		return
	}

	// Fetch user data from OAuth provider
	user, err := provider.FetchUser(token.AccessToken)
	if err != nil {
		server.logger.Error("GET: oauth2/callback: failed to fetch user data from oauth provider", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Failed to fetch user data")
		return
	}

	// Handle authorization with user credential
	server.handleOAuth(w, r, *user, provider.Name())
}

// handleOAuth handle the OAuth login or register
func (server *Server) handleOAuth(w http.ResponseWriter, r *http.Request, userData userData, provider string) {
	// Check if account is already registered with the email
	isRegistered, err := server.query.IsAccountRegistered(r.Context(), db.IsAccountRegisteredParams{
		OauthProvider:   sql.NullString{String: provider, Valid: true},
		OauthProviderID: sql.NullString{String: userData.ID, Valid: true},
	})
	if err != nil {
		server.logger.Error("GET oauth2/callback: failed to check if account is registered", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internel server error")
		return
	}

	// If account is registered, login the user
	if isRegistered {
		account, err := server.query.LoginWithOAuth(r.Context(), db.LoginWithOAuthParams{
			OauthProvider:   sql.NullString{String: provider, Valid: true},
			OauthProviderID: sql.NullString{String: userData.ID, Valid: true},
		})
		if err != nil {
			server.logger.Error("GET oauth2/callback: failed to login with OAuth", "error", err)
			server.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		// If the account status is not active
		if account.Status != db.AccountStatusActive {
			server.WriteError(w, http.StatusForbidden, "Account is not active")
			return
		}

		// If success, create JWT tokens (access token and refresh token)
		accessToken, err := server.jwtService.CreateToken(account.AccountID.String(), "access-token",
			int(account.TokenVersion), server.jwtService.TokenExpirationTime)
		if err != nil {
			server.logger.Error("GET oauth2/callback: failed to create JWT access token", "error", err)
			server.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		refreshToken, err := server.jwtService.CreateToken(account.AccountID.String(), "refresh-token",
			int(account.TokenVersion), server.jwtService.RefreshTokenExpirationTime)
		if err != nil {
			server.logger.Error("GET oauth2/callback: failed to create JWT refresh token", "error", err)
			server.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Return user info and tokens
		var resp = loginResponse{
			ID:           account.AccountID.String(),
			Email:        account.Email,
			Username:     account.Username,
			Avatar:       service.GenerateMediaLink(account.AccountID.String(), "avatar", "avatar.png"),
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}
		server.WriteJSON(w, http.StatusOK, resp)
		return
	}

	/*
	 * If the account is not registered:
	 * 1. Create a new account with the user data from OAuth provider
	 * 2. Create JWT tokens (access token and refresh token)
	 * 3. Return user info and tokens
	 * Note: as for the avatar:
	 * 1. We will run the downloading as a background task, so the user can use the app immediately
	 * 2. Downloading will have retry mechanism, if failed after 3 times, we will just use the default avatar
	 * 3. Since the avatar is located under the a folder named with the user ID, so there will be no conflict even if we
	 * user avatar.png as the value (hence the database don't need to store the full path to the avatar image). Same logic
	 * apply to the cover image
	 */

	// If account is not registered, create a new account
	account, err := server.query.CreateAccountWithOAuth(r.Context(), db.CreateAccountWithOAuthParams{
		Email:           userData.Email,
		Username:        userData.Username,
		OauthProvider:   sql.NullString{String: provider, Valid: true},
		OauthProviderID: sql.NullString{String: userData.ID, Valid: true},
	})
	if err != nil {
		// If the email is already taken
		if strings.Contains(err.Error(), "accounts_email_key") {
			server.WriteError(w, http.StatusBadRequest, "Email is already taken")
			return
		}

		// If the username is already taken
		if strings.Contains(err.Error(), "accounts_username_key") {
			server.WriteError(w, http.StatusBadRequest, "Username is already taken")
			return
		}

		// Other database error
		server.logger.Error("GET oauth2/callback: failed to create account with OAuth", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Failed to create account")
		return
	}

	// If success, create JWT tokens (access token and refresh token)
	accessToken, err := server.jwtService.CreateToken(account.AccountID.String(), "access-token",
		int(account.TokenVersion), server.jwtService.TokenExpirationTime)
	if err != nil {
		server.logger.Error("GET oauth2/callback: failed to create JWT access token", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	refreshToken, err := server.jwtService.CreateToken(account.AccountID.String(), "refresh-token",
		int(account.TokenVersion), server.jwtService.RefreshTokenExpirationTime)
	if err != nil {
		server.logger.Error("GET oauth2/callback: failed to create JWT refresh token", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Create user repositoty with default avatar and cover
	err = server.storage.CreateUserRepo(account.AccountID.String())
	if err != nil {
		server.logger.Error("POST /oauth2/callback: failed to create user repo", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Download the image and rewrite the default avatar
	server.logger.Info("Image path: ", filepath.Join(account.AccountID.String(), "avatar.png"), "")
	server.storage.DownloadURL(userData.Avatar, filepath.Join(account.AccountID.String(), "avatar.png"))

	// Return user info and tokens
	var resp = loginResponse{
		ID:           account.AccountID.String(),
		Email:        account.Email,
		Username:     account.Username,
		Avatar:       service.GenerateMediaLink(account.AccountID.String(), "avatar", "avatar.png"),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	server.WriteJSON(w, http.StatusOK, resp)
}

/*=== Auth shared logic ===*/

// HandleLogout handles the logout by invalidating the current tokens version
func (server *Server) HandleLogout(w http.ResponseWriter, r *http.Request) {
	/*
	 * POST auth/logout
	 * Success: 200 OK
	 * Error: 400 Bad Request, 500 Internal Server Error
	 */

	// Get the claims from the context
	claims := r.Context().Value(key)

	// Increment the token version to invalidate all existing tokens
	var uuid uuid.UUID
	// The verify already checked if claims is correct CustomClaims type, so we don't need to check again
	if err := uuid.Scan(claims.(*service.CustomClaims).ID); err != nil {
		server.logger.Error("POST /logout: failed to parse account ID", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	err := server.query.IncrementTokenVersion(r.Context(), uuid)
	if err != nil {
		// If no account found with the account ID
		if errors.Is(err, sql.ErrNoRows) {
			server.WriteError(w, http.StatusBadRequest, "Account does not exist")
			return
		}

		// Other database error
		server.logger.Error("POST /logout: failed to increment token version", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	server.WriteJSON(w, http.StatusOK, "Logged out successfully")
}

func (server *Server) HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	/*
	 * POST auth/token/refresh
	 * Success: 200 OK
	 * Error: 400 Bad Request, 500 Internal Server Error
	 *
	 * Although this request did need authentication, but we won't use the AuthMiddleware since we need
	 * the raw refresh token, not just the claims
	 */

	// Get the claims from the context
	claims := r.Context().Value(key)

	// Update token version in database to invalidate all existing tokens
	var uuid uuid.UUID
	if err := uuid.Scan(claims.(*service.CustomClaims).ID); err != nil {
		server.logger.Error("POST /auth/token/refresh: failed to parse account ID", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	err := server.query.IncrementTokenVersion(r.Context(), uuid)
	if err != nil {
		// If no account found with the account ID
		if errors.Is(err, sql.ErrNoRows) {
			server.WriteError(w, http.StatusBadRequest, "Account does not exist")
			return
		}

		// Other database error
		server.logger.Error("POST /auth/token/refresh: failed to increment token version", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Create new access token using the refresh token
	newAccessToken, err := server.jwtService.CreateToken(claims.(*service.CustomClaims).ID, "access-token",
		claims.(*service.CustomClaims).Version+1, server.jwtService.TokenExpirationTime)
	if err != nil {
		server.logger.Error("POST /auth/token/refresh: failed to create new access token", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	server.WriteJSON(w, http.StatusOK, map[string]string{
		"access_token": newAccessToken,
	})
}
