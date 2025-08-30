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
