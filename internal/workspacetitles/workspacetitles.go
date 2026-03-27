// Package workspacetitles loads per-page display titles from AFFiNE workspace Yjs
// snapshots (HTTP /api/workspaces/.../docs/...) and merges them into GraphQL doc list
// responses where GraphQL leaves title null.
package workspacetitles

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tomohiro-owada/affine-cli/internal/yjs"
)

// FetchPageTitles downloads the workspace index and docProperties docs and returns
// page ID → title. cookie and/or bearer are sent like the AFFiNE web app.
func FetchPageTitles(ctx context.Context, baseURL, workspaceID, cookie, bearer string) (map[string]string, error) {
	if cookie == "" && bearer == "" {
		return nil, fmt.Errorf("no session cookie or API token: cannot load workspace Yjs snapshot")
	}
	base := strings.TrimRight(baseURL, "/")
	cli := &http.Client{Timeout: 60 * time.Second}

	wsURL := fmt.Sprintf("%s/api/workspaces/%s/docs/%s", base, workspaceID, workspaceID)
	body, err := getBody(ctx, cli, wsURL, cookie, bearer)
	if err != nil {
		return nil, fmt.Errorf("workspace snapshot: %w", err)
	}

	eng, err := yjs.NewEngine()
	if err != nil {
		return nil, err
	}
	wsDocID, err := eng.ApplyUpdate(body)
	if err != nil {
		return nil, fmt.Errorf("apply workspace yjs: %w", err)
	}

	propsDocID := -1
	propsURL := fmt.Sprintf("%s/api/workspaces/%s/docs/db$%s$docProperties", base, workspaceID, workspaceID)
	propsBody, err := getBody(ctx, cli, propsURL, cookie, bearer)
	if err == nil && len(propsBody) > 0 {
		pid, perr := eng.ApplyUpdate(propsBody)
		if perr == nil {
			propsDocID = pid
		}
	}

	return eng.WorkspacePageTitles(wsDocID, propsDocID)
}

func getBody(ctx context.Context, cli *http.Client, urlStr, cookie, bearer string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("HTTP 404")
	}
	if resp.StatusCode != http.StatusOK {
		slurp, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(slurp)))
	}
	return io.ReadAll(resp.Body)
}

// MergeDocListTitles walks a ListDocs GraphQL JSON payload and fills each node's
// title from titles when GraphQL left it null or empty.
func MergeDocListTitles(data json.RawMessage, titles map[string]string) json.RawMessage {
	if len(titles) == 0 {
		return data
	}
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return data
	}
	walkMerge(root, titles)
	out, err := json.Marshal(root)
	if err != nil {
		return data
	}
	return out
}

func walkMerge(v any, titles map[string]string) {
	switch x := v.(type) {
	case map[string]any:
		if node, ok := x["node"].(map[string]any); ok {
			id, _ := node["id"].(string)
			if id != "" {
				if t, ok := titles[id]; ok && t != "" {
					mergeTitle(node, t)
				}
			}
		}
		for _, child := range x {
			walkMerge(child, titles)
		}
	case []any:
		for _, item := range x {
			walkMerge(item, titles)
		}
	}
}

func mergeTitle(node map[string]any, yjsTitle string) {
	cur := node["title"]
	if cur == nil {
		node["title"] = yjsTitle
		return
	}
	s, ok := cur.(string)
	if !ok || strings.TrimSpace(s) == "" {
		node["title"] = yjsTitle
	}
}
