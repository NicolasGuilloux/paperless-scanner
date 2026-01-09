package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type ESCLScanner struct {
	baseURL string
}

// ScannerStatusResponse represents the eSCL ScannerStatus XML response
type ScannerStatusResponse struct {
	XMLName xml.Name `xml:"ScannerStatus"`
	State   string   `xml:"State"`
	Version string   `xml:"Version"`
}

func NewESCLScanner(baseURL string) *ESCLScanner {
	return &ESCLScanner{baseURL: baseURL}
}

// getScannerStatus queries the scanner status endpoint and returns the parsed status
func (s *ESCLScanner) getScannerStatus() (*ScannerStatusResponse, error) {
	url := fmt.Sprintf("%s/eSCL/ScannerStatus", s.baseURL)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query scanner status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scanner status returned non-OK status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read scanner status response: %w", err)
	}

	var status ScannerStatusResponse
	if err := xml.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("failed to parse scanner status XML: %w", err)
	}

	return &status, nil
}

// promptUserToDismissError prompts the user to dismiss the error on the printer screen
func (s *ESCLScanner) promptUserToDismissError() bool {
	fmt.Println("\n⚠️  The scanner appears to have an error displayed on its screen.")
	fmt.Println("Please check the scanner and dismiss any error messages.")
	fmt.Print("Press Enter to retry scanning, or type 'quit' to abort: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Error reading user input: %v", err)
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input != "quit" && input != "q"
}

// getESCLMimeType maps file extensions to eSCL MIME types
func getESCLMimeType(ext string) (string, error) {
	ext = strings.ToLower(ext)
	switch ext {
	case ".pdf":
		return "application/pdf", nil
	case ".png":
		return "image/png", nil
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	default:
		return "", fmt.Errorf("unsupported output format: %s (supported: pdf, png, jpg, jpeg)", ext)
	}
}

