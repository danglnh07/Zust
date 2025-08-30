package service

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

func (storage *LocalStorage) CreateUserRepo(accID string) error {
	/*
	 * Directory structure:
	 * /storage/{user_id}
	 * /storage/{user_id}/thumbnail
	 * /storage/{user_id}/resource
	 *
	 * Since each user can only has one avatar and one cover, we can just give them the default names
	 * avatar.png and cover.png
	 * While videos and thumbnails will be saved with their video UUID (in database) as names
	 * Doing this will make it easier to manage files, though that means update mean overwrite
	 * the old file -> cannot retrieve
	 */

	// Create user repository directory with their ID as name
	userDir := filepath.Join(storage.BasePath, accID)

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

	_, err = io.Copy(destAvatar, srcAvatar) // <-- Fix: dest first, src second
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

	_, err = io.Copy(destCover, srcCover) // <-- Fix: dest first, src second
	if err != nil {
		return err
	}

	return nil
}

func GenerateMediaLink(accID, fileType, filename string) string {
	/*
	 * The media filepath will be encoded with the format:
	 * account_id:file_type:file_name. For example:
	 * acc0001:avatar:avatar.png
	 * acc0001:cover:cover.png
	 * acc0001:thumbnail:{videoID}.png
	 * acc0001:resource:{videoID_suffix}.png
	 */

	switch fileType {
	case "avatar":
		filename = "avatar.png"
	case "cover":
		filename = "cover.png"
	}

	id := util.Encode(fmt.Sprintf("%s:%s:%s", accID, fileType, filename))
	return fmt.Sprintf("%s:%s/media/%s", util.GetConfig().Domain, util.GetConfig().Port, id)
}

func ExtractFilePath(opaqueID string) string {
	paths := strings.Split(util.Decode(opaqueID), ":")
	base := filepath.Join(util.GetConfig().ResourcePath, paths[0]) // Resource path + accID
	if paths[1] == "avatar" || paths[1] == "cover" {
		return filepath.Join(base, paths[2])
	}

	return filepath.Join(base, paths[1], paths[2])
}

func GetVideoDuration(filename string) (int32, error) {
	/*
	 * Command:
	 * ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1
	 */

	// Execute command
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", filename)
	out, err := cmd.Output()
	if err != nil {
		return -1, err
	}

	// Parse data
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 32)
	if err != nil {
		return -1, err
	}

	return int32(duration), nil
}

func TranscodeVideo(filename string) error {
	/*
	 * Command:
	 * 1. Progressive streaming
	 * ffmpeg -i filename -c:v libx264 -preset fast -crf 23 -c:a aac -b:a 128k -movflags +faststart filename_temp1.mp4
	 * 2. Multi resolution
	 * ffmpeg -i input.mp4 -c:v libx264 -preset fast -crf 23 -c:a aac -b:a 128k -movflags +faststart -vf scale=-2:1080 output_1080p.mp4
	 * ffmpeg -i input.mp4 -c:v libx264 -preset fast -crf 23 -c:a aac -b:a 128k -movflags +faststart -vf scale=-2:720 output_720p.mp4
	 * ffmpeg -i input.mp4 -c:v libx264 -preset fast -crf 23 -c:a aac -b:a 128k -movflags +faststart -vf scale=-2:480 output_480p.mp4
	 * 3. Adaptive streaming (HLS)
	 * ffmpeg -i input.mp4 -map 0:v:0 -map 0:a:0 -c:v libx264 -b:v 3000k -s 1920x1080 -c:a aac -f hls -hls_time 6 -hls_playlist_type vod 1080p.m3u8
	 * ffmpeg -i input.mp4 -map 0:v:0 -map 0:a:0 -c:v libx264 -b:v 3000k -s 1280x720 -c:a aac -f hls -hls_time 6 -hls_playlist_type vod 720p.m3u8
	 * ffmpeg -i input.mp4 -map 0:v:0 -map 0:a:0 -c:v libx264 -b:v 3000k -s 854x480 -c:a aac -f hls -hls_time 6 -hls_playlist_type vod 480p.m3u8
	 */
	return nil
}
