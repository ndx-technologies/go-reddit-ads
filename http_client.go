package goredditads

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nikolaydubina/fpmoney"
)

const (
	defaultBaseURL = "https://ads-api.reddit.com/api/v3"
	tokenURL       = "https://www.reddit.com/api/v1/access_token"
	authorizeURL   = "https://www.reddit.com/api/v1/authorize"
)

type ErrHTTP struct {
	StatusCode int
	Body       string
}

func (e ErrHTTP) Error() string { return "http: " + strconv.Itoa(e.StatusCode) + ": " + e.Body }

type RedditHTTPClientSecrets struct {
	AppSecret   string `json:"app_secret"`
	AppAuthCode string `json:"app_auth_code"`
	AppToken    string `json:"app_token,omitempty"`
}

func (s RedditHTTPClientSecrets) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.MarshalWrite(f, s)
}

type RedditHTTPClientConfig struct {
	AppID       string
	AdAccountID string
	BaseURL     string
}

func (s RedditHTTPClientConfig) WithDefaults() RedditHTTPClientConfig {
	if s.BaseURL == "" {
		s.BaseURL = defaultBaseURL
	}
	return s
}

// RedditHTTPClient follows https://ads-api.reddit.com/docs/v3
type RedditHTTPClient struct {
	Config     RedditHTTPClientConfig
	Secrets    RedditHTTPClientSecrets
	HTTPClient *http.Client
}

func (s RedditHTTPClient) AuthorizeURL(redirectURI, state string) string {
	query := make(url.Values)
	query.Set("client_id", s.Config.AppID)
	query.Set("response_type", "code")
	query.Set("state", state)
	query.Set("redirect_uri", redirectURI)
	query.Set("duration", "permanent")
	query.Set("scope", "adsread adsedit read")

	return authorizeURL + "?" + query.Encode()
}

func (s RedditHTTPClient) FetchAccessTokenWithCode(ctx context.Context, code, redirectURI string) (string, error) {
	form := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {redirectURI},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(s.Config.AppID, s.Secrets.AppSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "server:github.com/ndx-technologies/marketing:v1.0.0 (by /u/ndxai)")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", ErrHTTP{StatusCode: resp.StatusCode, Body: string(body)}
	}

	var res struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.UnmarshalRead(resp.Body, &res); err != nil {
		return "", err
	}
	return res.AccessToken, nil
}

type pagination struct {
	NextURL string `json:"next_url"`
}

type listResponse[T any] struct {
	Data       []T        `json:"data"`
	Pagination pagination `json:"pagination"`
}

func (s RedditHTTPClient) initRequest(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+s.Secrets.AppToken)
	req.Header.Set("Content-Type", "application/json")
}

func (s RedditHTTPClient) FetchCampaigns(ctx context.Context) ([]Campaign, error) {
	url := s.Config.BaseURL + "/ad_accounts/" + s.Config.AdAccountID + "/campaigns"
	var campaigns []Campaign
	for url != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		s.initRequest(req)
		resp, err := s.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, ErrHTTP{StatusCode: resp.StatusCode, Body: string(body)}
		}
		var result listResponse[Campaign]
		if err := json.UnmarshalRead(resp.Body, &result); err != nil {
			return nil, err
		}
		campaigns = append(campaigns, result.Data...)
		url = result.Pagination.NextURL
	}
	return campaigns, nil
}

func (s RedditHTTPClient) FetchAdGroups(ctx context.Context) ([]AdGroup, error) {
	url := s.Config.BaseURL + "/ad_accounts/" + s.Config.AdAccountID + "/ad_groups"
	var adGroups []AdGroup
	for url != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		s.initRequest(req)
		resp, err := s.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, ErrHTTP{StatusCode: resp.StatusCode, Body: string(body)}
		}
		var result listResponse[AdGroup]
		if err := json.UnmarshalRead(resp.Body, &result); err != nil {
			return nil, err
		}
		adGroups = append(adGroups, result.Data...)
		url = result.Pagination.NextURL
	}
	return adGroups, nil
}

func (s RedditHTTPClient) FetchAds(ctx context.Context) ([]Ad, error) {
	url := s.Config.BaseURL + "/ad_accounts/" + s.Config.AdAccountID + "/ads"
	var ads []Ad
	for url != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		s.initRequest(req)
		resp, err := s.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, ErrHTTP{StatusCode: resp.StatusCode, Body: string(body)}
		}
		var result listResponse[Ad]
		if err := json.UnmarshalRead(resp.Body, &result); err != nil {
			return nil, err
		}
		ads = append(ads, result.Data...)
		url = result.Pagination.NextURL
	}
	return ads, nil
}

