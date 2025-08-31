package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	db "zust/db/sqlc"
	"zust/service"
	"zust/util"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Custom type to avoid context key collisions
type claimsKey string
type endpointKey string

var (
	clKey claimsKey   = "claims"
	epKey endpointKey = "endpoint"
)

// Server struct
type Server struct {
	query       *db.Queries
	jwtService  *service.JWTService
	mailService *service.EmailService
	storage     *service.LocalStorage
	mux         *http.ServeMux
	logger      *slog.Logger
	validate    *validator.Validate
	config      *util.Config
}

// NewServer creates a new HTTP server and setup routing
func NewServer(conn *sql.DB, logger *slog.Logger) *Server {
	config := util.GetConfig()

	server := &Server{
		query:       db.New(conn),
		jwtService:  service.NewJWTService(),
		mailService: service.NewEmailService(),
		storage:     service.NewLocalStorage(),
		mux:         http.NewServeMux(),
		logger:      logger,
		validate:    validator.New(validator.WithRequiredStructEnabled()),
		config:      &config,
	}

	server.RegisterHandler()

	return server
}

// RegisterHandler register all route
func (server *Server) RegisterHandler() {
	// Media serving
	server.mux.HandleFunc("GET /media/{id}", server.HandleFile)

	// Auth routes
	server.mux.HandleFunc("POST /auth/login", server.HandleLogin)
	server.mux.HandleFunc("POST /auth/register", server.HandleRegister)
	server.mux.HandleFunc("POST /auth/verification/resend", server.HandleResendVerification)
	server.mux.HandleFunc("GET /auth/verification", server.HandleVerify)
	server.mux.HandleFunc("GET /oauth2/callback", server.HandleCallback)
	server.mux.Handle("POST /auth/token/refresh", server.AuthMiddleware(http.HandlerFunc(server.HandleRefreshToken)))
	server.mux.Handle("POST /auth/logout", server.AuthMiddleware(http.HandlerFunc(server.HandleLogout)))

	// Account routes
	server.mux.HandleFunc("GET /accounts/{id}", server.HandleGetProfile)
	server.mux.Handle("PUT /accounts/{id}", server.AuthMiddleware(http.HandlerFunc(server.HandleEditProfile)))
	server.mux.Handle("POST /accounts/{id}/lock", server.AuthMiddleware(http.HandlerFunc(server.HandleLockAccount)))
	server.mux.Handle("POST /accounts/{id}/unlock", server.AuthMiddleware(http.HandlerFunc(server.HandleUnlockAccount)))
	server.mux.Handle("POST /subscribe", server.AuthMiddleware(http.HandlerFunc(server.HandleSubscribe)))
	server.mux.Handle("DELETE /subscribe", server.AuthMiddleware(http.HandlerFunc(server.HandleUnsubscribe)))

	// Video routes
	server.mux.Handle("POST /videos/", server.AuthMiddleware(http.HandlerFunc(server.HandleCreateVideo)))
	server.mux.HandleFunc("GET /videos/{id}", server.HandleGetVideo)

}

// Start runs the HTTP server on a specific address
func (server *Server) Start() error {
	config := util.GetConfig()
	server.logger.Info(fmt.Sprintf("Server start at %s:%s", config.Domain, config.Port))
	return http.ListenAndServe(fmt.Sprintf(":%s", config.Port), server.mux)
}

// WriteError writes an error response in JSON format
func (server *Server) WriteError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}

// WriteJSON writes a JSON response with the given status code and data in any data type
func (server *Server) WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"data": data,
	})
}

// Method to check if the request account status is active or not before processing request
func (server *Server) checkAccountStatus(w http.ResponseWriter, r *http.Request, accountID uuid.UUID) (*db.GetProfileRow, bool) {
	// Get old profile from database
	oldProfile, err := server.query.GetProfile(r.Context(), accountID)
	if err != nil {
		// Here, we assume that account ID should exist in DB (by checking if the data passed to this method equal
		// to account ID extract from access token, and since access token already assure that ID exist by verifying
		// the token -> accountID should match)
		server.logger.Error(fmt.Sprintf("%s: failed to get account status for checking", r.Context().Value(epKey)),
			"error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return nil, false
	}

	// Check if account status is active before processing request
	if oldProfile.Status != db.AccountStatusActive {
		return nil, false
	}

	return &oldProfile, true
}

// Method to check if the account ID provided in the request data match with the ID extract from the access token
func (server *Server) checkIDMatch(w http.ResponseWriter, r *http.Request, accountID string) bool {
	// Get the account ID from the claims and check if they match with the account ID given in request data
	claims := r.Context().Value(clKey)
	if claims.(*service.CustomClaims).ID != accountID {
		server.WriteError(w, http.StatusBadRequest, "Account ID not match with the ID from access token")
		return false
	}

	return true
}
