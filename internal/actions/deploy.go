package actions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/janyksteenbeek/autoploi/internal/ploi"
)

func runDeploy(client *ploi.Client, in Inputs) error {
	// Create site
	body := map[string]any{"root_domain": in.Domain, "web_directory": in.WebDirectory, "project_root": in.ProjectRoot}
	if strings.TrimSpace(in.ProjectType) != "" {
		body["project_type"] = in.ProjectType
	}
	if strings.TrimSpace(in.SystemUser) != "" {
		body["system_user"] = in.SystemUser
	}
	site, err := client.CreateSite(in.ServerID, body)
	if err != nil {
		return err
	}
	writeOutput("site_id", fmt.Sprintf("%d", site.ID))
	writeOutput("url", fmt.Sprintf("https://%s", in.Domain))

	// Install repository (from GITHUB_REPOSITORY)
	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		return fmt.Errorf("GITHUB_REPOSITORY is required")
	}
	if err := client.InstallRepository(in.ServerID, site.ID, map[string]any{
		"provider": "github",
		"name":     repo,
		"branch":   in.Branch,
	}); err != nil {
		return err
	}

	// SSL certificate (Let's Encrypt)
	if err := client.CreateCertificate(in.ServerID, site.ID, map[string]any{"type": "letsencrypt", "certificate": in.Domain}); err != nil {
		return err
	}

	// Deploy script
	if strings.TrimSpace(in.DeployScript) != "" {
		if err := client.UpdateDeployScript(in.ServerID, site.ID, map[string]any{"content": in.DeployScript}); err != nil {
			return err
		}
	}

	// Optional DB
	if strings.EqualFold(in.CreateDB, "true") || strings.EqualFold(in.CreateDB, "yes") {
		name := in.DBName
		user := in.DBUser
		if name == "" {
			name = sanitizeDBName(in.Domain)
		}
		if user == "" {
			user = name
		}
		pass := randomString(40)
		db, err := client.CreateDatabase(in.ServerID, map[string]any{"name": name})
		if err != nil {
			return err
		}
		payload := map[string]any{"name": user, "password": pass}
		if db != nil && db.ID > 0 {
			payload["databases"] = []int64{db.ID}
		}
		if err := client.CreateDatabaseUser(in.ServerID, payload); err != nil {
			return err
		}
		// DATABASE_URL
		scheme := "mysql"
		port := in.DBPort
		if strings.EqualFold(in.DBEngine, "postgres") || strings.EqualFold(in.DBEngine, "postgresql") {
			scheme = "postgres"
			if port == "" {
				port = "5432"
			}
		} else {
			if port == "" {
				port = "3306"
			}
		}
		dsn := fmt.Sprintf("%s://%s:%s@%s:%s/%s", scheme, user, pass, in.DBHost, port, name)
		if err := client.UpdateEnv(in.ServerID, site.ID, map[string]any{"content": "DATABASE_URL=" + dsn}); err != nil {
			return err
		}
	}

	// Environment
	if strings.TrimSpace(in.Environment) != "" {
		if err := client.UpdateEnv(in.ServerID, site.ID, map[string]any{"content": in.Environment}); err != nil {
			return err
		}
	}

	// Daemons (YAML)
	if strings.TrimSpace(in.DaemonsYAML) != "" {
		cmds, err := parseDaemonsYAMLMinimal(in.DaemonsYAML)
		if err != nil {
			return fmt.Errorf("invalid daemons YAML: %w", err)
		}
		for _, d := range cmds {
			payload := map[string]any{"command": d.Command}
			if d.Path != "" {
				payload["path"] = d.Path
			}
			if err := client.CreateDaemon(in.ServerID, site.ID, payload); err != nil {
				return err
			}
		}
	}

	// PR comment
	if isPR() && strings.TrimSpace(in.GithubToken) != "" {
		if err := commentPR(in.GithubToken, fmt.Sprintf("https://%s", in.Domain)); err != nil {
			return err
		}
	}
	return nil
}

// YAML parsing (minimal): supports arrays of scalars or maps with command/path.
type daemonSpec struct {
	Command string
	Path    string
}

func parseDaemonsYAMLMinimal(s string) ([]daemonSpec, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	if !strings.HasPrefix(s, "-") {
		return nil, errors.New("expected YAML array starting with '-'")
	}
	lines := strings.Split(s, "\n")
	var out []daemonSpec
	var cur daemonSpec
	inMap := false
	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "-") {
			// new item
			if inMap {
				out = append(out, cur)
				cur = daemonSpec{}
				inMap = false
			}
			entry := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
			if entry == "" { // object will follow in next indented lines
				inMap = true
				continue
			}
			// scalar command in same line
			out = append(out, daemonSpec{Command: entry})
			continue
		}
		// indented key: value under current map
		if !inMap {
			return nil, errors.New("unexpected mapping line without item prefix")
		}
		kv := strings.SplitN(strings.TrimSpace(line), ":", 2)
		if len(kv) != 2 {
			return nil, errors.New("invalid mapping line")
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		val = strings.TrimLeft(val, " ")
		if key == "command" {
			cur.Command = val
		}
		if key == "path" {
			cur.Path = val
		}
	}
	if inMap {
		out = append(out, cur)
	}
	// trim items
	for i := range out {
		out[i].Command = strings.Trim(out[i].Command, " ")
		out[i].Path = strings.Trim(out[i].Path, " ")
	}
	return out, nil
}

func isPR() bool {
	name := os.Getenv("GITHUB_EVENT_NAME")
	return name == "pull_request" || name == "pull_request_target"
}

func commentPR(token, url string) error {
	repo := os.Getenv("GITHUB_REPOSITORY")
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if repo == "" || eventPath == "" {
		return nil
	}
	f, err := os.Open(eventPath)
	if err != nil {
		return nil
	}
	defer f.Close()
	var payload map[string]any
	_ = json.NewDecoder(f).Decode(&payload)
	pr, _ := payload["pull_request"].(map[string]any)
	if pr == nil {
		return nil
	}
	numF64, _ := pr["number"].(float64)
	if numF64 == 0 {
		return nil
	}
	num := int(numF64)
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d/comments", repo, num)
	b, _ := json.Marshal(map[string]string{"body": fmt.Sprintf("Preview: %s", url)})
	req, _ := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	return nil
}

// util
func randomString(n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{}|;:,.<>?"
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = alphabet[i%len(alphabet)]
	}
	return string(b)
}

func sanitizeDBName(domain string) string {
	s := strings.ToLower(domain)
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	for i := 0; i < len(s); i++ {
		if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= '0' && s[i] <= '9') || s[i] == '_' {
			continue
		}
		s = s[:i] + "_" + s[i+1:]
	}
	if len(s) > 48 {
		s = s[:48]
	}
	return s
}

func writeOutput(key, value string) {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s=%s\n", key, value)
}
