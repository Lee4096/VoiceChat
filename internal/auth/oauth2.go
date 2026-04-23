package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"voicechat/internal/config"
)

type OAuth2Service struct {
	cfg    config.OAuth2Config
	client *http.Client
}

type OAuthUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type GitHubUser struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Login string `json:"login"`
}

type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func NewOAuth2Service(cfg config.OAuth2Config) *OAuth2Service {
	return &OAuth2Service{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *OAuth2Service) GitHubLoginURL() string {
	return fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=read:user,user:email",
		s.cfg.GitHub.ClientID,
		url.QueryEscape(s.cfg.GitHub.CallbackURL),
	)
}

func (s *OAuth2Service) GitHubCallback(ctx context.Context, code string) (*OAuthUser, error) {
	token, err := s.exchangeGitHubToken(ctx, code)
	if err != nil {
		return nil, err
	}

	return s.getGitHubUser(ctx, token)
}

func (s *OAuth2Service) exchangeGitHubToken(ctx context.Context, code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", s.cfg.GitHub.ClientID)
	data.Set("client_secret", s.cfg.GitHub.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", s.cfg.GitHub.CallbackURL)

	resp, err := s.client.PostForm("https://github.com/login/oauth/access_token", data)
	if err != nil {
		return "", fmt.Errorf("failed to exchange token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return values.Get("access_token"), nil
}

func (s *OAuth2Service) getGitHubUser(ctx context.Context, token string) (*OAuthUser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	defer resp.Body.Close()

	var ghUser GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
		return nil, fmt.Errorf("failed to decode user: %w", err)
	}

	name := ghUser.Name
	if name == "" {
		name = ghUser.Login
	}

	return &OAuthUser{
		ID:    fmt.Sprintf("gh_%d", ghUser.ID),
		Email: ghUser.Email,
		Name:  name,
	}, nil
}

func (s *OAuth2Service) GoogleLoginURL() string {
	return fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=email%%20profile",
		s.cfg.Google.ClientID,
		url.QueryEscape(s.cfg.Google.CallbackURL),
	)
}

func (s *OAuth2Service) GoogleCallback(ctx context.Context, code string) (*OAuthUser, error) {
	token, err := s.exchangeGoogleToken(ctx, code)
	if err != nil {
		return nil, err
	}

	return s.getGoogleUser(ctx, token)
}

func (s *OAuth2Service) exchangeGoogleToken(ctx context.Context, code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", s.cfg.Google.ClientID)
	data.Set("client_secret", s.cfg.Google.ClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", s.cfg.Google.CallbackURL)

	resp, err := s.client.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return "", fmt.Errorf("failed to exchange token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.AccessToken, nil
}

func (s *OAuth2Service) getGoogleUser(ctx context.Context, token string) (*OAuthUser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	defer resp.Body.Close()

	var gUser GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return nil, fmt.Errorf("failed to decode user: %w", err)
	}

	return &OAuthUser{
		ID:    "google_" + gUser.ID,
		Email: gUser.Email,
		Name:  gUser.Name,
	}, nil
}

func (s *OAuth2Service) GetProvider(provider string) string {
	provider = strings.ToLower(provider)
	switch provider {
	case "github":
		return "github"
	case "google":
		return "google"
	default:
		return provider
	}
}
