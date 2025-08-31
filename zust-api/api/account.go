package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	db "zust/db/sqlc"

	"github.com/google/uuid"
)

func (server *Server) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	// Get the account ID from path parameter
	id := r.PathValue("id")

	// Convert id (string) to uuid
	var accUuid uuid.UUID
	if err := accUuid.Scan(id); err != nil {
		server.WriteError(w, http.StatusBadRequest, "Invalid account ID")
		return
	}

	// Get account profile from database
	account, err := server.query.GetProfile(r.Context(), accUuid)
	if err != nil {
		// If account ID not match any record
		if errors.Is(err, sql.ErrNoRows) {
			server.WriteError(w, http.StatusNotFound, "Account not found")
			return
		}

		// Other database error
		server.logger.Error("GET /accounts/{id}: failed to get account profile", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Check if account status is active
	if account.Status != db.AccountStatusActive {
		server.WriteError(w, http.StatusForbidden, "Account is not active")
		return
	}

	// Return account profile
	server.WriteJSON(w, http.StatusOK, account)
}

func (server *Server) HandleEditProfile(w http.ResponseWriter, r *http.Request) {
	// Check if the account ID in path parameter match with the ID extract from access token
	if isIDMatched := server.checkIDMatch(w, r, r.PathValue("id")); !isIDMatched {
		return
	}

	// Check account status if it's active or not before processing with the request
	var accID uuid.UUID
	accID.Scan(r.PathValue("id"))
	r = r.WithContext(context.WithValue(r.Context(), epKey, "PUT /accounts/{id}"))
	oldProfile, isActive := server.checkAccountStatus(w, r, accID)
	if !isActive {
		return
	}

	// Parse request multipart form data
	r.Body = http.MaxBytesReader(w, r.Body, int64(server.config.ImageSize))
	base := filepath.Join(server.config.ResourcePath, accID.String())

	// Get new avatar image if provided
	avatar, _, err := r.FormFile("avatar")
	if err != nil {
		server.WriteError(w, http.StatusBadRequest, "Invalid avatar file")
		return
	}
	if avatar != nil {
		defer avatar.Close()
		// Copy new file to storage
		oldAvatar, err := os.OpenFile(filepath.Join(base, "avatar.png"), os.O_RDWR, os.ModePerm)
		if err != nil {
			server.logger.Error("PUT /accounts/{id}: failed to open the current avatar file in storage", "id", accID.String(),
				"error", err)
			server.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		defer oldAvatar.Close()

		_, err = io.Copy(oldAvatar, avatar)
		if err != nil {
			server.logger.Error("PUT /accounts/{id}: failed to overwrite avatar", "error", err)
			server.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}

	// Get new cover image file if provided
	cover, _, err := r.FormFile("cover")
	if err != nil {
		server.WriteError(w, http.StatusBadRequest, "Invalid cover file")
		return
	}
	if cover != nil {
		defer cover.Close()
		// Copy new file to storage
		oldCover, err := os.OpenFile(filepath.Join(base, "cover.png"), os.O_RDWR, os.ModePerm)
		if err != nil {
			server.logger.Error("PUT /accounts/{id}: failed to open the current cover file in storage", "id", accID.String(),
				"error", err)
			server.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		defer oldCover.Close()

		_, err = io.Copy(oldCover, cover)
		if err != nil {
			server.logger.Error("PUT /accounts/{id}: failed to overwrite cover image", "error", err)
			server.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

	}

	// Get username and description (if empty, use the old value from oldProfile)
	username := r.FormValue("username")
	description := r.FormValue("description")

	if username == "" {
		username = oldProfile.Username
	}

	if description == "" {
		description = oldProfile.Description.String
	}

	// Update profile
	account, err := server.query.EditProfile(r.Context(), db.EditProfileParams{
		AccountID:   accID,
		Username:    username,
		Description: sql.NullString{String: description, Valid: true},
	})

	if err != nil {
		server.logger.Error("PUT /accounts/{id}: failed to edit profile", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Return the newly updated profile back to client
	server.WriteJSON(w, http.StatusCreated, account)
}

func (server *Server) HandleLockAccount(w http.ResponseWriter, r *http.Request) {
	// Check if the account ID in path parameter match with the ID extract from access token
	if isIDMatched := server.checkIDMatch(w, r, r.PathValue("id")); !isIDMatched {
		return
	}

	// Check account status if it's active or not before processing with the request
	var accID uuid.UUID
	accID.Scan(r.PathValue("id"))
	r = r.WithContext(context.WithValue(r.Context(), epKey, "POST /accounts/{id}/lock"))
	if _, isActive := server.checkAccountStatus(w, r, accID); isActive {
		return
	}

	// Lock account
	err := server.query.LockAccount(r.Context(), accID)
	if err != nil {
		server.logger.Error("POST /accounts/{id}/lock: failed to lock account", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	server.WriteJSON(w, http.StatusCreated, fmt.Sprintf("Account with ID %s locked successfully", accID.String()))
}

type subscribeRequest struct {
	SubscriberID   uuid.UUID `json:"subscriber_id" validate:"required"`
	SubscriberToID uuid.UUID `json:"subscribe_to_id" validate:"required"`
}

func (server *Server) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	// Get request body
	var req subscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request body
	if err := server.validate.Struct(req); err != nil {
		server.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Check if the account ID (subscriber ID in this case) match with the ID extract from claims
	if isIDMatched := server.checkIDMatch(w, r, req.SubscriberID.String()); !isIDMatched {
		return
	}

	// Check if subscriber status is active or not before processing with the request
	r = r.WithContext(context.WithValue(r.Context(), epKey, "POST /subscribe"))
	if _, isActive := server.checkAccountStatus(w, r, req.SubscriberID); !isActive {
		return
	}

	// Create subscription
	result, err := server.query.Subscribe(r.Context(), db.SubscribeParams{
		SubscriberID:  req.SubscriberID,
		SubscribeToID: req.SubscriberToID,
	})

	if err != nil {
		server.logger.Error("POST /subscribe: failed to create subscription", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Return result back to client
	server.WriteJSON(w, http.StatusCreated, result)
}

func (server *Server) HandleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	// Get request body
	var req subscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request body
	if err := server.validate.Struct(req); err != nil {
		server.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Check if the account ID (subscriber ID in this case) match with the ID extract from claims
	if isIDMatched := server.checkIDMatch(w, r, req.SubscriberID.String()); !isIDMatched {
		return
	}

	// Check if subscriber status is active or not before processing with the request
	r = r.WithContext(context.WithValue(r.Context(), epKey, "DELETE /subscribe"))
	if _, isActive := server.checkAccountStatus(w, r, req.SubscriberID); !isActive {
		return
	}

	// Delete subscription
	err := server.query.Unsubscribe(r.Context(), db.UnsubscribeParams{
		SubscriberID:  req.SubscriberID,
		SubscribeToID: req.SubscriberToID,
	})

	if err != nil {
		server.logger.Error("DELETE /subscribe: failed to create subscription", "error", err)
		server.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Return result back to client
	server.WriteJSON(w, http.StatusOK, "Unsubscription successfully")
}
