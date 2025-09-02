package file

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"zust/service/security"
)

// Local storage struct, which hold configuration related to local storage
type LocalStorage struct {
	ResourcePath string
}

// Constructor method for local storage struct
func NewLocalStorage(config *security.Config) *LocalStorage {
	return &LocalStorage{
		ResourcePath: config.ResourcePath,
	}
}

// Method to download media from a URL.
// 'path' expect only the full file path of the destination file
func (storage *LocalStorage) DownloadURL(url, path string) error {
	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// Perform the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create file in local storage
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write response body to file
	_, err = io.Copy(file, resp.Body)
	return err
}

// Method to create user repository in local storage with default avatar and cover
func (storage *LocalStorage) CreateUserRepo(accID string) error {
	/*
	 * Directory structure example
	 * storage
	 * |__{account_id}
	 * |____resource
	 * |______{video_id}.mp4
	 * |______{video_id}_1080p.mp4
	 * |______{video_id}_720p.mp4
	 * |______{video_id}_480p.mp4
	 * |____thumbnail
	 * |______{video_id}.png
	 * |____avatar.png
	 * |____cover.png
	 */

	// Create user repository directory with their ID as name
	userDir := filepath.Join(storage.ResourcePath, accID)

	// Create 'thumbnail' and 'resource' subdirectories
	subDirs := []string{"resource", "thumbnail"}
	for _, dir := range subDirs {
		if err := os.MkdirAll(filepath.Join(userDir, dir), 0755); err != nil {
			return err
		}
	}

	// Create default avatar image
	srcAvatar, err := os.Open("asset/avatar.png")
	if err != nil {
		return err
	}
	defer srcAvatar.Close()

	destAvatar, err := os.Create(filepath.Join(userDir, "avatar.png"))
	if err != nil {
		return err
	}
	defer destAvatar.Close()

	_, err = io.Copy(destAvatar, srcAvatar)
	if err != nil {
		return err
	}

	// Create default cover image
	srcCover, err := os.Open("asset/cover.png")
	if err != nil {
		return err
	}
	defer srcCover.Close()

	destCover, err := os.Create(filepath.Join(userDir, "cover.png"))
	if err != nil {
		return err
	}
	defer destCover.Close()

	_, err = io.Copy(destCover, srcCover)
	if err != nil {
		return err
	}

	return nil
}
