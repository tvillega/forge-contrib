package main

import (
	"encoding/json"
	"fmt"
	"forge-contrib/core"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// Constants and Configuration
const (
	ThumbsUp    = "+1"
	ThumbsDown  = "-1"
	ApiURL      = "https://api.github.com/repos"
	BotBaseName = "arietta"
	BotSuffix   = " " // could be called like `arietta, add label bug`
	BotName     = BotBaseName + BotSuffix
)

var WhiteList = map[string]bool{
	"tvillega": true,
}

var (
	appID          = os.Getenv("GITHUB_APP_ID")
	privateKeyPath = os.Getenv("GITHUB_PRIVATE_KEY_PATH")
	webhookSecret  = os.Getenv("GITHUB_WEBHOOK_SECRET")
	port           = getEnvOrDefault("PORT", "5050")
)

// JSON Payload Structs
type WebhookPayload struct {
	Action       string       `json:"action"`
	Installation Installation `json:"installation"`
	Repository   Repository   `json:"repository"`
	Issue        Issue        `json:"issue"`
	Comment      Comment      `json:"comment"`
	Sender       Sender       `json:"sender"`
}

type Installation struct {
	ID int64 `json:"id"`
}

type Repository struct {
	FullName string `json:"full_name"`
}

type Issue struct {
	Number int `json:"number"`
}

type Comment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

type Sender struct {
	Login string `json:"login"`
}

// tryCatchAction executes the specific command requested by the user.
func tryCatchAction(token, repoFullName string, issueNumber int, commandType, value string) bool {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/vnd.github+json",
	}
	baseURL := fmt.Sprintf("%s/%s", ApiURL, repoFullName)

	// Defer to catch any panics (analogous to Python's try-except)
	defer func() {
		if r := recover(); r != nil {
			// Handled exception case
		}
	}()

	if commandType == "add label" {
		core.AddLabel(fmt.Sprintf("%s/issues/%d/labels", baseURL, issueNumber), headers, value)
	} else if commandType == "add milestone" {
		core.AddMilestone(fmt.Sprintf("%s/milestones", baseURL), fmt.Sprintf("%s/issues/%d", baseURL, issueNumber), headers, value)
	}

	return true
}

// webhook handles incoming GitHub event requests.
func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	// 1. Read body bytes once so we can verify the signature AND decode JSON
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// 2. Fix the signature verification call with bodyBytes and handle failure
	if !core.VerifyGitHubSignature(webhookSecret, r, bodyBytes) {
		http.Error(w, "Signature mismatch", http.StatusForbidden)
		return
	}

	event := r.Header.Get("X-GitHub-Event")
	if event != "issue_comment" {
		writeJSON(w, map[string]string{"status": "ignored event"}, http.StatusOK)
		return
	}

	// 3. Decode the JSON from our saved bodyBytes instead of r.Body
	var payload WebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		writeJSON(w, map[string]string{"error": "invalid payload"}, http.StatusBadRequest)
		return
	}

	if payload.Action != "created" {
		writeJSON(w, map[string]string{"status": "ignored action"}, http.StatusOK)
		return
	}

	username := payload.Sender.Login
	if !WhiteList[username] {
		writeJSON(w, map[string]string{"status": "user not whitelisted"}, http.StatusOK)
		return
	}

	commentBody := strings.TrimSpace(payload.Comment.Body)
	commentLower := strings.ToLower(commentBody)
	commentID := payload.Comment.ID

	installationID := payload.Installation.ID
	repoFullName := payload.Repository.FullName
	issueNumber := payload.Issue.Number

	if !strings.HasPrefix(commentLower, BotName) {
		writeJSON(w, map[string]string{"status": "not an arietta command"}, http.StatusOK)
		return
	}

	commandStr := strings.TrimSpace(strings.TrimPrefix(commentBody, BotName))
	token, _ := core.GetInstallationAccessToken(installationID, privateKeyPath, appID)
	success := false

	if after, ok := strings.CutPrefix(commandStr, "add label "); ok {
		value := strings.TrimSpace(after)
		success = tryCatchAction(token, repoFullName, issueNumber, "add label", value)
	} else if after0, ok0 := strings.CutPrefix(commandStr, "add milestone "); ok0 {
		value := strings.TrimSpace(after0)
		success = tryCatchAction(token, repoFullName, issueNumber, "add milestone", value)
	}

	reaction := ThumbsDown
	if success {
		reaction = ThumbsUp
	}

	reactionURL := fmt.Sprintf("%s/%s/issues/comments/%d/reactions", ApiURL, repoFullName, commentID)
	err = core.AddReaction(reactionURL, token, reaction)
	if err != nil {
		return
	}

	statusMsg := "failed command"
	if success {
		statusMsg = "success"
	}
	writeJSON(w, map[string]string{"status": statusMsg}, http.StatusOK)
}

func writeJSON(w http.ResponseWriter, data map[string]string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		return
	}
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func main() {
	http.HandleFunc("/webhook", webhookHandler)
	log.Printf("Server running on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
