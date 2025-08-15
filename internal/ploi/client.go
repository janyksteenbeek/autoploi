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

type Site struct {
	ID       int64  `json:"id"`
	Domain   string `json:"domain"`
	ServerID int64  `json:"server_id"`
}

type envelope[T any] struct {
	Data T `json:"data"`
}

type sitesList struct {
	Data []Site `json:"data"`
}

func (c *Client) CreateSite(serverID string, body map[string]any) (*Site, error) {
	url := fmt.Sprintf("%s/servers/%s/sites", c.baseURL, serverID)
	resp := c.req(http.MethodPost, url, body)
	defer resp.Body.Close()
	var out envelope[Site]
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

func (c *Client) InstallRepository(serverID string, siteID int64, payload map[string]any) error {
	url := fmt.Sprintf("%s/servers/%s/sites/%d/repository", c.baseURL, serverID, siteID)
	resp := c.req(http.MethodPost, url, payload)
	resp.Body.Close()
	return nil
}

func (c *Client) CreateCertificate(serverID string, siteID int64, payload map[string]any) error {
	url := fmt.Sprintf("%s/servers/%s/sites/%d/certificates", c.baseURL, serverID, siteID)
	resp := c.req(http.MethodPost, url, payload)
	resp.Body.Close()
	return nil
}

func (c *Client) UpdateDeployScript(serverID string, siteID int64, payload map[string]any) error {
	url := fmt.Sprintf("%s/servers/%s/sites/%d/deploy-script", c.baseURL, serverID, siteID)
	resp := c.req(http.MethodPut, url, payload)
	resp.Body.Close()
	return nil
}

type Database struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func (c *Client) CreateDatabase(serverID string, payload map[string]any) (*Database, error) {
	url := fmt.Sprintf("%s/servers/%s/databases", c.baseURL, serverID)
	resp := c.req(http.MethodPost, url, payload)
	defer resp.Body.Close()
	var out envelope[Database]
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

func (c *Client) CreateDatabaseUser(serverID string, payload map[string]any) error {
	url := fmt.Sprintf("%s/servers/%s/database-users", c.baseURL, serverID)
	resp := c.req(http.MethodPost, url, payload)
	resp.Body.Close()
	return nil
}

func (c *Client) UpdateEnv(serverID string, siteID int64, payload map[string]any) error {
	url := fmt.Sprintf("%s/servers/%s/sites/%d/env", c.baseURL, serverID, siteID)
	resp := c.req(http.MethodPut, url, payload)
	resp.Body.Close()
	return nil
}

func (c *Client) CreateDaemon(serverID string, siteID int64, payload map[string]any) error {
	url := fmt.Sprintf("%s/servers/%s/sites/%d/daemons", c.baseURL, serverID, siteID)
	resp := c.req(http.MethodPost, url, payload)
	resp.Body.Close()
	return nil
}

func (c *Client) FindSiteByDomain(serverID, domain string) (*Site, error) {
	q := url.Values{}
	q.Set("filter[domain]", domain)
	endpoint := fmt.Sprintf("%s/servers/%s/sites?%s", c.baseURL, serverID, q.Encode())
	resp := c.req(http.MethodGet, endpoint, nil)
	defer resp.Body.Close()
	var list sitesList
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
	url := fmt.Sprintf("%s/servers/%s/sites/%d", c.baseURL, serverID, siteID)
	resp := c.req(http.MethodDelete, url, nil)
	resp.Body.Close()
	return nil
}

func (c *Client) req(method, url string, body any) *http.Response {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		panic(fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b)))
	}
	return resp
}
