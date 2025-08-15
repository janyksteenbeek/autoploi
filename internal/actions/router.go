package actions

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/janyksteenbeek/autoploi/internal/ploi"
)

func Run() error {
	action := strings.ToLower(strings.TrimSpace(os.Getenv("ACTION")))
	if action == "" {
		action = "deploy"
	}
	in := fromEnv()
	if in.PloiToken == "" || in.ServerID == "" {
		return fmt.Errorf("missing required inputs")
	}
	client := ploi.NewClient(http.DefaultClient, in.PloiToken)

	switch action {
	case "deploy":
		if in.Domain == "" {
			return fmt.Errorf("DOMAIN is required for deploy")
		}
		return runDeploy(client, in)
	case "find-site-by-domain":
		if in.Domain == "" {
			return fmt.Errorf("DOMAIN is required for find-site-by-domain")
		}
		s, err := client.FindSiteByDomain(in.ServerID, in.Domain)
		if err != nil {
			return err
		}
		writeOutput("site_id", fmt.Sprintf("%d", s.ID))
		return nil
	case "delete-site":
		siteIDStr := strings.TrimSpace(os.Getenv("SITE_ID"))
		if siteIDStr == "" {
			return fmt.Errorf("SITE_ID is required for delete-site")
		}
		sid, err := strconv.ParseInt(siteIDStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid SITE_ID: %w", err)
		}
		return client.DeleteSite(in.ServerID, sid)
	default:
		return fmt.Errorf("unknown ACTION: %s", action)
	}
}
