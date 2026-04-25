package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Modality string

const (
	ModalityImage Modality = "image"
	ModalityVideo Modality = "video"
	ModalityAudio Modality = "audio"
)

type ModelClient interface {
	Detect(ctx context.Context, modality Modality, ossURL string, taskID string, userID string) (resultJSON []byte, err error)
}

type DetectRequest struct {
	OssURL string `json:"oss_url"`
	TaskID string `json:"task_id"`
	UserID string `json:"user_id"`
}

type HTTPModelClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPModelClient(baseURL string, timeoutSec int) *HTTPModelClient {
	return &HTTPModelClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
	}
}

func (c *HTTPModelClient) Detect(ctx context.Context, modality Modality, ossURL string, taskID string, userID string) ([]byte, error) {
	var endpoint string
	switch modality {
	case ModalityImage:
		endpoint = "/api/detect/image"
	case ModalityVideo:
		endpoint = "/api/detect/video"
	case ModalityAudio:
		endpoint = "/api/detect/audio"
	default:
		return nil, fmt.Errorf("unsupported modality: %s", modality)
	}

	body, _ := json.Marshal(DetectRequest{OssURL: ossURL, TaskID: taskID, UserID: userID})
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("modelserver %d: %s", resp.StatusCode, truncate(string(raw), 1024))
	}
	return raw, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
