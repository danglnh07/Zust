package service

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"zust/util"
)

type LocalStorage struct {
	BasePath string
}

func NewLocalStorage() *LocalStorage {
	config := util.GetConfig()
	return &LocalStorage{BasePath: config.ResourcePath}
}

func (storage *LocalStorage) DownloadURL(url, filename string) error {
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
	file, err := os.Create(filepath.Join(storage.BasePath, filename))
	if err != nil {
		return err
	}
	defer file.Close()

	// Write response body to file
	_, err = io.Copy(file, resp.Body)
	return err
}
