package workspacetitles

import (
	"encoding/json"
	"testing"
)

func TestMergeDocListTitles(t *testing.T) {
	raw := json.RawMessage(`{"workspace":{"docs":{"edges":[{"node":{"id":"abc","title":null}},{"node":{"id":"def","title":""}},{"node":{"id":"ghi","title":"Keep"}}]}}}`)
	titles := map[string]string{"abc": "From Yjs", "def": "Also Yjs", "ghi": "Ignored"}

	out := MergeDocListTitles(raw, titles)
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}

	ws := got["workspace"].(map[string]any)
	docs := ws["docs"].(map[string]any)
	edges := docs["edges"].([]any)
	n0 := edges[0].(map[string]any)["node"].(map[string]any)
	n1 := edges[1].(map[string]any)["node"].(map[string]any)
	n2 := edges[2].(map[string]any)["node"].(map[string]any)

	if n0["title"] != "From Yjs" {
		t.Errorf("abc title = %v", n0["title"])
	}
	if n1["title"] != "Also Yjs" {
		t.Errorf("def title = %v", n1["title"])
	}
	if n2["title"] != "Keep" {
		t.Errorf("ghi title should stay Keep, got %v", n2["title"])
	}
}
