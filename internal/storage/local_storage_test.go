package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

type mockFile struct {
	*bytes.Reader
}

func (m *mockFile) Close() error {
	return nil
}

func TestLocalStorage(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	t.Run("SaveFile", func(t *testing.T) {
		content := []byte("test video content")
		reader := &mockFile{bytes.NewReader(content)}

		info := FileInfo{
			Filename:    "test.mp4",
			ContentType: "video/mp4",
			Size:        int64(len(content)),
		}

		filename, err := storage.SaveFile(reader, info)
		if err != nil {
			t.Fatalf("Failed to save file: %v", err)
		}

		if filepath.Ext(filename) != ".mp4" {
			t.Errorf("Expected .mp4 extension, got %s", filepath.Ext(filename))
		}

		savedPath := filepath.Join(tmpDir, filename)
		if _, err := os.Stat(savedPath); os.IsNotExist(err) {
			t.Errorf("File was not saved to expected location: %s", savedPath)
		}
	})

	t.Run("OpenFile", func(t *testing.T) {
		content := []byte("test video content")
		testFile := "test-file.mp4"
		fullPath := filepath.Join(tmpDir, testFile)

		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := storage.OpenFile(testFile)
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}
		defer file.Close()

		buf := make([]byte, len(content))
		n, err := file.Read(buf)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if n != len(content) || !bytes.Equal(buf, content) {
			t.Errorf("File content mismatch")
		}
	})

	t.Run("DeleteFile", func(t *testing.T) {
		testFile := "delete-test.mp4"
		fullPath := filepath.Join(tmpDir, testFile)

		if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		if err := storage.DeleteFile(testFile); err != nil {
			t.Fatalf("Failed to delete file: %v", err)
		}

		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			t.Errorf("File was not deleted")
		}
	})

	t.Run("PathTraversalPrevention", func(t *testing.T) {
		_, err := storage.OpenFile("../../../etc/passwd")
		if err == nil {
			t.Errorf("Path traversal was not prevented")
		}

		err = storage.DeleteFile("../../../etc/passwd")
		if err == nil {
			t.Errorf("Path traversal was not prevented in delete")
		}
	})
}
