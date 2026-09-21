package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"

	"forge-contrib/core"

	"github.com/joho/godotenv"
)

// Configuration variables
var (
	appID          string
	privateKeyPath string
	webhookSecret  string
	port           string
	botName        string
	userWhitelist  []string
	repoWhitelist  []string
)

const (
	thumbsUp   = "+1"
	thumbsDown = "-1"
	apiURL     = "https://api.github.com/repos"
)

func init() {
	// Load .env file if present
	_ = godotenv.Load()

	appID = os.Getenv("GITHUB_APP_ID")
	privateKeyPath = os.Getenv("GITHUB_PRIVATE_KEY_PATH")
	webhookSecret = os.Getenv("GITHUB_WEBHOOK_SECRET")
	port = os.Getenv("PORT")
	if port == "" {
		port = "5050"
	}

	botName = os.Getenv("BOT_NAME")
	if botName != "" {
		botName += " "
	}

	userWhitelist = parseList(os.Getenv("USER_WHITELIST"))
	repoWhitelist = parseList(os.Getenv("REPO_WHITELIST"))
}

func parseList(val string) []string {
	if val == "" {
		return []string{}
	}
	parts := strings.Split(val, ",")
	var result []string
	for _, p := range parts {
		result = append(result, strings.TrimSpace(p))
	}
	return result
}

func contains(slice []string, val string) bool {
	return slices.Contains(slice, val)
}

func tryCatchAction(token, repoFullName string, issueNumber int, commandType core.CommandType, username, value string) bool {
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
		"Accept":        "application/vnd.github+json",
	}
	baseURL := fmt.Sprintf("%s/%s", apiURL, repoFullName)
	issueURL := fmt.Sprintf("%s/issues/%d", baseURL, issueNumber)

	defer func() {
		if r := recover(); r != nil {
			// Catch potential panics similar to Python's except Exception
		}
	}()

	var success bool
	var err error

	switch commandType {
	case core.AddLabel:
		success, err = core.AddLabelAction(fmt.Sprintf("%s/labels", issueURL), headers, value, baseURL)
	case core.RmLabel:
		success, err = core.RmLabelAction(fmt.Sprintf("%s/labels", issueURL), headers, value)
	case core.AddMilestone:
		success, err = core.AddMilestoneAction(fmt.Sprintf("%s/milestones", baseURL), issueURL, headers, value)
	case core.RmMilestone:
		success, err = core.RmMilestoneAction(issueURL, headers)
	case core.SelfAssign:
		success, err = core.AssignSelfAction(issueURL, headers, username)
	case core.SelfUnassign:
		success, err = core.UnassignSelfAction(issueURL, headers, username)
	default:
		return false
	}

	return err == nil && success
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify GitHub signature
	if !core.VerifyGitHubSignature(r, webhookSecret) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	event := r.Header.Get("X-GitHub-Event")
	if event != "issue_comment" {
		writeJSONStatus(w, "ignored event")
		return
	}

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Bad request payload", http.StatusBadRequest)
		return
	}

	action, _ := payload["action"].(string)
	if action != "created" {
		writeJSONStatus(w, "ignored action")
		return
	}

	repo, ok := payload["repository"].(map[string]any)
	var repoFullName string
	if ok {
		repoFullName, _ = repo["full_name"].(string)
	}

	if !contains(repoWhitelist, repoFullName) {
		writeJSONStatus(w, "repository not whitelisted")
		return
	}

	sender, _ := payload["sender"].(map[string]any)
	username, _ := sender["login"].(string)

	if !contains(userWhitelist, username) {
		writeJSONStatus(w, "user not whitelisted")
		return
	}

	comment, _ := payload["comment"].(map[string]any)
	commentBody, _ := comment["body"].(string)
	commentBody = strings.TrimSpace(commentBody)
	commentLower := strings.ToLower(commentBody)

	commentID := int64(0)
	if idVal, ok := comment["id"].(float64); ok {
		commentID = int64(idVal)
	}

	installation, _ := payload["installation"].(map[string]any)
	// Handle dynamic JSON unmarshaled number types safely
	installationID := int64(0)
	if idVal, ok := installation["id"].(float64); ok {
		installationID = int64(idVal)
	}

	issue, _ := payload["issue"].(map[string]any)
	issueNumber := 0
	if numVal, ok := issue["number"].(float64); ok {
		issueNumber = int(numVal)
	}

	if botName != "" && !strings.HasPrefix(commentLower, strings.ToLower(botName)) {
		writeJSONStatus(w, "not an arietta command")
		return
	}

	commandStr := strings.TrimSpace(strings.TrimPrefix(commentBody, botName))
	token, err := core.GetInstallationAccessToken(installationID, privateKeyPath, appID)
	if err != nil {
		writeJSONStatus(w, "failed to get access token")
		return
	}

	success := false
	cmd := core.ParseCommand(commandStr)

	if cmd.Type != core.Unknown {
		success = tryCatchAction(token, repoFullName, issueNumber, cmd.Type, username, cmd.Value)
		if success {
			core.ExecuteCommand(cmd, username, issueNumber, repoFullName) // Pass repoFullName here
		}
	}

	reaction := thumbsDown
	if success {
		reaction = thumbsUp
	}

	reactionURL := fmt.Sprintf("%s/%s/issues/comments/%d/reactions", apiURL, repoFullName, commentID)
	_ = core.AddReaction(reactionURL, token, reaction)

	statusMsg := "failed command"
	if success {
		statusMsg = "success"
	}
	writeJSONStatus(w, statusMsg)
}

func writeJSONStatus(w http.ResponseWriter, status string) {
	w.Header().Set("Content-Type", dotenvKeyExistsContentType())
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": status})
}

func dotenvKeyExistsContentType() string {
	return "application/json"
}

func main() {
	http.HandleFunc("/webhook", webhookHandler)
	log.Printf("Server starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
