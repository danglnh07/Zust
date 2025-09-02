package file

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"zust/service/security"
)

// Media service struct, which holds configuration related to media processing
type MediaService struct {
	Domain       string
	Port         string
	ResourcePath string
}

// Constructor method for media service struct
func NewMediaService(config *security.Config) *MediaService {
	return &MediaService{
		Domain:       config.Domain,
		Port:         config.Port,
		ResourcePath: config.ResourcePath,
	}
}

// File type for accssing media resource in user repository
type FileType string

var (
	Avatar    FileType = "avatar"
	Cover     FileType = "cover"
	Video     FileType = "resource"
	Thumbnail FileType = "thumbnail"
)

// Method to generate the URL for accessing media in user repository.
// filename is only the filename, not the full path
func (service *MediaService) GenerateMediaLink(accountID, filename string, fileType FileType) string {
	/*
	 * The media filepath will be encoded with the format:
	 * account_id:file_type:file_name
	 */

	switch fileType {
	case Avatar:
		filename = "avatar.png"
	case Cover:
		filename = "cover.png"
	}

	id := security.Encode(fmt.Sprintf("%s:%s:%s", accountID, fileType, filename))
	return fmt.Sprintf("%s:%s/media/%s", service.Domain, service.Port, id)
}

// Method to extract the full file path from ID generated from the GenerateMediaLink
func (service *MediaService) ExtractFilePath(opaqueID string) string {
	// Split the ID after decoding
	paths := strings.Split(security.Decode(opaqueID), ":")

	// base = resource path + account_id
	base := filepath.Join(service.ResourcePath, paths[0])

	// If this is avatar or cover, we skip the second element of paths, since avatar and cover are not located
	// under sub dirirectory
	if paths[1] == "avatar" || paths[1] == "cover" {
		return filepath.Join(base, paths[2])
	}

	// Otherwise, we use both elements in 'paths' to reconstruct the full file path
	return filepath.Join(base, paths[1], paths[2])
}

// Helper method: get video duration. 'input' expects a full path to where the video located
func (service *MediaService) GetVideoDuration(input string) (int32, error) {
	/*
	 * Command:
	 * ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 input.mp4
	 */

	// Execute command
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", input)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return -1, fmt.Errorf("ffprobe failed for getting video duration: %v\nOutput: %s", err, string(out))
	}

	// Parse data
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 32)
	if err != nil {
		return -1, err
	}

	return int32(duration), nil
}

// Helper method: transcode video into suitable for web progressive streaming.
// Both 'input' and 'output' expect to be a full file path
func TranscodeVideo(input, output string) error {
	/*
	 * Command:
	 * ffmpeg -i input.mp4 -c:v libx264 -preset fast -crf 23 -c:a aac -b:a 128k -movflags +faststart output.mp4
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
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed for transcoding video: %v\nOutput: %s", err, string(out))
	}
	return nil
}

// Video resolution config for transcoding
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

// Helper method: transcode video into suitable web progressive streaming with multiple resolutions.
// 'input' expects a full file path.
// resolutions expects the key to be the ResolutionConfig constants, while the value to be the output full file path
func (service *MediaService) MultiResolution(input string, resolutions map[ResolutionConfig]string) error {
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

	// Create command arguments and add initial value: input and filter_complex
	args := []string{"-i", input, "-filter_complex", filter.String()}

	// Build the rest of the arguments for each resolution
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
		return fmt.Errorf("ffmpeg failed for multi-resolution transcoding: %v\nOutput: %s", err, string(out))
	}
	return nil
}
