package api

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	db "zust/db/sqlc"
	"zust/service/file"

	"github.com/google/uuid"
)

// HandleCreateVideo handle the video uploading.
// endpoint: POST /videos
// Success: 201
// Fail: 400, 403
func (server *Server) HandleCreateVideo(w http.ResponseWriter, r *http.Request) {
	// Check if requester account status is active or not
	var accountID uuid.UUID
	accountID.Scan(r.Context().Value(clKey))
	if _, isActive := server.checkAccountStatus(w, r, accountID); !isActive {
		return
	}

	// Get video metadata and insert into database with status 'pending'
	if err := r.ParseMultipartForm(server.config.VideoSize); err != nil {
		server.WriteError(w, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		server.WriteError(w, http.StatusBadRequest, "Title cannot be empty")
		return
	}

	desc := strings.TrimSpace(r.FormValue("description"))
	var description sql.NullString
	description.Scan(desc)

	publisherID := r.FormValue("publisher_id")
	if publisherID != accountID.String() {
		server.WriteError(w, http.StatusBadRequest, "Publisher ID must be the ID of the requester")
		return
	}

	video, err := server.query.CreateVideo(r.Context(), db.CreateVideoParams{
		Title:       title,
		Description: description,
		PublisherID: accountID,
	})

	if err != nil {
		server.logger.Error("POST /videos: failed to create video", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Try downloading the uploaded video
	resource, _, err := r.FormFile("resource")
	if err != nil || resource == nil {
		server.WriteError(w, http.StatusBadRequest, "Failed to read uploaded video")
		return
	}
	defer resource.Close()

	base := filepath.Join(server.config.ResourcePath, accountID.String())
	filename := filepath.Join(base, "resource", fmt.Sprintf("%s.mp4", video.VideoID.String()))
	dest, err := os.Create(filename)
	if err != nil {
		server.logger.Error("POST /videos: failed to create resource video file in local storage", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer dest.Close()

	_, err = io.Copy(dest, resource)
	if err != nil {
		server.logger.Error("POST /videos: failed to copy the user uploaded video to local storage", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Get video duration and update to database
	duration, err := server.mediaService.GetVideoDuration(filename)
	if err != nil {
		server.logger.Error("POST /videos: failed to get video duration", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	err = server.query.UpdateVideoDuration(r.Context(), db.UpdateVideoDurationParams{
		VideoID:  video.VideoID,
		Duration: duration,
	})
	if err != nil {
		server.logger.Error("POST /videos: failed to update video duration to database", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Get and download thumbnail
	thumbnail, _, err := r.FormFile("thumbnail")
	if err != nil || thumbnail == nil {
		server.WriteError(w, http.StatusBadRequest, "Failed to read uploaded video")
		return
	}

	filename = filepath.Join(base, "thumbnail", fmt.Sprintf("%s.png", video.VideoID.String()))
	dest, err = os.Create(filename)
	if err != nil {
		server.logger.Error("POST /videos: failed to create thumbnail file in local storage", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	_, err = io.Copy(dest, thumbnail)
	if err != nil {
		server.logger.Error("POST /videos: failed to copy the user uploaded thumbnail to local storage", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Return the result back to client
	server.WriteJSON(w, http.StatusCreated, "Video uploaded successfully! The video may not available right away")

	// Transcode video (background services)
}

// request body for GetVideo
type getVideoResponse struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	Resource          string    `json:"resource"`
	Thumbnail         string    `json:"thumbnail"`
	Duration          int       `json:"duration"`
	Description       string    `json:"description"`
	CreatedAt         time.Time `json:"created_at"`
	PublisherID       string    `json:"publisher_id"`
	PublisherUsername string    `json:"username"`
	PublisherAvatar   string    `json:"avatar"`
	TotalSubscriber   int       `json:"total_subscribers"`
	TotakLike         int       `json:"total_like"`
	TotalView         int       `json:"total_view"`
}

// HandleGetVideo handles the GET request for video.
// endpoint: GET /videos/{id}?resolution=...
// Success: 200
// Fail: 400, 403, 404, 500
func (server *Server) HandleGetVideo(w http.ResponseWriter, r *http.Request) {
	// Get video ID
	id := r.PathValue("id")

	// Convert ID (string) to UUID
	var videoUuid uuid.UUID
	if err := videoUuid.Scan(id); err != nil {
		server.WriteError(w, http.StatusBadRequest, "Invalid video ID")
		return
	}

	// Get video
	video, err := server.query.GetVideo(r.Context(), videoUuid)
	if err != nil {
		// If video ID didn't match any record
		if errors.Is(err, sql.ErrNoRows) {
			server.WriteError(w, http.StatusNotFound, "Cannot found any video with this ID")
			return
		}

		// Other database error
		server.logger.Error("GET /videos/{id}: failed to get video", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Check video status
	switch video.Status {
	case db.VideoStatusDeleted:
		server.WriteError(w, http.StatusForbidden, "Video is deleted")
	case db.VideoStatusPending:
		server.WriteError(w, http.StatusBadRequest, "Video is not available for now")
	}

	// Get video based on request parameter
	resourceName := video.VideoID.String()
	switch r.URL.Query().Get("resolution") {
	case "":
		resourceName += ".mp4"
	case "1080p":
		resourceName += "_1080p.mp4"
	case "720p":
		resourceName += "_720p.mp4"
	case "480p":
		resourceName += "_480p.mp4"
	default:
		server.WriteError(w, http.StatusBadRequest, "Unsupport resolution")
		return
	}

	// Send data back to client
	resource := server.mediaService.GenerateMediaLink(video.AccountID.String(), resourceName, file.Video)
	thumbnail := server.mediaService.GenerateMediaLink(
		video.AccountID.String(), fmt.Sprintf("%s.png", video.VideoID.String()), file.Thumbnail,
	)
	avatar := server.mediaService.GenerateMediaLink(video.AccountID.String(), "avatar.png", file.Avatar)
	data := getVideoResponse{
		ID:                video.VideoID.String(),
		Title:             video.Title,
		Resource:          resource,
		Thumbnail:         thumbnail,
		Duration:          int(video.Duration),
		Description:       video.Description.String,
		CreatedAt:         video.CreatedAt,
		PublisherID:       video.AccountID.String(),
		PublisherUsername: video.Username,
		PublisherAvatar:   avatar,
		TotalSubscriber:   int(video.TotalSubscriber),
		TotakLike:         int(video.TotalLike),
		TotalView:         int(video.TotalView),
	}

	server.WriteJSON(w, http.StatusOK, data)
}
