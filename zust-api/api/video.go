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
	"zust/service"

	"github.com/google/uuid"
)

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

	// Try downloading uploaded video
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

	// Get video duration
	duration, err := service.GetVideoDuration(filename)
	if err != nil {
		server.logger.Error("POST /videos: failed to get video duration", "error", err)
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

	// Update the video metadata in database with duration and status 'published'
	publishedVideo, err := server.query.PublishVideo(r.Context(), db.PublishVideoParams{
		VideoID:  video.VideoID,
		Duration: duration,
	})

	if err != nil {
		server.logger.Error("POST /videos: failed to published video", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Return the result back to client
	server.WriteJSON(w, http.StatusCreated, publishedVideo)

	// Transcode video (background services)
}

type getVideoResponse struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	Media             string    `json:"media"`
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

	// Prepare data for returning back to client
	data := getVideoResponse{
		ID:                video.VideoID.String(),
		Title:             video.Title,
		Media:             service.GenerateMediaLink(video.AccountID.String(), "video", video.VideoID.String()), // We use video ID as filename for now
		Thumbnail:         service.GenerateMediaLink(video.AccountID.String(), "thumbnail", video.VideoID.String()),
		Duration:          int(video.Duration),
		Description:       video.Description.String,
		CreatedAt:         video.CreatedAt,
		PublisherID:       video.AccountID.String(),
		PublisherUsername: video.Username,
		PublisherAvatar:   service.GenerateMediaLink(video.AccountID.String(), "avatar", "avatar.png"),
		TotalSubscriber:   int(video.TotalSubscriber),
		TotakLike:         int(video.TotalLike),
		TotalView:         int(video.TotalView),
	}

	// Send data back to client
	server.WriteJSON(w, http.StatusOK, data)
}
