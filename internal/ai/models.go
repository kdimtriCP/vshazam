package ai

import (
	"context"
	"fmt"
	"log"
	"time"
)

type OpenAIClientInterface interface {
	GetFrameCaption(ctx context.Context, imageData []byte) (string, error)
}

type GoogleVisionClientInterface interface {
	AnalyzeImage(ctx context.Context, imageData []byte) (*VisionFeatures, error)
}

type VisionServiceImpl struct {
	openAIClient OpenAIClientInterface
	googleClient GoogleVisionClientInterface
	config       *Config
}

func NewVisionService(config *Config) (*VisionServiceImpl, error) {
	if config.OpenAIAPIKey == "" && config.GoogleVisionKey == "" && config.GoogleVisionServiceAccount == "" {
		return nil, fmt.Errorf("at least one API key is required (OpenAI, Google Vision API key, or Google Service Account)")
	}

	service := &VisionServiceImpl{
		config: config,
	}

	if config.OpenAIAPIKey != "" {
		service.openAIClient = NewOpenAIClient(config.OpenAIAPIKey)
		log.Printf("OpenAI Vision service enabled")
	} else {
		log.Printf("OpenAI Vision service disabled (no API key)")
	}

	if config.GoogleVisionServiceAccount != "" {
		client, err := NewGoogleVisionClientWithServiceAccount(config.GoogleVisionServiceAccount)
		if err != nil {
			return nil, fmt.Errorf("failed to create Google Vision client with service account: %w", err)
		}
		service.googleClient = client
		log.Printf("Google Vision service enabled (service account: %s)", config.GoogleVisionServiceAccount)
	} else if config.GoogleVisionKey != "" {
		service.googleClient = NewGoogleVisionClient(config.GoogleVisionKey)
		log.Printf("Google Vision service enabled (API key)")
	} else {
		log.Printf("Google Vision service disabled (no API key or service account)")
	}

	return service, nil
}

func (s *VisionServiceImpl) AnalyzeFrame(ctx context.Context, imageData []byte) (*FrameAnalysis, error) {
	analysis := &FrameAnalysis{
		Timestamp: time.Now(),
	}

	// Only run OpenAI if client is available
	if s.openAIClient != nil {
		captionCh := make(chan struct {
			caption string
			err     error
		}, 1)

		go func() {
			caption, err := s.openAIClient.GetFrameCaption(ctx, imageData)
			captionCh <- struct {
				caption string
				err     error
			}{caption, err}
		}()

		captionResult := <-captionCh
		if captionResult.err != nil {
			log.Printf("Error getting caption from OpenAI: %v", captionResult.err)
		} else {
			analysis.Caption = captionResult.caption
		}
	}

	// Only run Google Vision if client is available
	if s.googleClient != nil {
		visionCh := make(chan struct {
			features *VisionFeatures
			err      error
		}, 1)

		go func() {
			features, err := s.googleClient.AnalyzeImage(ctx, imageData)
			visionCh <- struct {
				features *VisionFeatures
				err      error
			}{features, err}
		}()

		visionResult := <-visionCh
		if visionResult.err != nil {
			log.Printf("Error analyzing image with Google Vision: %v", visionResult.err)
		} else if visionResult.features != nil {
			analysis.Labels = visionResult.features.Labels
			analysis.TextOCR = visionResult.features.Texts
			analysis.Faces = visionResult.features.Faces
			analysis.Colors = visionResult.features.Colors
		}
	}

	analysis.Confidence = s.calculateConfidence(analysis)

	// Return error if no analysis was performed
	if s.openAIClient == nil && s.googleClient == nil {
		return nil, fmt.Errorf("no AI services available")
	}

	return analysis, nil
}

func (s *VisionServiceImpl) calculateConfidence(analysis *FrameAnalysis) float64 {
	confidence := 0.0
	components := 0

	if analysis.Caption != "" {
		confidence += 0.4
		components++
	}

	if len(analysis.Labels) > 0 {
		labelConfidence := 0.0
		for _, label := range analysis.Labels {
			if label.Confidence > labelConfidence {
				labelConfidence = label.Confidence
			}
		}
		confidence += labelConfidence * 0.3
		components++
	}

	if len(analysis.TextOCR) > 0 {
		confidence += 0.2
		components++
	}

	if len(analysis.Faces) > 0 {
		confidence += 0.1
		components++
	}

	if components == 0 {
		return 0.0
	}

	return confidence
}
