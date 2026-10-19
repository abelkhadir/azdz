package imageupload

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
)

const maxImageBytes = 5 << 20

var ErrInvalidImage = errors.New("invalid image")

// Save validates the image contents and stores it under a random name.
func Save(file io.Reader, directory, publicPrefix string) (string, string, error) {
	data, err := io.ReadAll(io.LimitReader(file, maxImageBytes+1))
	if err != nil {
		return "", "", err
	}
	if len(data) == 0 || len(data) > maxImageBytes {
		return "", "", fmt.Errorf("%w: image must be between 1 byte and 5 MB", ErrInvalidImage)
	}

	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", "", fmt.Errorf("%w: only valid JPEG, PNG and GIF images are allowed", ErrInvalidImage)
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > 10000 || config.Height > 10000 {
		return "", "", fmt.Errorf("%w: image dimensions are not allowed", ErrInvalidImage)
	}
	extensions := map[string]string{"jpeg": ".jpg", "png": ".png", "gif": ".gif"}
	extension, ok := extensions[format]
	if !ok {
		return "", "", fmt.Errorf("%w: unsupported image format", ErrInvalidImage)
	}

	randomName := make([]byte, 16)
	if _, err := rand.Read(randomName); err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(directory, 0755); err != nil {
		return "", "", err
	}
	filename := hex.EncodeToString(randomName) + extension
	diskPath := filepath.Join(directory, filename)
	if err := os.WriteFile(diskPath, data, 0644); err != nil {
		return "", "", err
	}
	return diskPath, publicPrefix + "/" + filename, nil
}
