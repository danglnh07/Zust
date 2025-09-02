package api

import (
	"net/http"
)

// HandleMedia handle static serving media file
// endpoint: GET /media/{id}
// Fail: 404
func (server *Server) HandleMedia(w http.ResponseWriter, r *http.Request) {
	// Get the ID from path parameter
	id := r.PathValue("id")

	// Get file path
	path := server.mediaService.ExtractFilePath(id)

	// Serve file
	http.ServeFile(w, r, path)
}
