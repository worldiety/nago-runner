package apply

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func response(status int, contentType, body string) *http.Response {
	h := http.Header{}
	if contentType != "" {
		h.Set("Content-Type", contentType)
	}
	return &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(strings.NewReader(body))}
}

func TestDecodeConfigurationRejectsErrors(t *testing.T) {
	problem := `{"title":"Forbidden","status":403}`
	cases := []*http.Response{
		response(http.StatusForbidden, "application/problem+json; charset=utf-8", problem),
		response(http.StatusInternalServerError, "application/json", `{"applications":[]}`),
		response(http.StatusBadRequest, "", ""),
		response(http.StatusOK, "application/problem+json", problem),
		response(http.StatusOK, "", ""),
	}

	for i, resp := range cases {
		if _, err := decodeConfiguration(resp); err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
}

func TestDecodeConfigurationAcceptsLegacyHub(t *testing.T) {
	// legacy hubs do not set a content type, thus net/http sniffs text/plain
	cfg, err := decodeConfiguration(response(http.StatusOK, "text/plain; charset=utf-8", `{"applications":[{"instanceId":"abc"}]}`))
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Applications) != 1 || cfg.Applications[0].InstID != "abc" {
		t.Fatalf("unexpected: %+v", cfg)
	}
}
