package ai

import (
	"context"
	"math"
	"testing"
)

type mockOpenAIClient struct {
	caption string
	err     error
}

func (m *mockOpenAIClient) GetFrameCaption(ctx context.Context, imageData []byte) (string, error) {
	return m.caption, m.err
}

type mockGoogleVisionClient struct {
	features *VisionFeatures
	err      error
}

func (m *mockGoogleVisionClient) AnalyzeImage(ctx context.Context, imageData []byte) (*VisionFeatures, error) {
	return m.features, m.err
}

func TestVisionServiceAnalyzeFrame(t *testing.T) {
	tests := []struct {
		name            string
		mockCaption     string
		mockFeatures    *VisionFeatures
		expectedCaption string
		expectedLabels  int
		expectedOCR     int
	}{
		{
			name:        "successful analysis",
			mockCaption: "A scene from a movie showing actors in a city",
			mockFeatures: &VisionFeatures{
				Labels: []Label{
					{Name: "movie", Confidence: 0.9},
					{Name: "city", Confidence: 0.8},
				},
				Texts: []string{"Movie Title", "2023"},
				Faces: []FaceDetection{{
					BoundingBox: BoundingBox{X: 10, Y: 10, Width: 100, Height: 100},
					Confidence:  0.95,
				}},
				Colors: []ColorInfo{
					{Color: "rgb(255,0,0)", Score: 0.7, PixelRatio: 0.3},
				},
			},
			expectedCaption: "A scene from a movie showing actors in a city",
			expectedLabels:  2,
			expectedOCR:     2,
		},
		{
			name:            "only caption available",
			mockCaption:     "A dramatic scene",
			mockFeatures:    &VisionFeatures{},
			expectedCaption: "A dramatic scene",
			expectedLabels:  0,
			expectedOCR:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &VisionServiceImpl{
				openAIClient: &mockOpenAIClient{caption: tt.mockCaption},
				googleClient: &mockGoogleVisionClient{features: tt.mockFeatures},
				config:       &Config{},
			}

			imageData := []byte("fake image data")
			analysis, err := service.AnalyzeFrame(context.Background(), imageData)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if analysis.Caption != tt.expectedCaption {
				t.Errorf("expected caption %q, got %q", tt.expectedCaption, analysis.Caption)
			}

			if len(analysis.Labels) != tt.expectedLabels {
				t.Errorf("expected %d labels, got %d", tt.expectedLabels, len(analysis.Labels))
			}

			if len(analysis.TextOCR) != tt.expectedOCR {
				t.Errorf("expected %d OCR texts, got %d", tt.expectedOCR, len(analysis.TextOCR))
			}

			if analysis.Confidence <= 0 {
				t.Error("expected positive confidence score")
			}
		})
	}
}

func TestCalculateConfidence(t *testing.T) {
	service := &VisionServiceImpl{config: &Config{}}

	tests := []struct {
		name               string
		analysis           *FrameAnalysis
		expectedConfidence float64
	}{
		{
			name: "full analysis",
			analysis: &FrameAnalysis{
				Caption: "Test caption",
				Labels: []Label{
					{Name: "test", Confidence: 0.8},
				},
				TextOCR: []string{"text"},
				Faces:   []FaceDetection{{Confidence: 0.9}},
			},
			expectedConfidence: 0.94, // 0.4 + (0.8 * 0.3) + 0.2 + 0.1
		},
		{
			name:               "empty analysis",
			analysis:           &FrameAnalysis{},
			expectedConfidence: 0.0,
		},
		{
			name: "caption only",
			analysis: &FrameAnalysis{
				Caption: "Test caption",
			},
			expectedConfidence: 0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := service.calculateConfidence(tt.analysis)
			if math.Abs(confidence-tt.expectedConfidence) > 0.001 {
				t.Errorf("expected confidence %f, got %f", tt.expectedConfidence, confidence)
			}
		})
	}
}