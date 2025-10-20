package frame_analysis

import (
	"encoding/json"
	"time"
)

type FrameAnalysisDB struct {
	ID           string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	VideoID      string          `gorm:"type:uuid;not null;index;uniqueIndex:idx_video_frame" json:"video_id"`
	FrameNumber  int             `gorm:"not null;uniqueIndex:idx_video_frame" json:"frame_number"`
	GPTCaption   string          `gorm:"type:text" json:"gpt_caption"`
	VisionLabels json.RawMessage `gorm:"type:jsonb" json:"vision_labels"`
	OCRText      []string        `gorm:"type:jsonb;serializer:json" json:"ocr_text"`
	FaceCount    int             `gorm:"default:0" json:"face_count"`
	AnalysisTime time.Time       `gorm:"not null;index" json:"analysis_time"`
	RawResponse  json.RawMessage `gorm:"type:jsonb" json:"raw_response"`
}

func (FrameAnalysisDB) TableName() string {
	return "frame_analyses"
}
