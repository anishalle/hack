package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/anishalle/hack/internal/models"
	"github.com/anishalle/hack/server/internal/middleware"
	"github.com/anishalle/hack/server/internal/store"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthHandler struct {
	userStore    *store.UserStore
	oauthConfig  *oauth2.Config
	pendingCodes sync.Map // device_code -> *pendingAuth
}

type pendingAuth struct {
	DeviceCode string
	UserCode   string
	ExpiresAt  time.Time
	Token      *oauth2.Token
	Email      string
	Name       string
	Completed  bool
	Error      string
}

func NewAuthHandler(userStore *store.UserStore) *AuthHandler {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	oauthConfig := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes: []string{
			"openid",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		RedirectURL: "urn:ietf:wg:oauth:2.0:oob",
	}

	return &AuthHandler{
		userStore:   userStore,
		oauthConfig: oauthConfig,
	}
}

func (h *AuthHandler) DeviceCode(w http.ResponseWriter, r *http.Request) {
	deviceCode, err := generateCode(32)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate device code")
		return
	}

	userCode, err := generateUserCode()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate user code")
		return
	}

	pending := &pendingAuth{
		DeviceCode: deviceCode,
		UserCode:   userCode,
		ExpiresAt:  time.Now().Add(15 * time.Minute),
	}

	h.pendingCodes.Store(deviceCode, pending)
	h.pendingCodes.Store("user:"+userCode, pending)

	authURL := h.oauthConfig.AuthCodeURL(deviceCode, oauth2.AccessTypeOffline)

	respondJSON(w, http.StatusOK, map[string]any{
		"device_code":      deviceCode,
		"user_code":        userCode,
		"verification_url": authURL,
		"expires_in":       900,
		"interval":         5,
	})
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		respondError(w, http.StatusBadRequest, "missing code or state")
		return
	}

	val, ok := h.pendingCodes.Load(state)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid or expired device code")
		return
	}
	pending := val.(*pendingAuth)

	if time.Now().After(pending.ExpiresAt) {
		respondError(w, http.StatusGone, "device code expired")
		return
	}

	token, err := h.oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		slog.Error("oauth exchange failed", "error", err)
		pending.Error = "authentication failed"
		respondError(w, http.StatusInternalServerError, "failed to exchange authorization code")
		return
	}

	userInfo, err := fetchGoogleUserInfo(r.Context(), token.AccessToken)
	if err != nil {
		slog.Error("failed to fetch user info", "error", err)
		pending.Error = "failed to fetch user info"
		respondError(w, http.StatusInternalServerError, "failed to get user info")
		return
	}

	pending.Token = token
	pending.Email = userInfo.Email
	pending.Name = userInfo.Name
	pending.Completed = true

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html><body>
<h1>Authentication Successful</h1>
<p>You can close this window and return to the terminal.</p>
<script>window.close();</script>
</body></html>`)
}

func (h *AuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceCode string `json:"device_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	val, ok := h.pendingCodes.Load(req.DeviceCode)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid device code")
		return
	}
	pending := val.(*pendingAuth)

	if time.Now().After(pending.ExpiresAt) {
		h.pendingCodes.Delete(req.DeviceCode)
		h.pendingCodes.Delete("user:" + pending.UserCode)
		respondError(w, http.StatusGone, "device code expired")
		return
	}

	if pending.Error != "" {
		respondError(w, http.StatusInternalServerError, pending.Error)
		return
	}

	if !pending.Completed {
		respondJSON(w, http.StatusAccepted, map[string]any{
			"status": "authorization_pending",
		})
		return
	}

	user, err := h.userStore.GetByEmail(r.Context(), pending.Email)
	if err != nil {
		user = &models.User{
			Email:    pending.Email,
			Name:     pending.Name,
			Projects: make(map[string]string),
		}
		if err := h.userStore.Create(r.Context(), user); err != nil {
			slog.Error("failed to create user", "error", err, "email", pending.Email)
			respondError(w, http.StatusInternalServerError, "failed to create user account")
			return
		}
	}

	accessToken, err := middleware.GenerateToken(user.Email, user.Name, 24*time.Hour)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}

	refreshToken, err := middleware.GenerateRefreshToken(user.Email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate refresh token")
		return
	}

	h.pendingCodes.Delete(req.DeviceCode)
	h.pendingCodes.Delete("user:" + pending.UserCode)

	respondJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    86400,
		"email":         user.Email,
		"name":          user.Name,
		"token_type":    "Bearer",
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	claims, err := middleware.ValidateToken(req.RefreshToken)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	if claims.Issuer != "hack-refresh" {
		respondError(w, http.StatusUnauthorized, "invalid token type")
		return
	}

	user, err := h.userStore.GetByEmail(r.Context(), claims.Email)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "user not found")
		return
	}

	accessToken, err := middleware.GenerateToken(user.Email, user.Name, 24*time.Hour)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"access_token": accessToken,
		"expires_in":   86400,
		"email":        user.Email,
		"token_type":   "Bearer",
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"email":    user.Email,
		"name":     user.Name,
		"projects": user.Projects,
	})
}

type googleUserInfo struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func fetchGoogleUserInfo(ctx context.Context, accessToken string) (*googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return &info, nil
}

func generateCode(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generateUserCode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := fmt.Sprintf("HACK-%s", hex.EncodeToString(b)[:6])
	return code, nil
}
