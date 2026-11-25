package providers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	githubAuthorizeURL = "https://github.com/login/oauth/authorize"
	githubTokenURL     = "https://github.com/login/oauth/access_token"
	githubUserURL      = "https://api.github.com/user"
	githubEmailsURL    = "https://api.github.com/user/emails"
)

type Profile struct {
	Provider          string
	ProviderAccountID string
	Name              string
	Email             string
	AvatarURL         string
	AccessToken       string
	RefreshToken      string
	ExpiresAt         *time.Time
}

type OAuthProvider interface {
	ProviderName() string
	AuthURL(state string) string
	ExchangeUser(ctx context.Context, code string) (*Profile, error)
}

type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	HTTPClient   *http.Client
}

type GitHubProvider struct {
	cfg        GitHubConfig
	httpClient *http.Client
}

type githubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

type githubUserResponse struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

func NewGitHubProvider(cfg GitHubConfig) *GitHubProvider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"read:user", "user:email"}
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	return &GitHubProvider{
		cfg: GitHubConfig{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
		},
		httpClient: client,
	}
}

func (p *GitHubProvider) ProviderName() string {
	return "github"
}

func (p *GitHubProvider) AuthURL(state string) string {
	q := url.Values{}
	q.Set("client_id", p.cfg.ClientID)
	q.Set("redirect_uri", p.cfg.RedirectURL)
	q.Set("state", state)
	if len(p.cfg.Scopes) > 0 {
		q.Set("scope", strings.Join(p.cfg.Scopes, " "))
	}
	return githubAuthorizeURL + "?" + q.Encode()
}

func (p *GitHubProvider) ExchangeUser(ctx context.Context, code string) (*Profile, error) {
	accessToken, err := p.exchangeCode(ctx, code)
	if err != nil {
		return nil, err
	}

	ghUser, err := p.fetchUser(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	name := ghUser.Name
	if name == "" {
		name = ghUser.Login
	}

	return &Profile{
		Provider:          p.ProviderName(),
		ProviderAccountID: strconv.FormatInt(ghUser.ID, 10),
		Name:              name,
		Email:             ghUser.Email,
		AvatarURL:         ghUser.AvatarURL,
		AccessToken:       accessToken,
	}, nil
}

func (p *GitHubProvider) exchangeCode(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("client_id", p.cfg.ClientID)
	form.Set("client_secret", p.cfg.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", p.cfg.RedirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github token status %d: %s", resp.StatusCode, string(body))
	}

	var tokenRes githubTokenResponse
	if err := json.Unmarshal(body, &tokenRes); err != nil {
		return "", err
	}
	if tokenRes.Error != "" {
		return "", fmt.Errorf("%s: %s", tokenRes.Error, tokenRes.ErrorDesc)
	}
	if tokenRes.AccessToken == "" {
		return "", errors.New("empty access_token")
	}

	return tokenRes.AccessToken, nil
}

func (p *GitHubProvider) fetchUser(ctx context.Context, accessToken string) (*githubUserResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubUserURL, nil)
	if err != nil {
		return nil, err
	}
	p.setGitHubHeaders(req, accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github user status %d: %s", resp.StatusCode, string(body))
	}

	var user githubUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	if user.Email == "" {
		if email, err := p.fetchPrimaryEmail(ctx, accessToken); err == nil {
			user.Email = email
		}
	}

	return &user, nil
}

func (p *GitHubProvider) fetchPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubEmailsURL, nil)
	if err != nil {
		return "", err
	}
	p.setGitHubHeaders(req, accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github emails status %d", resp.StatusCode)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	return "", nil
}

func (p *GitHubProvider) setGitHubHeaders(req *http.Request, accessToken string) {
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

func GenerateState(numBytes int) (string, error) {
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
