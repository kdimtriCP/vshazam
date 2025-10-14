package ai

import (
	"context"
	"time"
)

type VisionService interface {
	AnalyzeFrame(ctx context.Context, imageData []byte) (*FrameAnalysis, error)
}

type FrameAnalysis struct {
	Caption    string          `json:"caption"`
	Labels     []Label         `json:"labels"`
	TextOCR    []string        `json:"text_ocr"`
	Faces      []FaceDetection `json:"faces"`
	Colors     []ColorInfo     `json:"colors"`
	Confidence float64         `json:"confidence"`
	Timestamp  time.Time       `json:"timestamp"`
}

type Label struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
}

type FaceDetection struct {
	BoundingBox BoundingBox `json:"bounding_box"`
	Confidence  float64     `json:"confidence"`
}

type BoundingBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type ColorInfo struct {
	Color      string  `json:"color"`
	Score      float64 `json:"score"`
	PixelRatio float64 `json:"pixel_ratio"`
}

type Config struct {
	OpenAIAPIKey               string
	GoogleVisionKey            string
	GoogleVisionServiceAccount string
	GoogleSearchAPIKey         string
	GoogleCSEID                string
	TMDbAPIKey                 string
	MaxFramesPerVideo          int
	FrameSize                  int
}

func NewConfig() *Config {
	return &Config{
		MaxFramesPerVideo: 5,
		FrameSize:         512,
	}
}
