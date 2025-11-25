package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleAuthHandler struct {
	cfg   *Config
	oauth *oauth2.Config
}

func NewGoogleAuthHandler(cfg *Config) *GoogleAuthHandler {
	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
		return &GoogleAuthHandler{cfg: cfg}
	}
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
	return &GoogleAuthHandler{cfg: cfg, oauth: oauthCfg}
}

func (h *GoogleAuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if h.oauth == nil {
		writeJson(w, http.StatusNotImplemented, map[string]string{"error": "google oauth not configured"})
		return
	}

	state, err := randomState()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})

	url := h.oauth.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusFound)
}

func (h *GoogleAuthHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
	if h.oauth == nil {
		writeJson(w, http.StatusNotImplemented, map[string]string{"error": "google oauth not configured"})
		return
	}

	log.Printf("[auth] google callback hit; accept=%q app_base_url=%q", r.Header.Get("Accept"), h.cfg.AppBaseURL)

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	cookie, _ := r.Cookie("oauth_state")
	if cookie == nil || cookie.Value == "" || cookie.Value != state {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": "invalid oauth state"})
		return
	}

	token, err := h.oauth.Exchange(r.Context(), code)
	if err != nil {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": "token exchange failed"})
		return
	}

	userInfo, err := h.fetchUserInfo(r, token)
	if err != nil {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": "failed to fetch user info"})
		return
	}

	jwtToken, err := h.issueJWT(userInfo.Email)
	if err != nil {
		writeJson(w, http.StatusInternalServerError, map[string]string{"error": "failed to issue jwt"})
		return
	}

	accept := r.Header.Get("Accept")
	// If the caller expects JSON (API tools / curl), return JSON.
	if strings.Contains(accept, "application/json") {
		log.Printf("[auth] returning JSON to caller; user=%s token_len=%d", userInfo.Email, len(jwtToken))
		writeJson(w, http.StatusOK, map[string]any{
			"token": jwtToken,
			"user":  userInfo,
		})
		return
	}

	// Otherwise redirect back to the frontend with the token.
	redirectBase := strings.TrimRight(h.cfg.AppBaseURL, "/")
	if redirectBase == "" {
		redirectBase = "http://localhost:18510"
	}
	u, _ := url.Parse(redirectBase + "/")
	q := u.Query()
	q.Set("token", jwtToken)
	u.RawQuery = q.Encode()

	log.Printf("[auth] redirecting to frontend; user=%s redirect_base=%s token_len=%d", userInfo.Email, redirectBase, len(jwtToken))
	http.Redirect(w, r, u.String(), http.StatusFound)
}

type googleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func (h *GoogleAuthHandler) fetchUserInfo(r *http.Request, token *oauth2.Token) (*googleUserInfo, error) {
	client := h.oauth.Client(r.Context(), token)
	resp, err := client.Get("https://openidconnect.googleapis.com/v1/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("userinfo status %d", resp.StatusCode)
	}
	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (h *GoogleAuthHandler) issueJWT(email string) (string, error) {
	if h.cfg.JwtSecret == "" {
		return "", fmt.Errorf("JWT_SECRET not configured")
	}

	claims := jwt.MapClaims{
		"sub": email,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.cfg.JwtSecret))
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func logIfError(err error) {
	if err != nil {
		log.Println(err)
	}
}
