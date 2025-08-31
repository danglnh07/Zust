package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"
	"zust/service"

	"github.com/google/uuid"
)

func (server *Server) HandleCreateVideo(w http.ResponseWriter, r *http.Request) {

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
