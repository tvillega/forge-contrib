package core

import (
	"bytes"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// VerifyGitHubSignature verifies the X-Hub-Signature-256 header from GitHub webhooks.
func VerifyGitHubSignature(r *http.Request, webhookSecret string) bool {
	signatureHeader := r.Header.Get("X-Hub-Signature-256")
	if signatureHeader == "" || !strings.HasPrefix(signatureHeader, "sha256=") {
		return false
	}

	sigParts := strings.SplitN(signatureHeader, "=", 2)
	if len(sigParts) != 2 {
		return false
	}
	expectedMAC, err := hex.DecodeString(sigParts[1])
	if err != nil {
		return false
	}

	// Read body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	// Restore the io.ReadCloser so downstream handlers can read it if needed
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write(bodyBytes)
	calculatedMAC := mac.Sum(nil)

	return hmac.Equal(expectedMAC, calculatedMAC)
}

// GetInstallationAccessToken generates a JWT using the GitHub App's private key and App ID,
// then exchanges it for an installation access token.
func GetInstallationAccessToken(installationID int64, privateKeyPath, appID string) (string, error) {
	if installationID == 0 || privateKeyPath == "" || appID == "" {
		return "", fmt.Errorf("missing parameters for installation token generation")
	}

	// 1. Read private key file
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(privateKeyBytes)
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM block containing the private key")
	}

	var parsedKey any
	if parsed, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		parsedKey = parsed
	} else if parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		var ok bool
		parsedKey, ok = parsed.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("not an RSA private key")
		}
	} else {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	// 2. Create JWT token
	now := time.Now()
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now.Add(-60 * time.Second)), // clock drift leeway
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
		Issuer:    appID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedJWT, err := token.SignedString(parsedKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	// 3. Request installation access token from GitHub API
	tokenURL := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	req, err := http.NewRequest(http.MethodPost, tokenURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", signedJWT))
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get installation access token, status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	if tokenResp.Token == "" {
		return "", fmt.Errorf("received empty token from GitHub")
	}

	return tokenResp.Token, nil
}
