package yandexid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"GolangTemplateProject/internal/config"
	"GolangTemplateProject/internal/ports"
)

const (
	authorizeEndpoint = "https://oauth.yandex.com/authorize"
	tokenEndpoint     = "https://oauth.yandex.com/token"
	userInfoEndpoint  = "https://login.yandex.ru/info?format=json"
)

type Client struct {
	cfg        config.YandexID
	httpClient *http.Client
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

type userInfoResponse struct {
	ID           string   `json:"id"`
	Login        string   `json:"login"`
	DefaultEmail string   `json:"default_email"`
	Emails       []string `json:"emails"`
	FirstName    string   `json:"first_name"`
	LastName     string   `json:"last_name"`
	RealName     string   `json:"real_name"`
}

func New(cfg config.YandexID) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout(),
		},
	}
}

func (c *Client) BuildAuthURL(state string) string {
	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("client_id", c.cfg.ClientID)
	values.Set("redirect_uri", c.cfg.RedirectURI)
	values.Set("state", state)
	if len(c.cfg.Scopes) > 0 {
		values.Set("scope", strings.Join(c.cfg.Scopes, " "))
	}
	return authorizeEndpoint + "?" + values.Encode()
}

func (c *Client) ExchangeCode(ctx context.Context, code string) (*ports.YandexUserProfile, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("yandex token exchange failed with status %d", response.StatusCode)
	}

	var token tokenResponse
	if err := json.NewDecoder(response.Body).Decode(&token); err != nil {
		return nil, err
	}

	return c.fetchUser(ctx, token.AccessToken)
}

func (c *Client) fetchUser(ctx context.Context, accessToken string) (*ports.YandexUserProfile, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoEndpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "OAuth "+accessToken)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("yandex user info failed with status %d", response.StatusCode)
	}

	var user userInfoResponse
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		return nil, err
	}

	email := strings.TrimSpace(user.DefaultEmail)
	if email == "" && len(user.Emails) > 0 {
		email = strings.TrimSpace(user.Emails[0])
	}

	return &ports.YandexUserProfile{
		Subject:   strings.TrimSpace(user.ID),
		Login:     strings.TrimSpace(user.Login),
		Email:     email,
		FirstName: strings.TrimSpace(user.FirstName),
		LastName:  strings.TrimSpace(user.LastName),
	}, nil
}
