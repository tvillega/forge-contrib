package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AccessTokenResponse maps the JSON response from GitHub when requesting an installation token.
type AccessTokenResponse struct {
	Token string `json:"token"`
}

// VerifyGitHubSignature validates the X-Hub-Signature-256 header sent by GitHub.
// Note: In Go, we pass the read body bytes explicitly since http.Request body can only be read once.
func VerifyGitHubSignature(webhookSecret string, r *http.Request, bodyBytes []byte) bool {
	signatureHeader := r.Header.Get("X-Hub-Signature-256")
	if signatureHeader == "" {
		return false
	}

	parts := strings.SplitN(signatureHeader, "=", 2)
	if len(parts) != 2 || parts[0] != "sha256" {
		return false
	}
	receivedSig := parts[1]

	// Compute HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write(bodyBytes)
	expectedMAC := mac.Sum(nil)
	expectedSig := hex.EncodeToString(expectedMAC)

	// Constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(expectedSig), []byte(receivedSig)) == 1
}

// GenerateJWT generates a JWT signed with RS256 for authenticating as the GitHub App.
func GenerateJWT(privateKeyPath, appID string) (string, error) {
	keyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key file: %w", err)
	}

	// Parse the RSA private key from PEM bytes
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse RSA private key: %w", err)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(10 * time.Minute).Unix(), // JWT expires in 10 minutes
		"iss": appID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

// GetInstallationAccessToken obtains an installation access token for repository API requests.
func GetInstallationAccessToken(installationID int64, privateKeyPath, appID string) (string, error) {
	jwtToken, err := GenerateJWT(privateKeyPath, appID)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request for installation token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api returned non-success status: %d", resp.StatusCode)
	}

	var result AccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode installation token response: %w", err)
	}

	return result.Token, nil
}
