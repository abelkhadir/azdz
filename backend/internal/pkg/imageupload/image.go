package imageupload

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
)

const maxImageBytes = 5 << 20

// Save validates the image contents and stores it under a random name.
func Save(file io.Reader, directory, publicPrefix string) (string, string, error) {
	data, err := io.ReadAll(io.LimitReader(file, maxImageBytes+1))
	if err != nil {
		return "", "", err
	}
	if len(data) == 0 || len(data) > maxImageBytes {
		return "", "", errors.New("image must be between 1 byte and 5 MB")
	}

	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", "", errors.New("invalid image file")
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > 10000 || config.Height > 10000 {
		return "", "", errors.New("image dimensions are not allowed")
	}
	extensions := map[string]string{"jpeg": ".jpg", "png": ".png", "gif": ".gif"}
	extension, ok := extensions[format]
	if !ok {
		return "", "", errors.New("unsupported image format")
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
