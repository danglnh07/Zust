package util

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// Config struct to hold environment variables
type Config struct {
	// Server config
	Domain string
	Port   string

	// Database config
	DbDriver string
	DbSource string

	// OAuth config
	GithubClientID     string
	GithubClientSecret string
	GoogleClientID     string
	GoogleClientSecret string

	// JWT config
	SecretKey                  string
	TokenExpirationTime        time.Duration
	RefreshTokenExpirationTime time.Duration

	// Email config
	SMTPHost    string
	SMTPPort    string
	Email       string
	AppPassword string

	// Resource path
	ResourcePath string

	// File upload constraint
	ImageSize int64
	VideoSize int64
}

var config Config

// Load global variable to hold the configuration
func LoadConfig(path string) error {
	// Load .env file
	err := godotenv.Load(path)
	if err != nil {
		return err
	}

	// Try parse environment variables to its correct type
	tokenExpiration, err := strconv.Atoi(os.Getenv("TOKEN_EXPIRATION"))
	if err != nil {
		return err
	}
	refreshTokenExpiration, err := strconv.Atoi(os.Getenv("REFRESH_TOKEN_EXPIRATION"))
	if err != nil {
		return err
	}

	// Parse image size constraint from string to int
	imageSize, err := strconv.ParseInt(os.Getenv("MAX_IMAGE_SIZE"), 10, 64)
	if err != nil {
		return err
	}
	imageSize <<= 20 // Stored as byte

	// Parse video size constraint from string to int
	videoSize, err := strconv.ParseInt(os.Getenv("MAX_VIDEO_UPLOAD"), 10, 64)
	if err != nil {
		return err
	}
	videoSize <<= 20

	config = Config{
		Domain:                     os.Getenv("DOMAIN"),
		Port:                       os.Getenv("PORT"),
		DbDriver:                   os.Getenv("DB_DRIVER"),
		DbSource:                   os.Getenv("DB_SOURCE"),
		GithubClientID:             os.Getenv("GITHUB_CLIENT_ID"),
		GithubClientSecret:         os.Getenv("GITHUB_CLIENT_SECRET"),
		GoogleClientID:             os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:         os.Getenv("GOOGLE_CLIENT_SECRET"),
		SecretKey:                  os.Getenv("SECRET_KEY"),
		TokenExpirationTime:        time.Duration(tokenExpiration),
		RefreshTokenExpirationTime: time.Duration(refreshTokenExpiration),
		SMTPHost:                   os.Getenv("SMTP_HOST"),
		SMTPPort:                   os.Getenv("SMTP_PORT"),
		Email:                      os.Getenv("EMAIL"),
		AppPassword:                os.Getenv("APP_PASSWORD"),
		ResourcePath:               os.Getenv("RESOURCE_PATH"),
		ImageSize:                  imageSize,
		VideoSize:                  videoSize,
	}
	return err
}

// Method to get the configuration
func GetConfig() Config {
	return config
}

// Method to hash a string using SHA-256
func Hash(str string) string {
	hasher := sha256.New()
	hasher.Write([]byte(str))
	return hex.EncodeToString(hasher.Sum(nil))
}

// Methods to encode a string using Base64 URL encoding
func Encode(str string) string {
	return base64.URLEncoding.EncodeToString([]byte(str))
}

// Method to decode a Base64 URL encoded string
func Decode(str string) string {
	data, err := base64.URLEncoding.DecodeString(str)
	if err != nil {
		return ""
	}
	return string(data)
}

// Methods to hash passwords using bcrypt
func BcryptHash(str string) (string, error) {
	// Use bcrypt to hash the password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// Method to compare a bcrypt hashed password with a plain text password
func BcryptCompare(hashedStr, plainStr string) bool {
	// Compare the hashed password with the plain text password
	err := bcrypt.CompareHashAndPassword([]byte(hashedStr), []byte(plainStr))
	return err == nil
}
