package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Scanner struct {
	url string
}

func NewScanner(url string) *Scanner {
	return &Scanner{url: url}
}

// getScanImageFormat maps file extensions to scanimage format names
func getScanImageFormat(ext string) (string, error) {
	ext = strings.ToLower(ext)
	switch ext {
	case ".pdf":
		return "pdf", nil
	case ".png":
		return "png", nil
	case ".jpg", ".jpeg":
		return "jpeg", nil
	default:
		return "", fmt.Errorf("unsupported output format: %s (supported: pdf, png, jpg, jpeg)", ext)
	}
}

// Scan performs a scan using the SANE network scanner and returns the path to the scanned file
// format specifies the output format extension (e.g., ".pdf", ".png", ".jpg")
func (s *Scanner) Scan(format string) (string, error) {
	// Default to PDF if no format specified
	if format == "" {
		format = ".pdf"
	}

	// Validate and get scanimage format
	scanImageFormat, err := getScanImageFormat(format)
	if err != nil {
		return "", err
	}

	// Create temporary directory for scans
	tmpDir := filepath.Join(os.TempDir(), "paperless-scanner")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Generate output filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	outputFile := filepath.Join(tmpDir, fmt.Sprintf("scan-%s%s", timestamp, format))

	log.Printf("Scanning to: %s (format: %s)", outputFile, scanImageFormat)

	// Build scanimage command
	// For Canon PIXMA network scanners, use pixma:<model>_<ip> format
	// For other network scanners, use net:<ip> format
	// The device name should match the output from scanimage -L
	deviceName := s.url

	cmd := exec.Command("scanimage",
		"--device-name", deviceName,
		"--format", scanImageFormat,
		"--output-file", outputFile,
		"--progress",
		"--resolution", "300", // 300 DPI for good quality
		"--mode", "Color",     // Color scanning
	)

	// Capture output for logging
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("scanimage command failed: %w\nOutput: %s", err, string(output))
	}

	log.Printf("Scanimage output: %s", string(output))

	// Verify the file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		return "", fmt.Errorf("scan file was not created: %s", outputFile)
	}

	return outputFile, nil
}