type dataRequest[T any] struct {
	Data T `json:"data"`
}

func (s RedditHTTPClient) UpdateCampaign(ctx context.Context, c Campaign) error {
	url := s.Config.BaseURL + "/campaigns/" + string(c.ID)
	b, err := json.Marshal(dataRequest[Campaign]{Data: c})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	s.initRequest(req)
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return ErrHTTP{StatusCode: resp.StatusCode, Body: string(body)}
	}
	return nil
}

func (s RedditHTTPClient) UpdateAdGroup(ctx context.Context, ag AdGroup) error {
	url := s.Config.BaseURL + "/ad_groups/" + string(ag.ID)
	b, err := json.Marshal(dataRequest[AdGroup]{Data: ag})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	s.initRequest(req)
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return ErrHTTP{StatusCode: resp.StatusCode, Body: string(body)}
	}
	return nil
}

func (s RedditHTTPClient) UpdateAd(ctx context.Context, a Ad) error {
	url := s.Config.BaseURL + "/ads/" + string(a.ID)
	b, err := json.Marshal(dataRequest[Ad]{Data: a})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	s.initRequest(req)
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return ErrHTTP{StatusCode: resp.StatusCode, Body: string(body)}
	}
	return nil
}

type AdAccount struct {
	Currency            fpmoney.Currency `json:"currency"`
	ExcludedCommunities []string         `json:"excluded_communities,omitempty"`
	ExcludedKeywords    []string         `json:"excluded_keywords,omitempty"`
}

func (s RedditHTTPClient) FetchAdAccount(ctx context.Context) (AdAccount, error) {
	url := s.Config.BaseURL + "/ad_accounts/" + s.Config.AdAccountID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return AdAccount{}, err
	}
	s.initRequest(req)
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return AdAccount{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return AdAccount{}, ErrHTTP{StatusCode: resp.StatusCode, Body: string(body)}
	}
	var result struct {
		Data AdAccount `json:"data"`
	}
	if err := json.UnmarshalRead(resp.Body, &result); err != nil {
		return result.Data, err
	}
	return result.Data, nil
}

func (s RedditHTTPClient) FetchPost(ctx context.Context, postID PostID) (Post, error) {
	url := s.Config.BaseURL + "/posts/" + postID.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Post{}, err
	}
	s.initRequest(req)
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return Post{}, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Post{}, ErrHTTP{StatusCode: resp.StatusCode, Body: string(body)}
	}
	var result struct {
		Data Post `json:"data"`
	}
	if err := json.UnmarshalRead(resp.Body, &result); err != nil {
		return Post{}, err
	}
	return result.Data, nil
}

func (s RedditHTTPClient) FetchReport(ctx context.Context, from, until time.Time, breakdowns []ReportBreakdown, fields []ReportField) ([]ReportMetric, error) {
	type reportRequest struct {
		StartsAt   string            `json:"starts_at"`
		EndsAt     string            `json:"ends_at"`
		Breakdowns []ReportBreakdown `json:"breakdowns"`
		Fields     []ReportField     `json:"fields"`
	}
	type reportData struct {
		Metrics []ReportMetric `json:"metrics"`
	}
	type reportResponse struct {
		Data       reportData `json:"data"`
		Pagination pagination `json:"pagination"`
	}

	nextURL := s.Config.BaseURL + "/ad_accounts/" + s.Config.AdAccountID + "/reports"
	var all []ReportMetric
	for nextURL != "" {
		req := reportRequest{
			StartsAt:   from.UTC().Format("2006-01-02T00:00:00Z"),
			EndsAt:     until.UTC().Add(24 * time.Hour).Format("2006-01-02T00:00:00Z"),
			Breakdowns: breakdowns,
			Fields:     fields,
		}
		b, err := json.Marshal(dataRequest[reportRequest]{Data: req})
		if err != nil {
			return nil, err
		}
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, nextURL, bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		s.initRequest(httpReq)
		resp, err := s.HTTPClient.Do(httpReq)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, ErrHTTP{StatusCode: resp.StatusCode, Body: string(body)}
		}
		var result reportResponse
		if err := json.UnmarshalRead(resp.Body, &result); err != nil {
			return nil, err
		}
		all = append(all, result.Data.Metrics...)
		nextURL = result.Pagination.NextURL
	}
	return all, nil
}
