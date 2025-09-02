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

func (storage *LocalStorage) GenerateMediaLink(accID, fileType, filename, domain, port string) (string, error) {
	/*
	 * The media filepath will be encoded with the format:
	 * account_id:file_type:file_name. For example:
	 */

	if fileType != "avatar" && fileType != "cover" && fileType != "resource" && fileType != "thumbnail" {
		return "", fmt.Errorf("invalid file type")
	}

	switch fileType {
	case "avatar":
		filename = "avatar.png"
	case "cover":
		filename = "cover.png"
	}

	id := util.Encode(fmt.Sprintf("%s:%s:%s", accID, fileType, filename))
	return fmt.Sprintf("%s:%s/media/%s", domain, port, id), nil
}

func (storage *LocalStorage) ExtractFilePath(opaqueID string) string {
	paths := strings.Split(util.Decode(opaqueID), ":")
	base := filepath.Join(storage.BasePath, paths[0]) // Resource path + accID
	if paths[1] == "avatar" || paths[1] == "cover" {
		return filepath.Join(base, paths[2])
	}
	return filepath.Join(base, paths[1], paths[2])
}

func (storage *LocalStorage) GetVideoDuration(filename string) (int32, error) {
	/*
	 * Command:
	 * ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 input.mp4
	 */

	// Execute command
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", filename)
	out, err := cmd.CombinedOutput()
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

func TranscodeVideo(input, output string) error {
	/*
	 * Progressive streaming
	 * Command:
	 * ffmpeg -i filename_raw.mp4 -c:v libx264 -preset fast -crf 23 -c:a aac -b:a 128k -movflags +faststart filename.mp4
	 */

	// Execute the command
	cmd := exec.Command(
		"ffmpeg",
		"-i", input,
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		output,
	)
	_, err := cmd.CombinedOutput()
	return err
}

type ResolutionConfig struct {
	Resolution   string
	CRF          string
	AudiobitRate string
}

var (
	Resolution1080p = ResolutionConfig{
		Resolution:   "1920:1080",
		CRF:          "23",
		AudiobitRate: "128k",
	}

	Resolution720p = ResolutionConfig{
		Resolution:   "1280:720",
		CRF:          "26",
		AudiobitRate: "128k",
	}

	Resolution480p = ResolutionConfig{
		Resolution:   "854:480",
		CRF:          "28",
		AudiobitRate: "96k",
	}
)

func (storage *LocalStorage) MultiResolution(input string, resolutions map[ResolutionConfig]string) error {
	/*
	 * Multi-resolution with progressive streaming
	 * Command:
	 * ffmpeg -i filename.mp4
	 * -filter_complex "[0:v]split=3[v1][v2][v3]; [v1]scale=854:480[v1out]; [v2]scale=1280:720[v2out]; [v3]scale=1920:1080[v3out]"
	 * -map "[v1out]" -map 0:a -c:v libx264 -preset fast -crf 28 -c:a aac -b:a 96k -movflags +faststart filename_480p.mp4
	 * -map "[v2out]" -map 0:a -c:v libx264 -preset fast -crf 26 -c:a aac -b:a 128k -movflags +faststart filename_720p.mp4
	 * -map "[v3out]" -map 0:a -c:v libx264 -preset fast -crf 23 -c:a aac -b:a 128k -movflags +faststart filename_1080p.mp4
	 */

	// Build the filter complex argument
	var (
		filter strings.Builder
		i      = 1
	)

	filter.WriteString(fmt.Sprintf("\"[0:v]split=%d", len(resolutions)))

	for i < len(resolutions) {
		filter.WriteString(fmt.Sprintf("[v%d]", i))
		i++
	}
	i = 1

	for res := range resolutions {
		filter.WriteString(fmt.Sprintf("; [v%d]scale=%s[v%dout]", i, res.Resolution, i))
		i++
	}
	i = 1
	filter.WriteString("\"")

	args := []string{"-i", input, "-filter_complex", filter.String()}

	// Build the arguments for each resolution
	for res, output := range resolutions {
		args = append(args,
			"-map", fmt.Sprintf("\"[v%dout]\"", i),
			"-map", "0;a",
			"-c:v", "libx264",
			"-preset", "fast",
			"-crf", res.CRF,
			"-c:a", "aac",
			"-b:a", res.AudiobitRate,
			"-movflags", "+faststart",
			output,
		)
	}

	// Create command and execute it
	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %v\nOutput: %s", err, string(out))
	}
	return nil
}
