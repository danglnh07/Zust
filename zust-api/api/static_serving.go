package api

import (
	"net/http"
)

func (server *Server) HandleFile(w http.ResponseWriter, r *http.Request) {
	// Get the ID from path parameter
	id := r.PathValue("id")

	// Get file path
	path := server.storage.ExtractFilePath(id)

	// Serve file
	http.ServeFile(w, r, path)
}