// Scan performs a scan using the eSCL (AirScan) protocol and returns the path to the scanned file
// format specifies the output format extension (e.g., ".pdf", ".png", ".jpg")
func (s *ESCLScanner) Scan(format string) (string, error) {
	// Default to PDF if no format specified
	if format == "" {
		format = ".pdf"
	}

	// Validate and get MIME type
	mimeType, err := getESCLMimeType(format)
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

	log.Printf("Scanning via eSCL to: %s (format: %s)", outputFile, mimeType)

	// Create scan job with specified format
	scanSettings := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<scan:ScanSettings xmlns:scan="http://schemas.hp.com/imaging/escl/2011/05/03" xmlns:pwg="http://www.pwg.org/schemas/2010/12/sm">
  <pwg:Version>2.0</pwg:Version>
  <scan:Intent>Document</scan:Intent>
  <pwg:ScanRegions>
    <pwg:ScanRegion>
      <pwg:ContentRegionUnits>escl:ThreeHundredthsOfInches</pwg:ContentRegionUnits>
      <pwg:XOffset>0</pwg:XOffset>
      <pwg:YOffset>0</pwg:YOffset>
      <pwg:Width>2550</pwg:Width>
      <pwg:Height>3508</pwg:Height>
    </pwg:ScanRegion>
  </pwg:ScanRegions>
  <scan:InputSource>Platen</scan:InputSource>
  <scan:ColorMode>RGB24</scan:ColorMode>
  <scan:XResolution>300</scan:XResolution>
  <scan:YResolution>300</scan:YResolution>
  <pwg:DocumentFormat>%s</pwg:DocumentFormat>
</scan:ScanSettings>`, mimeType)

	// Submit scan job
	jobURL, err := s.createScanJob(scanSettings)
	if err != nil {
		return "", fmt.Errorf("failed to create scan job: %w", err)
	}

	log.Printf("Scan job created: %s", jobURL)

	// Wait a bit for scanning to start
	time.Sleep(2 * time.Second)

	// Download the scanned document
	if err := s.downloadDocument(jobURL, outputFile); err != nil {
		return "", fmt.Errorf("failed to download scan: %w", err)
	}

	// Verify the file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		return "", fmt.Errorf("scan file was not created: %s", outputFile)
	}

	return outputFile, nil
}

func (s *ESCLScanner) createScanJob(settings string) (string, error) {
	url := fmt.Sprintf("%s/eSCL/ScanJobs", s.baseURL)
	maxRetries := 5
	userPromptedFor503 := false

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retrying scan job creation (attempt %d/%d)...", attempt+1, maxRetries)
			time.Sleep(2 * time.Second)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBufferString(settings))
		if err != nil {
			return "", err
		}

		req.Header.Set("Content-Type", "text/xml")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}

		if resp.StatusCode == http.StatusCreated {
			// Get the job location from the Location header
			location := resp.Header.Get("Location")
			resp.Body.Close()
			if location == "" {
				return "", fmt.Errorf("no Location header in response")
			}
			return location, nil
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Handle 503 Service Unavailable - likely an error on scanner screen
		if resp.StatusCode == http.StatusServiceUnavailable && !userPromptedFor503 {
			log.Printf("Scanner returned 503 Service Unavailable")

			// Query scanner status to check for errors
			status, err := s.getScannerStatus()
			if err != nil {
				log.Printf("Warning: Failed to query scanner status: %v", err)
			} else {
				log.Printf("Scanner state: %s", status.State)
			}

			// Prompt user to dismiss error regardless of status check result
			// (503 typically means there's an error displayed)
			if !s.promptUserToDismissError() {
				return "", fmt.Errorf("scan aborted by user")
			}
			userPromptedFor503 = true
			// Reset retry counter after user dismisses error
			attempt = -1 // Will be 0 after continue
			continue
		}

		// For last attempt or non-503 errors, return the error
		if attempt == maxRetries-1 || resp.StatusCode != http.StatusServiceUnavailable {
			return "", fmt.Errorf("failed to create scan job, status: %d, body: %s", resp.StatusCode, string(body))
		}
	}

	return "", fmt.Errorf("failed to create scan job after %d attempts", maxRetries)
}

func (s *ESCLScanner) downloadDocument(jobURL string, outputFile string) error {
	// Extract job ID from URL
	re := regexp.MustCompile(`/ScanJobs/(.+)$`)
	matches := re.FindStringSubmatch(jobURL)
	if len(matches) < 2 {
		return fmt.Errorf("invalid job URL: %s", jobURL)
	}
	jobID := matches[1]

	// Build document URL
	docURL := fmt.Sprintf("%s/eSCL/ScanJobs/%s/NextDocument", s.baseURL, jobID)

	// Poll for document (it might not be ready immediately)
	maxRetries := 30
	userPromptedFor503 := false

	for i := 0; i < maxRetries; i++ {
		log.Printf("Attempting to download scan (attempt %d/%d)...", i+1, maxRetries)

		resp, err := http.Get(docURL)
		if err != nil {
			return err
		}

		if resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()

			// Save the document
			outFile, err := os.Create(outputFile)
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, resp.Body)
			if err != nil {
				return err
			}

			log.Printf("Scan downloaded successfully")
			return nil
		}

		resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			// Document not ready yet, wait and retry
			time.Sleep(1 * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusServiceUnavailable {
			// 503 might indicate an error on the scanner screen
			// Check scanner status to determine if we should prompt the user
			if !userPromptedFor503 {
				status, err := s.getScannerStatus()
				if err != nil {
					log.Printf("Warning: Failed to query scanner status: %v", err)
				} else {
					log.Printf("Scanner state: %s", status.State)
					// If scanner is not in a normal state, it might have an error displayed
					if status.State != "Idle" && status.State != "Processing" {
						// Prompt user to dismiss error
						if !s.promptUserToDismissError() {
							return fmt.Errorf("scan aborted by user")
						}
						userPromptedFor503 = true
						// Reset retry counter to give more attempts after user dismisses error
						i = 0
						continue
					}
				}
			}
			// Document not ready yet, wait and retry
			time.Sleep(1 * time.Second)
			continue
		}

		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return fmt.Errorf("scan document not ready after %d attempts", maxRetries)
}
