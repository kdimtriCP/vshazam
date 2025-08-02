package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./vshazam.db"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	fmt.Println("🔍 Checking AI Analysis Results")
	fmt.Println("================================")

	// Check if AI is configured
	openAIKey := os.Getenv("OPENAI_API_KEY")
	googleKey := os.Getenv("GOOGLE_VISION_API_KEY")

	if openAIKey == "" && googleKey == "" {
		fmt.Println("⚠️  WARNING: No AI API keys configured!")
		fmt.Println("   Set at least one: OPENAI_API_KEY or GOOGLE_VISION_API_KEY")
		fmt.Println()
	} else {
		fmt.Println("✅ AI Services configured:")
		if openAIKey != "" {
			fmt.Println("   - OpenAI Vision: Enabled")
		} else {
			fmt.Println("   - OpenAI Vision: Disabled")
		}
		if googleKey != "" {
			fmt.Println("   - Google Vision: Enabled")
		} else {
			fmt.Println("   - Google Vision: Disabled")
		}
		fmt.Println()
	}

	// Count total videos
	var videoCount int
	err = db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
	if err != nil {
		log.Fatal("Failed to count videos:", err)
	}
	fmt.Printf("📹 Total videos: %d\n", videoCount)

	// Count analyzed frames
	var frameCount int
	err = db.QueryRow("SELECT COUNT(*) FROM frame_analyses").Scan(&frameCount)
	if err != nil {
		fmt.Println("❌ No frame_analyses table found (AI not yet used)")
		return
	}
	fmt.Printf("🖼️  Total analyzed frames: %d\n\n", frameCount)

	// Show recent analyses
	rows, err := db.Query(`
		SELECT 
			v.title,
			fa.frame_number,
			fa.gpt_caption,
			fa.vision_labels,
			fa.ocr_text,
			fa.face_count
		FROM frame_analyses fa
		JOIN videos v ON fa.video_id = v.id
		ORDER BY fa.analysis_time DESC
		LIMIT 5
	`)
	if err != nil {
		log.Fatal("Failed to query analyses:", err)
	}
	defer rows.Close()

	fmt.Println("📊 Recent AI Analyses:")
	fmt.Println("---------------------")

	count := 0
	for rows.Next() {
		var title string
		var frameNum int
		var caption string
		var labelsJSON string
		var ocrJSON string
		var faceCount int

		err := rows.Scan(&title, &frameNum, &caption, &labelsJSON, &ocrJSON, &faceCount)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		count++
		fmt.Printf("\n🎬 Video: %s (Frame %d)\n", title, frameNum)

		if caption != "" {
			fmt.Printf("   📝 Caption: %.100s...\n", caption)
		}

		// Parse labels
		if labelsJSON != "" {
			var labels []map[string]interface{}
			if err := json.Unmarshal([]byte(labelsJSON), &labels); err == nil && len(labels) > 0 {
				fmt.Printf("   🏷️  Labels: ")
				for i, label := range labels {
					if i > 0 {
						fmt.Print(", ")
					}
					if name, ok := label["name"].(string); ok {
						fmt.Print(name)
					}
					if i >= 2 {
						fmt.Print("...")
						break
					}
				}
				fmt.Println()
			}
		}

		// Parse OCR
		if ocrJSON != "" && ocrJSON != "[]" {
			var ocrTexts []string
			if err := json.Unmarshal([]byte(ocrJSON), &ocrTexts); err == nil && len(ocrTexts) > 0 {
				fmt.Printf("   📄 OCR Text: %v\n", ocrTexts)
			}
		}

		if faceCount > 0 {
			fmt.Printf("   👤 Faces detected: %d\n", faceCount)
		}
	}

	if count == 0 {
		fmt.Println("No AI analyses found yet. Upload a video to test!")
	} else {
		fmt.Printf("\n✅ AI integration is working! Found %d recent analyses.\n", count)
	}
}
