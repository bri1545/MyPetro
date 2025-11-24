package storage

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

const (
	MaxFileSize   = 5 * 1024 * 1024
	MaxFilesCount = 3
)

var allowedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
}

func SaveProjectImages(projectID int, files []*multipart.FileHeader) ([]string, error) {
	if len(files) > MaxFilesCount {
		return nil, fmt.Errorf("максимум %d фотографий разрешено", MaxFilesCount)
	}

	uploadDir := fmt.Sprintf("uploads/%d", projectID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, err
	}

	var savedPaths []string

	for i, fileHeader := range files {
		if fileHeader.Size > MaxFileSize {
			return nil, fmt.Errorf("файл %s превышает максимальный размер 5MB", fileHeader.Filename)
		}

		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if !allowedExtensions[ext] {
			return nil, fmt.Errorf("файл %s имеет недопустимый формат. Разрешены только JPG и PNG", fileHeader.Filename)
		}

		file, err := fileHeader.Open()
		if err != nil {
			return nil, err
		}
		defer file.Close()

		filename := fmt.Sprintf("photo_%d%s", i+1, ext)
		filepath := filepath.Join(uploadDir, filename)

		dst, err := os.Create(filepath)
		if err != nil {
			return nil, err
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			return nil, err
		}

		savedPaths = append(savedPaths, "/"+filepath)
	}

	return savedPaths, nil
}
