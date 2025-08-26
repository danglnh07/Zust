package api

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"zust/service"
	"zust/util"

	"github.com/go-playground/validator/v10"
)

type Server struct {
	store      *sql.DB
	jwtService *service.JWTService
	mux        *http.ServeMux
	logger     *slog.Logger
	validate   *validator.Validate
}

func NewServer(store *sql.DB, logger *slog.Logger) *Server {
	server := &Server{
		store:      store,
		jwtService: service.NewJWTService(),
		mux:        http.NewServeMux(),
		logger:     logger,
		validate:   validator.New(validator.WithRequiredStructEnabled()),
	}

	server.RegisterHandler()

	return server
}

func (server *Server) RegisterHandler() {
}

func (server *Server) Start() error {
	config := util.GetConfig()
	server.logger.Info(fmt.Sprintf("Server start at %s:%s", config.Domain, config.Port))
	return http.ListenAndServe(fmt.Sprintf(":%s", config.Port), server.mux)
}
