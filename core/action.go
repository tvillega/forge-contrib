package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// AddReaction adds a reaction (+1 or -1) to the triggering comment.
func AddReaction(url, token, reaction string) error {
	payload := map[string]string{"content": reaction}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal reaction payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// AddLabel creates or adds a label to the issue/PR.
func AddLabel(url string, headers map[string]string, value string) bool {
	payload := map[string][]string{"labels": {value}}
	body, err := json.Marshal(payload)
	if err != nil {
		return false
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return false
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated
}

// Milestone represents a GitHub milestone object for parsing JSON.
type Milestone struct {
	Title  string `json:"title"`
	Number int    `json:"number"`
}

// AddMilestone checks if a milestone exists by title and assigns it to the issue.
func AddMilestone(urlMilestone, urlIssue string, headers map[string]string, value string) bool {
	// 1. Fetch existing milestones
	reqGet, err := http.NewRequest(http.MethodGet, urlMilestone, nil)
	if err != nil {
		return false
	}

	for k, v := range headers {
		reqGet.Header.Set(k, v)
	}

	respGet, err := httpClient.Do(reqGet)
	if err != nil || respGet.StatusCode != http.StatusOK {
		if respGet != nil {
			respGet.Body.Close()
		}
		return false
	}
	defer respGet.Body.Close()

	var milestones []Milestone
	if err := json.NewDecoder(respGet.Body).Decode(&milestones); err != nil {
		return false
	}

	// 2. Find matching milestone ID (case-insensitive)
	var milestoneID *int
	for _, m := range milestones {
		if strings.EqualFold(m.Title, value) {
			num := m.Number
			milestoneID = &num
			break
		}
	}

	if milestoneID == nil {
		return false
	}

	// 3. Patch the issue with the milestone ID
	payload := map[string]int{"milestone": *milestoneID}
	body, err := json.Marshal(payload)
	if err != nil {
		return false
	}

	reqPatch, err := http.NewRequest(http.MethodPatch, urlIssue, bytes.NewBuffer(body))
	if err != nil {
		return false
	}

	for k, v := range headers {
		reqPatch.Header.Set(k, v)
	}
	reqPatch.Header.Set("Content-Type", "application/json")

	respPatch, err := httpClient.Do(reqPatch)
	if err != nil {
		return false
	}
	defer respPatch.Body.Close()

	return respPatch.StatusCode == http.StatusOK
}
