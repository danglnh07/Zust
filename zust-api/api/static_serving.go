package api

import (
	"net/http"
	"zust/service"
)

func (server *Server) HandleFile(w http.ResponseWriter, r *http.Request) {
	// Get the ID from path parameter
	id := r.PathValue("id")

	// Get file path
	path := service.ExtractFilePath(id)

	// Serve file
	http.ServeFile(w, r, path)
}
