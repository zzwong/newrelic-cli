package api

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type literalTransport func(*http.Request) (*http.Response, error)

func (f literalTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestResolveAppIDNameLiteral(t *testing.T) {
	cases := []struct{ name, escaped string }{
		{"ordinary-app", "ordinary-app"},
		{"O'Brien", `O\'Brien`},
		{`a\b`, `a\\b`},
		{`a\'b`, `a\\\'b`},
		{"x' OR name = 'other", `x\' OR name = \'other`},
		{"日本語 service", "日本語 service"},
		{"", ""},
	}
	guid := base64.StdEncoding.EncodeToString([]byte("2712640|APM|APPLICATION|137708979"))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := &Client{NerdGraphURL: "https://fixture.invalid/graphql", HTTPClient: &http.Client{Transport: literalTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				var req struct {
					Variables map[string]string `json:"variables"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatal(err)
				}
				want := "name = '" + tc.escaped + "' AND domain = 'APM' AND type = 'APPLICATION'"
				if req.Variables["query"] != want {
					t.Errorf("query = %q, want %q", req.Variables["query"], want)
				}
				body := `{"data":{"actor":{"entitySearch":{"results":{"entities":[{"guid":"` + guid + `"}]}}}}}`
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
			})}}
			got, err := c.ResolveAppID(tc.name)
			if err != nil || got != "137708979" || calls != 1 {
				t.Fatalf("got %q, %v, calls %d", got, err, calls)
			}
		})
	}
}

func TestResolveAppIDLiteralFastPaths(t *testing.T) {
	guid := base64.StdEncoding.EncodeToString([]byte("2712640|APM|APPLICATION|137708979"))
	for _, id := range []string{"137708979", guid} {
		t.Run(id, func(t *testing.T) {
			c := &Client{HTTPClient: &http.Client{Transport: literalTransport(func(*http.Request) (*http.Response, error) { t.Fatal("unexpected transport call"); return nil, nil })}}
			got, err := c.ResolveAppID(id)
			if got != "137708979" || err != nil {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
}
