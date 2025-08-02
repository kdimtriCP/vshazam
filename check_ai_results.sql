-- SQLite queries to check AI analysis results

-- Check if frame_analyses table exists
SELECT name FROM sqlite_master WHERE type='table' AND name='frame_analyses';

-- Count total analyses
SELECT COUNT(*) as total_analyses FROM frame_analyses;

-- View latest analyses with video info
SELECT 
    v.title,
    v.upload_time,
    fa.frame_number,
    fa.gpt_caption,
    fa.face_count,
    fa.analysis_time
FROM frame_analyses fa
JOIN videos v ON fa.video_id = v.id
ORDER BY fa.analysis_time DESC
LIMIT 10;

-- Check OCR text detected
SELECT 
    v.title,
    fa.frame_number,
    fa.ocr_text
FROM frame_analyses fa
JOIN videos v ON fa.video_id = v.id
WHERE fa.ocr_text != '[]' AND fa.ocr_text IS NOT NULL;

-- Check vision labels
SELECT 
    v.title,
    fa.frame_number,
    fa.vision_labels
FROM frame_analyses fa
JOIN videos v ON fa.video_id = v.id
WHERE fa.vision_labels IS NOT NULL
LIMIT 5;