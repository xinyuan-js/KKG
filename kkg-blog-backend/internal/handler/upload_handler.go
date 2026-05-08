package handler

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"awesomeProject/internal/storage"
	"awesomeProject/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	maxImageSize = 10 << 20 // 10MB
)

type UploadHandler struct {
	storage *storage.MinIOStorage
}

func NewUploadHandler(storage *storage.MinIOStorage) *UploadHandler {
	return &UploadHandler{storage: storage}
}

func (h *UploadHandler) UploadImage(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}
	if fileHeader.Size <= 0 {
		response.BadRequest(c, "empty file")
		return
	}
	if fileHeader.Size > maxImageSize {
		response.BadRequest(c, "image too large, max 10MB")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.ServerError(c, "open file failed")
		return
	}
	defer file.Close()

	contentType, ext, reader, err := detectImage(fileHeader, file)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	url, err := h.storage.UploadImage(c.Request.Context(), reader, fileHeader.Size, contentType, ext)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.OK(c, gin.H{"url": url})
}

func detectImage(fileHeader *multipart.FileHeader, file multipart.File) (string, string, io.Reader, error) {
	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && err != io.EOF {
		return "", "", nil, err
	}
	head = head[:n]
	contentType := strings.ToLower(http.DetectContentType(head))
	if !strings.HasPrefix(contentType, "image/") {
		return "", "", nil, errBad("only image files are allowed")
	}

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileHeader.Filename)), ".")
	if ext == "" {
		ext = extFromContentType(contentType)
	}
	reader := io.MultiReader(bytes.NewReader(head), file)
	return contentType, ext, reader, nil
}

func extFromContentType(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	case "image/svg+xml":
		return "svg"
	case "image/bmp":
		return "bmp"
	default:
		return "jpg"
	}
}

type badRequestErr string

func (e badRequestErr) Error() string {
	return string(e)
}

func errBad(msg string) error {
	return badRequestErr(msg)
}
