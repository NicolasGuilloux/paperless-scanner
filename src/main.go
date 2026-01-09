package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	ScannerURL    string
	PaperlessURL  string
	PaperlessToken string
}

func loadConfig(requirePaperless bool, scannerURL, paperlessURL, paperlessToken string) (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using environment variables")
	}

	config := &Config{
		ScannerURL:     scannerURL,
		PaperlessURL:   paperlessURL,
		PaperlessToken: paperlessToken,
	}

	// Use environment variables as fallback if flags not provided
	if config.ScannerURL == "" {
		config.ScannerURL = os.Getenv("SCANNER_URL")
	}
	if config.PaperlessURL == "" {
		config.PaperlessURL = os.Getenv("PAPERLESS_URL")
	}
	if config.PaperlessToken == "" {
		config.PaperlessToken = os.Getenv("PAPERLESS_TOKEN")
	}

	// Validate required fields
	if config.ScannerURL == "" {
		return nil, fmt.Errorf("SCANNER_URL is required (set via --scanner_url flag or SCANNER_URL env var)")
	}

	// Only validate Paperless config if uploading
	if requirePaperless {
		if config.PaperlessURL == "" {
			return nil, fmt.Errorf("PAPERLESS_URL is required when using --upload-to-paperless (set via --paperless_url flag or PAPERLESS_URL env var)")
		}
		if config.PaperlessToken == "" {
			return nil, fmt.Errorf("PAPERLESS_TOKEN is required when using --upload-to-paperless (set via --paperless_token flag or PAPERLESS_TOKEN env var)")
		}
	}

	return config, nil
}

func main() {
	// Define CLI flags
	outputPath := flag.String("output", "", "Save scan to this path (required if not uploading to Paperless)")
	uploadToPaperless := flag.Bool("upload-to-paperless", false, "Upload scan to Paperless-ngx")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	scannerURL := flag.String("scanner_url", "", "Scanner URL (overrides SCANNER_URL env var)")
	paperlessURL := flag.String("paperless_url", "", "Paperless URL (overrides PAPERLESS_URL env var)")
	paperlessToken := flag.String("paperless_token", "", "Paperless API token (overrides PAPERLESS_TOKEN env var)")

	// Customize usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A CLI tool that scans documents from SANE/eSCL scanners with optional upload to Paperless-ngx.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Scan and save to file\n")
		fmt.Fprintf(os.Stderr, "  %s -output scan.pdf\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Scan and upload to Paperless\n")
		fmt.Fprintf(os.Stderr, "  %s --upload-to-paperless\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Scan, save, and upload\n")
		fmt.Fprintf(os.Stderr, "  %s -output scan.pdf --upload-to-paperless\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Environment variables (can be set in .env file):\n")
		fmt.Fprintf(os.Stderr, "  SCANNER_URL         Scanner device URL (required)\n")
		fmt.Fprintf(os.Stderr, "  PAPERLESS_URL       Paperless-ngx server URL\n")
		fmt.Fprintf(os.Stderr, "  PAPERLESS_TOKEN     Paperless-ngx API token\n")
	}

	flag.Parse()

	// Show help if no arguments provided
	if flag.NFlag() == 0 && len(os.Args) == 1 {
		flag.Usage()
		os.Exit(0)
	}

	// Setup logging
	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Validate flags
	if !*uploadToPaperless && *outputPath == "" {
		log.Fatalf("Error: Must specify either --upload-to-paperless or -output (or both)")
	}

	// Load configuration
	config, err := loadConfig(*uploadToPaperless, *scannerURL, *paperlessURL, *paperlessToken)
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	log.Printf("Starting scan from %s", config.ScannerURL)

	// Determine output format from -output flag extension, default to PDF
	outputFormat := ".pdf"
	if *outputPath != "" {
		outputFormat = filepath.Ext(*outputPath)
		if outputFormat == "" {
			log.Fatalf("Output path must have a file extension (e.g., .pdf, .png, .jpg)")
		}
	}

	// Perform scan using the appropriate scanner type
	var scanFile string

	// Check if it's an HTTP/HTTPS URL (eSCL scanner) or a SANE device
	if len(config.ScannerURL) >= 7 && (config.ScannerURL[:7] == "http://" || config.ScannerURL[:8] == "https://") {
		// Use eSCL scanner for HTTP URLs
		esclScanner := NewESCLScanner(config.ScannerURL)
		scanFile, err = esclScanner.Scan(outputFormat)
	} else {
		// Use SANE scanner for device names
		scanner := NewScanner(config.ScannerURL)
		scanFile, err = scanner.Scan(outputFormat)
	}
	if err != nil {
		log.Fatalf("Scan failed: %v", err)
	}

	log.Printf("Scan completed: %s", scanFile)

	// Save locally if -output specified
	if *outputPath != "" {
		if err := os.Rename(scanFile, *outputPath); err != nil {
			log.Fatalf("Failed to save file: %v", err)
		}
		log.Printf("Scan saved to: %s", *outputPath)
		scanFile = *outputPath // Update scanFile to the new path for potential upload
	}

	// Upload to Paperless if --upload-to-paperless specified
	if *uploadToPaperless {
		client := NewPaperlessClient(config.PaperlessURL, config.PaperlessToken)
		docID, err := client.UploadDocument(scanFile)
		if err != nil {
			log.Fatalf("Upload to Paperless failed: %v", err)
		}
		log.Printf("Successfully uploaded to Paperless (Document ID: %d)", docID)
	}

	// Cleanup temporary file if we didn't save it locally
	if *outputPath == "" && scanFile != "" {
		os.Remove(scanFile)
		log.Printf("Cleaned up temporary file: %s", scanFile)
	}
}
