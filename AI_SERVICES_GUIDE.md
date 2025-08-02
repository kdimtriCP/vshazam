# AI Services Configuration Guide

## Flexible API Key Configuration

You can use **either** OpenAI OR Google Vision, or both! The system will work with whatever you provide.

### Option 1: OpenAI Only (Recommended for captions)
```bash
OPENAI_API_KEY=sk-your-openai-key
# GOOGLE_VISION_API_KEY=  # Not required
```
**You get:** Detailed scene descriptions and movie identification hints

### Option 2: Google Vision Only (Recommended for text/labels)
```bash
# OPENAI_API_KEY=  # Not required
GOOGLE_VISION_API_KEY=your-google-key
```
**You get:** Object labels, OCR text detection, face counting, color analysis

### Option 3: Both APIs (Best results)
```bash
OPENAI_API_KEY=sk-your-openai-key
GOOGLE_VISION_API_KEY=your-google-key
```
**You get:** Everything combined with higher confidence scores

## What Each Service Provides

### OpenAI GPT-4o Vision
- **Scene descriptions**: "A noir detective scene in a dimly lit office"
- **Movie hints**: "Appears to be from a 1940s film noir"
- **Actor descriptions**: "Man in fedora and trench coat"
- **Context clues**: Setting, era, genre identification

### Google Vision API
- **Labels**: ["person", "car", "night", "city"]
- **OCR Text**: Reads on-screen text, titles, credits
- **Face Detection**: Number of faces and positions
- **Colors**: Dominant color palette

## Cost Comparison

### OpenAI
- ~$0.00064 per frame (512x512)
- ~$0.002 per video (3 frames)

### Google Vision
- First 1000 requests/month FREE
- Then $1.50 per 1000 requests
- ~$0.0015 per frame after free tier

## Quick Start Examples

### Minimal setup (OpenAI only):
```bash
export OPENAI_API_KEY=sk-your-key
go run cmd/server/main.go
```

### Free tier testing (Google only):
```bash
export GOOGLE_VISION_API_KEY=your-key
go run cmd/server/main.go
```

### Full setup:
```bash
cat > .env << EOF
OPENAI_API_KEY=sk-your-key
GOOGLE_VISION_API_KEY=your-key
MAX_FRAMES_PER_VIDEO=3
FRAME_SIZE=512
EOF

source .env && go run cmd/server/main.go
```

## Testing Your Configuration

After setting up, the server will log which services are enabled:
```
OpenAI Vision service enabled
Google Vision service disabled (no API key)
```
or
```
OpenAI Vision service disabled (no API key)
Google Vision service enabled
```

## Confidence Scores

The confidence score adapts based on available services:
- **Caption only** (OpenAI): Max 0.4
- **Labels only** (Google): Max 0.3 based on label confidence
- **OCR detected**: +0.2
- **Faces detected**: +0.1
- **Both services**: Combined scoring up to 1.0