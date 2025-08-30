package api

import (
	"net/http"
	"zust/service"
)

func (server *Server) HandleFile(w http.ResponseWriter, r *http.Request) {
	// Get the ID from path parameter
	id := r.PathValue("id")
	server.logger.Info(id)

	// Get file path
	path := service.ExtractFilePath(id)
	server.logger.Info(path)

	// Serve file
	http.ServeFile(w, r, path)
}
