package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type PaperlessClient struct {
	baseURL string
	token   string
	client  *http.Client
}

type PaperlessResponse struct {
	ID int `json:"id"`
}

func NewPaperlessClient(baseURL, token string) *PaperlessClient {
	// Remove trailing slash from baseURL
	baseURL = strings.TrimRight(baseURL, "/")

	return &PaperlessClient{
		baseURL: baseURL,
		token:   token,
		client:  &http.Client{},
	}
}

// UploadDocument uploads a document to Paperless-ngx and returns the document ID
func (p *PaperlessClient) UploadDocument(filePath string) (int, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add document file
	part, err := writer.CreateFormFile("document", filepath.Base(filePath))
	if err != nil {
		return 0, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return 0, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Close the writer to finalize the multipart message
	if err := writer.Close(); err != nil {
		return 0, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create the request
	url := fmt.Sprintf("%s/api/documents/post_document/", p.baseURL)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", p.token))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send the request
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response to get document ID
	var paperlessResp PaperlessResponse
	if err := json.Unmarshal(respBody, &paperlessResp); err != nil {
		// If we can't parse the response but upload succeeded, return 0 as ID
		return 0, nil
	}

	return paperlessResp.ID, nil
}
