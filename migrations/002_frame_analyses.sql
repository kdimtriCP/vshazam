-- Create frame_analyses table for storing AI analysis results
CREATE TABLE IF NOT EXISTS frame_analyses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    frame_number INT NOT NULL,
    gpt_caption TEXT,
    vision_labels JSONB,
    ocr_text JSONB,
    face_count INT DEFAULT 0,
    analysis_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    raw_response JSONB,
    UNIQUE(video_id, frame_number)
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_frame_analyses_video_id ON frame_analyses(video_id);
CREATE INDEX IF NOT EXISTS idx_frame_analyses_analysis_time ON frame_analyses(analysis_time);

-- Create GIN index for JSONB columns for efficient searching
CREATE INDEX IF NOT EXISTS idx_frame_analyses_vision_labels ON frame_analyses USING GIN (vision_labels);
CREATE INDEX IF NOT EXISTS idx_frame_analyses_raw_response ON frame_analyses USING GIN (raw_response);