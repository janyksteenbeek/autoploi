package ploi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	baseURL string
	hc      *http.Client
	token   string
}

func NewClient(httpClient *http.Client, token string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{baseURL: "https://ploi.io/api", hc: httpClient, token: token}
}

// urlf builds a full API URL by formatting a path and prefixing it with baseURL.
func (c *Client) urlf(format string, a ...any) string {
	path := fmt.Sprintf(format, a...)
	return strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

// doEnvelope executes the request and decodes the JSON {"data": T} envelope.
func doEnvelope[T any](c *Client, method, url string, body any) (*T, error) {
	resp, err := c.req(method, url, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out envelope[T]
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// doNoContent executes the request where the response body is irrelevant.
func (c *Client) doNoContent(method, url string, body any) error {
	resp, err := c.req(method, url, body)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return nil
}

type Site struct {
	ID       int64  `json:"id"`
	Domain   string `json:"domain"`
	ServerID int64  `json:"server_id"`
}

type envelope[T any] struct {
	Data T `json:"data"`
}

func (c *Client) CreateSite(serverID string, body map[string]any) (*Site, error) {
	url := c.urlf("/servers/%s/sites", serverID)
	return doEnvelope[Site](c, http.MethodPost, url, body)
}

func (c *Client) InstallRepository(serverID string, siteID int64, payload map[string]any) error {
	url := c.urlf("/servers/%s/sites/%d/repository", serverID, siteID)
	return c.doNoContent(http.MethodPost, url, payload)
}

func (c *Client) CreateCertificate(serverID string, siteID int64, payload map[string]any) error {
	url := c.urlf("/servers/%s/sites/%d/certificates", serverID, siteID)
	return c.doNoContent(http.MethodPost, url, payload)
}

func (c *Client) UpdateDeployScript(serverID string, siteID int64, payload map[string]any) error {
	url := c.urlf("/servers/%s/sites/%d/deploy/script", serverID, siteID)
	return c.doNoContent(http.MethodPatch, url, payload)
}

type Database struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func (c *Client) CreateDatabase(serverID string, payload map[string]any) (*Database, error) {
	url := c.urlf("/servers/%s/databases", serverID)
	return doEnvelope[Database](c, http.MethodPost, url, payload)
}

func (c *Client) CreateDatabaseUser(serverID string, databaseID int64, payload map[string]any) error {
	url := c.urlf("/servers/%s/databases/%d/users", serverID, databaseID)
	return c.doNoContent(http.MethodPost, url, payload)
}

func (c *Client) UpdateEnv(serverID string, siteID int64, payload map[string]any) error {
	url := c.urlf("/servers/%s/sites/%d/env", serverID, siteID)
	return c.doNoContent(http.MethodPut, url, payload)
}

func (c *Client) CreateDaemon(serverID string, siteID int64, payload map[string]any) error {
	url := c.urlf("/servers/%s/daemons", serverID)
	return c.doNoContent(http.MethodPost, url, payload)
}

func (c *Client) FindSiteByDomain(serverID, domain string) (*Site, error) {
	q := url.Values{}
	q.Set("filter[domain]", domain)
	endpoint := c.urlf("/servers/%s/sites?%s", serverID, q.Encode())
	resp, err := c.req(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var list envelope[[]Site]
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	for _, s := range list.Data {
		if strings.EqualFold(s.Domain, domain) {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("site not found for domain %s", domain)
}

func (c *Client) DeleteSite(serverID string, siteID int64) error {
	url := c.urlf("/servers/%s/sites/%d", serverID, siteID)
	return c.doNoContent(http.MethodDelete, url, nil)
}

func (c *Client) req(method, url string, body any) (*http.Response, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return resp, nil
}
