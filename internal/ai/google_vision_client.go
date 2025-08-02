package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const googleVisionAPIURL = "https://vision.googleapis.com/v1/images:annotate"

type GoogleVisionClient struct {
	apiKey     string
	httpClient *http.Client
}

func NewGoogleVisionClient(apiKey string) *GoogleVisionClient {
	return &GoogleVisionClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type googleVisionRequest struct {
	Requests []imageRequest `json:"requests"`
}

type imageRequest struct {
	Image    imageContent  `json:"image"`
	Features []featureType `json:"features"`
}

type imageContent struct {
	Content string `json:"content"`
}

type featureType struct {
	Type       string `json:"type"`
	MaxResults int    `json:"maxResults,omitempty"`
}

type googleVisionResponse struct {
	Responses []annotateResponse `json:"responses"`
	Error     *googleError       `json:"error"`
}

type googleError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type annotateResponse struct {
	LabelAnnotations     []labelAnnotation     `json:"labelAnnotations"`
	TextAnnotations      []textAnnotation      `json:"textAnnotations"`
	FaceAnnotations      []faceAnnotation      `json:"faceAnnotations"`
	ImagePropertiesAnnotation *imageProperties `json:"imagePropertiesAnnotation"`
	Error                *googleError          `json:"error"`
}

type labelAnnotation struct {
	Description string  `json:"description"`
	Score       float64 `json:"score"`
	Confidence  float64 `json:"confidence"`
}

type textAnnotation struct {
	Description string `json:"description"`
	Locale      string `json:"locale"`
}

type faceAnnotation struct {
	BoundingPoly     boundingPoly `json:"boundingPoly"`
	DetectionConfidence float64   `json:"detectionConfidence"`
}

type boundingPoly struct {
	Vertices []vertex `json:"vertices"`
}

type vertex struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type imageProperties struct {
	DominantColors dominantColors `json:"dominantColors"`
}

type dominantColors struct {
	Colors []colorInfo `json:"colors"`
}

type colorInfo struct {
	Color         color   `json:"color"`
	Score         float64 `json:"score"`
	PixelFraction float64 `json:"pixelFraction"`
}

type color struct {
	Red   int `json:"red"`
	Green int `json:"green"`
	Blue  int `json:"blue"`
}

type VisionFeatures struct {
	Labels     []Label
	Texts      []string
	Faces      []FaceDetection
	Colors     []ColorInfo
}

func (c *GoogleVisionClient) AnalyzeImage(ctx context.Context, imageData []byte) (*VisionFeatures, error) {
	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	reqBody := googleVisionRequest{
		Requests: []imageRequest{
			{
				Image: imageContent{
					Content: imageBase64,
				},
				Features: []featureType{
					{Type: "LABEL_DETECTION", MaxResults: 10},
					{Type: "TEXT_DETECTION", MaxResults: 10},
					{Type: "FACE_DETECTION", MaxResults: 10},
					{Type: "IMAGE_PROPERTIES", MaxResults: 5},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s?key=%s", googleVisionAPIURL, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var visionResp googleVisionResponse
	if err := json.Unmarshal(body, &visionResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if visionResp.Error != nil {
		return nil, fmt.Errorf("Google Vision API error: %s", visionResp.Error.Message)
	}

	if len(visionResp.Responses) == 0 {
		return nil, fmt.Errorf("no response from Google Vision API")
	}

	response := visionResp.Responses[0]
	if response.Error != nil {
		return nil, fmt.Errorf("Google Vision API error: %s", response.Error.Message)
	}

	features := &VisionFeatures{
		Labels: make([]Label, 0, len(response.LabelAnnotations)),
		Texts:  make([]string, 0, len(response.TextAnnotations)),
		Faces:  make([]FaceDetection, 0, len(response.FaceAnnotations)),
		Colors: make([]ColorInfo, 0),
	}

	for _, label := range response.LabelAnnotations {
		features.Labels = append(features.Labels, Label{
			Name:       label.Description,
			Confidence: label.Score,
		})
	}

	for i, text := range response.TextAnnotations {
		if i == 0 && len(response.TextAnnotations) > 1 {
			continue
		}
		features.Texts = append(features.Texts, text.Description)
	}

	for _, face := range response.FaceAnnotations {
		if len(face.BoundingPoly.Vertices) >= 4 {
			minX, minY := face.BoundingPoly.Vertices[0].X, face.BoundingPoly.Vertices[0].Y
			maxX, maxY := minX, minY
			
			for _, v := range face.BoundingPoly.Vertices {
				if v.X < minX {
					minX = v.X
				}
				if v.X > maxX {
					maxX = v.X
				}
				if v.Y < minY {
					minY = v.Y
				}
				if v.Y > maxY {
					maxY = v.Y
				}
			}
			
			features.Faces = append(features.Faces, FaceDetection{
				BoundingBox: BoundingBox{
					X:      minX,
					Y:      minY,
					Width:  maxX - minX,
					Height: maxY - minY,
				},
				Confidence: face.DetectionConfidence,
			})
		}
	}

	if response.ImagePropertiesAnnotation != nil {
		for _, c := range response.ImagePropertiesAnnotation.DominantColors.Colors {
			features.Colors = append(features.Colors, ColorInfo{
				Color:      fmt.Sprintf("rgb(%d,%d,%d)", c.Color.Red, c.Color.Green, c.Color.Blue),
				Score:      c.Score,
				PixelRatio: c.PixelFraction,
			})
		}
	}

	return features, nil
}