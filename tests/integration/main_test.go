package integration

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Setup code here if needed
	
	// Run tests
	code := m.Run()
	
	// Cleanup code here if needed
	
	os.Exit(code)
}