package storage

import (
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
