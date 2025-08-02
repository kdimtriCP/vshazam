package ai

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type FrameExtractor struct {
	ffmpegPath string
	tempDir    string
}

func NewFrameExtractor() (*FrameExtractor, error) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}
	log.Printf("Found ffmpeg at: %s", ffmpegPath)

	tempDir := filepath.Join(os.TempDir(), "vshazam-frames")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	log.Printf("Using temp directory: %s", tempDir)

	return &FrameExtractor{
		ffmpegPath: ffmpegPath,
		tempDir:    tempDir,
	}, nil
}

func (fe *FrameExtractor) ExtractFrames(videoPath string, count int, size int) ([][]byte, error) {
	// Log for debugging
	log.Printf("ExtractFrames called with: path=%s, count=%d, size=%d", videoPath, count, size)
	
	// Check if video file exists
	if _, err := os.Stat(videoPath); err != nil {
		return nil, fmt.Errorf("video file not accessible: %w", err)
	}
	
	duration, err := fe.getVideoDuration(videoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get video duration: %w", err)
	}

	log.Printf("Video duration: %.2f seconds", duration)

	if duration <= 0 {
		return nil, fmt.Errorf("invalid video duration: %f", duration)
	}

	frames := make([][]byte, 0, count)
	interval := duration / float64(count+1)

	successCount := 0
	for i := 1; i <= count; i++ {
		timestamp := interval * float64(i)
		log.Printf("Extracting frame %d/%d at timestamp %.2f", i, count, timestamp)
		
		frameData, err := fe.extractSingleFrame(videoPath, timestamp, size)
		if err != nil {
			log.Printf("Failed to extract frame %d: %v", i, err)
			continue
		}
		frames = append(frames, frameData)
		successCount++
		log.Printf("Successfully extracted frame %d (size: %d bytes)", i, len(frameData))
	}

	if len(frames) == 0 {
		return nil, fmt.Errorf("failed to extract any frames from video (attempted %d frames)", count)
	}
	
	log.Printf("Successfully extracted %d/%d frames", successCount, count)

	return frames, nil
}

func (fe *FrameExtractor) getVideoDuration(videoPath string) (float64, error) {
	// Try ffprobe first for more reliable duration detection
	ffprobePath, err := exec.LookPath("ffprobe")
	if err == nil {
		cmd := exec.Command(ffprobePath,
			"-v", "error",
			"-show_entries", "format=duration",
			"-of", "default=noprint_wrappers=1:nokey=1",
			videoPath)

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err == nil {
			durationStr := strings.TrimSpace(stdout.String())
			if duration, err := strconv.ParseFloat(durationStr, 64); err == nil && duration > 0 {
				return duration, nil
			}
		}
	}

	// Fallback to parsing ffmpeg output
	cmd := exec.Command(fe.ffmpegPath,
		"-i", videoPath,
		"-f", "null",
		"-")
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_ = cmd.Run()
	output := stderr.String()
	durationPrefix := "Duration: "
	startIndex := strings.Index(output, durationPrefix)
	if startIndex == -1 {
		return 0, fmt.Errorf("duration not found in ffmpeg output")
	}

	startIndex += len(durationPrefix)
	endIndex := strings.Index(output[startIndex:], ",")
	if endIndex == -1 {
		return 0, fmt.Errorf("invalid duration format")
	}

	durationStr := output[startIndex : startIndex+endIndex]
	parts := strings.Split(durationStr, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid duration format: %s", durationStr)
	}

	hours, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, err
	}
	seconds, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0, err
	}

	return hours*3600 + minutes*60 + seconds, nil
}

func (fe *FrameExtractor) extractSingleFrame(videoPath string, timestamp float64, size int) ([]byte, error) {
	tempFile := filepath.Join(fe.tempDir, fmt.Sprintf("frame_%f.jpg", timestamp))
	defer os.Remove(tempFile)

	// Build ffmpeg command with simpler parameters first
	args := []string{
		"-ss", fmt.Sprintf("%.2f", timestamp),
		"-i", videoPath,
		"-vframes", "1",
		"-vf", fmt.Sprintf("scale='min(%d,iw)':'min(%d,ih)':force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black", size, size, size, size),
		"-q:v", "2",
		"-f", "mjpeg",
		tempFile,
	}
	
	cmd := exec.Command(fe.ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	log.Printf("Running ffmpeg command: %s %v", fe.ffmpegPath, args)

	if err := cmd.Run(); err != nil {
		log.Printf("FFmpeg stderr output: %s", stderr.String())
		return nil, fmt.Errorf("failed to extract frame at %f: %w", timestamp, err)
	}

	file, err := os.Open(tempFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open extracted frame: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode frame: %w", err)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, fmt.Errorf("failed to encode frame: %w", err)
	}

	return buf.Bytes(), nil
}

func (fe *FrameExtractor) Cleanup() error {
	return os.RemoveAll(fe.tempDir)
}