package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Helper function to execute HTTP requests with headers and optional JSON body
func doRequest(method, url string, headers map[string]string, payload any) (*http.Response, error) {
	var bodyReader *bytes.Reader
	if payload != nil {
		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	return client.Do(req)
}

// AddReaction adds a reaction (+1 or -1) to the triggering comment.
func AddReaction(url, token, reaction string) error {
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
		"Accept":        "application/vnd.github+json",
	}
	payload := map[string]string{"content": reaction}
	resp, err := doRequest(http.MethodPost, url, headers, payload)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)
	return nil
}

// addOrCreateLabel internal helper
func addOrCreateLabel(url string, headers map[string]string, value string) bool {
	payload := map[string]any{
		"labels": []string{value},
	}
	resp, err := doRequest(http.MethodPost, url, headers, payload)
	if err != nil {
		return false
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated
}

// AddLabelAction checks if label exists on the repo first, then adds it.
func AddLabelAction(labelsURL string, headers map[string]string, value, baseURL string) (bool, error) {
	labelCheckURL := fmt.Sprintf("%s/labels/%s", baseURL, value)
	resp, err := doRequest(http.MethodGet, labelCheckURL, headers, nil)
	if err != nil {
		return false, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode == http.StatusOK {
		return addOrCreateLabel(labelsURL, headers, value), nil
	}
	return false, nil
}

// RmLabelAction removes a label from an issue.
func RmLabelAction(labelsURL string, headers map[string]string, value string) (bool, error) {
	removeURL := fmt.Sprintf("%s/%s", labelsURL, value)
	resp, err := doRequest(http.MethodDelete, removeURL, headers, nil)
	if err != nil {
		return false, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent, nil
}

// AddMilestoneAction adds an existing milestone to an issue.
func AddMilestoneAction(urlMilestone, urlIssue string, headers map[string]string, value string) (bool, error) {
	resp, err := doRequest(http.MethodGet, urlMilestone, headers, nil)
	if err != nil {
		return false, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var milestones []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&milestones); err != nil {
		return false, err
	}

	var milestoneID any
	for _, m := range milestones {
		title, ok := m["title"].(string)
		if ok && strings.EqualFold(title, value) {
			milestoneID = m["number"]
			break
		}
	}

	if milestoneID == nil {
		return false, nil
	}

	payload := map[string]any{"milestone": milestoneID}
	patchResp, err := doRequest(http.MethodPatch, urlIssue, headers, payload)
	if err != nil {
		return false, err
	}
	defer patchResp.Body.Close()

	return patchResp.StatusCode == http.StatusOK, nil
}

// RmMilestoneAction removes the milestone from an issue.
func RmMilestoneAction(urlIssue string, headers map[string]string) (bool, error) {
	payload := map[string]any{"milestone": nil}
	resp, err := doRequest(http.MethodPatch, urlIssue, headers, payload)
	if err != nil {
		return false, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	return resp.StatusCode == http.StatusOK, nil
}

// AssignSelfAction assigns an issue to the specified username.
func AssignSelfAction(urlIssue string, headers map[string]string, username string) (bool, error) {
	payload := map[string]any{
		"assignees": []string{username},
	}
	resp, err := doRequest(http.MethodPatch, urlIssue, headers, payload)
	if err != nil {
		return false, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	return resp.StatusCode == http.StatusOK, nil
}

// UnassignSelfAction removes the specified username from the issue's assignees.
func UnassignSelfAction(urlIssue string, headers map[string]string, username string) (bool, error) {
	resp, err := doRequest(http.MethodGet, urlIssue, headers, nil)
	if err != nil {
		return false, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var issueData map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&issueData); err != nil {
		return false, err
	}

	var currentAssignees []string
	if assignees, ok := issueData["assignees"].([]any); ok {
		for _, a := range assignees {
			if assigneeMap, ok := a.(map[string]any); ok {
				if login, ok := assigneeMap["login"].(string); ok {
					currentAssignees = append(currentAssignees, login)
				}
			}
		}
	}

	// Filter out the username
	var updatedAssignees []string
	found := false
	for _, u := range currentAssignees {
		if u == username {
			found = true
			continue
		}
		updatedAssignees = append(updatedAssignees, u)
	}

	if !found {
		return true, nil // Already unassigned
	}

	payload := map[string]any{
		"assignees": updatedAssignees,
	}
	patchResp, err := doRequest(http.MethodPatch, urlIssue, headers, payload)
	if err != nil {
		return false, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(patchResp.Body)

	return patchResp.StatusCode == http.StatusOK, nil
}

// UpdateIssueState opens or closes a GitHub issue depending on the state ("open" or "closed").
func UpdateIssueState(url, token, state string) (map[string]any, error) {
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
		"Accept":        "application/vnd.github+json",
	}
	if state == "" {
		state = "closed"
	}
	payload := map[string]string{"state": state}
	resp, err := doRequest(http.MethodPatch, url, headers, payload)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	var result map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}
