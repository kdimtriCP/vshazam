package storage

import (
	"fmt"
	"io"
	"mime/multipart"
)

type FileInfo struct {
	Filename    string
	ContentType string
	Size        int64
}

type Storage interface {
	SaveFile(file multipart.File, info FileInfo) (string, error)
	OpenFile(path string) (io.ReadSeekCloser, error)
	DeleteFile(path string) error
}

func FormatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
