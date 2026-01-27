package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/browser"
)

// decodeJSON decodes JSON from a reader into the target
func decodeJSON(r io.Reader, target interface{}) error {
	return json.NewDecoder(r).Decode(target)
}

const (
	// OAuthCallbackPort is the port for the local OAuth callback server.
	// Using 19876 to match opencode for consistency.
	OAuthCallbackPort = 19876

	// OAuthTimeout is the maximum time to wait for OAuth callback
	OAuthTimeout = 5 * time.Minute
)

// OAuthConfig holds configuration for an OAuth provider
type OAuthConfig struct {
	ProviderName string
	AuthURL      string
	TokenURL     string
	ClientID     string
	ClientSecret string // Optional, some flows don't need it
	Scopes       []string
	RedirectURI  string // Will be set automatically if empty
}

// OAuthResult contains the result of a successful OAuth flow
type OAuthResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // seconds until expiry
	TokenType    string
}

// StartOAuthFlow initiates the OAuth authorization code flow.
// It opens the user's browser to the authorization URL and starts a local
// server to receive the callback. Returns the authorization code on success.
func StartOAuthFlow(ctx context.Context, config OAuthConfig) (*OAuthResult, error) {
	// Generate CSRF state token
	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Set default redirect URI
	if config.RedirectURI == "" {
		config.RedirectURI = fmt.Sprintf("http://localhost:%d/callback", OAuthCallbackPort)
	}

	// Channel to receive the authorization code
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Start local callback server
	server, err := startCallbackServer(state, codeChan, errChan)
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	defer func() { _ = server.Close() }()

	// Build authorization URL
	authURL, err := buildAuthURL(config, state)
	if err != nil {
		return nil, fmt.Errorf("failed to build auth URL: %w", err)
	}

	// Open browser
	fmt.Printf("Opening browser for %s authentication...\n", config.ProviderName)
	fmt.Printf("If browser doesn't open, visit: %s\n\n", authURL)

	if err := browser.OpenURL(authURL); err != nil {
		fmt.Printf("Could not open browser automatically. Please visit the URL above.\n")
	}

	// Wait for callback with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, OAuthTimeout)
	defer cancel()

	select {
	case code := <-codeChan:
		// Exchange code for tokens
		return exchangeCodeForTokens(config, code)
	case err := <-errChan:
		return nil, err
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("OAuth flow timed out after %v", OAuthTimeout)
	}
}

// RefreshAccessToken uses a refresh token to get a new access token
func RefreshAccessToken(config OAuthConfig, refreshToken string) (*OAuthResult, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", config.ClientID)
	if config.ClientSecret != "" {
		data.Set("client_secret", config.ClientSecret)
	}

	resp, err := http.PostForm(config.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return parseTokenResponse(resp)
}

// generateState creates a random state string for CSRF protection
func generateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// buildAuthURL constructs the OAuth authorization URL
func buildAuthURL(config OAuthConfig, state string) (string, error) {
	u, err := url.Parse(config.AuthURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("client_id", config.ClientID)
	q.Set("redirect_uri", config.RedirectURI)
	q.Set("response_type", "code")
	q.Set("state", state)
	if len(config.Scopes) > 0 {
		q.Set("scope", strings.Join(config.Scopes, " "))
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// startCallbackServer starts a local HTTP server to receive the OAuth callback
func startCallbackServer(expectedState string, codeChan chan<- string, errChan chan<- error) (*http.Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", OAuthCallbackPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %d: %w", OAuthCallbackPort, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Validate state to prevent CSRF
		state := r.URL.Query().Get("state")
		if state != expectedState {
			errChan <- fmt.Errorf("invalid state parameter - potential CSRF attack")
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlError("Authentication failed: invalid state parameter")))
			return
		}

		// Check for error response
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			errDesc := r.URL.Query().Get("error_description")
			errChan <- fmt.Errorf("OAuth error: %s - %s", errMsg, errDesc)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlError(fmt.Sprintf("Authentication failed: %s", errDesc))))
			return
		}

		// Extract authorization code
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no authorization code in callback")
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlError("Authentication failed: no authorization code")))
			return
		}

		// Success - send code and show success page
		codeChan <- code
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(htmlSuccess()))
	})

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	return server, nil
}

// exchangeCodeForTokens exchanges an authorization code for access/refresh tokens
func exchangeCodeForTokens(config OAuthConfig, code string) (*OAuthResult, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", config.RedirectURI)
	data.Set("client_id", config.ClientID)
	if config.ClientSecret != "" {
		data.Set("client_secret", config.ClientSecret)
	}

	resp, err := http.PostForm(config.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return parseTokenResponse(resp)
}

// parseTokenResponse parses the token endpoint response
func parseTokenResponse(resp *http.Response) (*OAuthResult, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned status %d", resp.StatusCode)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}

	if err := decodeJSON(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("token error: %s - %s", result.Error, result.ErrorDesc)
	}

	if result.AccessToken == "" {
		return nil, fmt.Errorf("no access token in response")
	}

	return &OAuthResult{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		TokenType:    result.TokenType,
	}, nil
}

// HTML templates for callback responses
func htmlSuccess() string {
	return `<!DOCTYPE html>
<html>
<head>
    <title>Authentication Successful</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
               display: flex; justify-content: center; align-items: center; height: 100vh;
               margin: 0; background: #1a1a2e; color: #eee; }
        .container { text-align: center; padding: 40px; }
        .icon { font-size: 64px; margin-bottom: 20px; }
        h1 { color: #4ade80; margin-bottom: 10px; }
        p { color: #888; }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">✓</div>
        <h1>Authentication Successful</h1>
        <p>You can close this window and return to the terminal.</p>
    </div>
</body>
</html>`
}

func htmlError(message string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Authentication Failed</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
               display: flex; justify-content: center; align-items: center; height: 100vh;
               margin: 0; background: #1a1a2e; color: #eee; }
        .container { text-align: center; padding: 40px; }
        .icon { font-size: 64px; margin-bottom: 20px; }
        h1 { color: #f87171; margin-bottom: 10px; }
        p { color: #888; }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">✗</div>
        <h1>Authentication Failed</h1>
        <p>%s</p>
    </div>
</body>
</html>`, message)
}
