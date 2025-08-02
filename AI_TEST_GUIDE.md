# AI Integration Testing Guide

## Prerequisites
1. Install FFmpeg: `brew install ffmpeg` (macOS) or `apt-get install ffmpeg` (Linux)
2. Get API keys:
   - OpenAI: https://platform.openai.com/api-keys
   - Google Vision: https://console.cloud.google.com/apis/credentials

## Step 1: Set Up Environment

Create a `.env` file with your actual API keys:
```bash
OPENAI_API_KEY=sk-your-actual-key-here
GOOGLE_VISION_API_KEY=your-google-vision-key
MAX_FRAMES_PER_VIDEO=3
FRAME_SIZE=512
```

## Step 2: Start the Server

```bash
# Load environment and start server
source .env && go run cmd/server/main.go
```

Look for this message:
- ✅ If configured: Server will start normally
- ❌ If not configured: "AI services not configured. Set OPENAI_API_KEY and GOOGLE_VISION_API_KEY to enable."

## Step 3: Upload a Test Video

1. Open http://localhost:8080
2. Click "Upload Video"
3. Upload a short video clip (ideally from a movie)
4. Watch the server logs!

## Step 4: Check the Logs

You should see:
```
Extracted 3 frames from video <uuid>
Successfully analyzed frame 0 for video <uuid> (confidence: 0.XX)
Successfully analyzed frame 1 for video <uuid> (confidence: 0.XX)
Successfully analyzed frame 2 for video <uuid> (confidence: 0.XX)
```

## Step 5: Verify Results in Database

Run the check tool:
```bash
go run cmd/check-ai/main.go
```

Or use SQLite directly:
```bash
sqlite3 vshazam.db < check_ai_results.sql
```

## Step 6: Check for Common Issues

### If no analysis happens:
- Check API keys are set correctly
- Ensure FFmpeg is installed: `ffmpeg -version`
- Check server logs for errors

### If analysis fails:
- Check API key validity
- Ensure video file is valid
- Check internet connection
- Look for specific error messages in logs

## Test Video Suggestions

For best results, test with:
- Movie clips with clear scenes
- Videos with visible text/titles
- Clips with recognizable actors
- Scenes with distinct visual elements

## Expected AI Output

A successful analysis will show:
- **GPT-4 Caption**: Detailed scene description
- **Labels**: Scene elements (e.g., "person", "car", "outdoor")
- **OCR Text**: Any on-screen text detected
- **Face Count**: Number of faces detected
- **Confidence Score**: Combined confidence (0.0-1.0)

## Quick Debug Commands

```bash
# Check if ffmpeg works
ffmpeg -version

# Test frame extraction manually
ffmpeg -i test.mp4 -vf "select=eq(n\,0)" -vframes 1 test_frame.jpg

# Check database
sqlite3 vshazam.db "SELECT COUNT(*) FROM frame_analyses;"

# View raw AI responses
sqlite3 vshazam.db "SELECT raw_response FROM frame_analyses LIMIT 1;" | jq
```