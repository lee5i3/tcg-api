package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Minimal API Gateway v2 proxy event/response shapes — just the fields the
// handlers (libs/httpapi) actually read. Kept stdlib-only on purpose.
type proxyEvent struct {
	Version               string            `json:"version"`
	RouteKey              string            `json:"routeKey"`
	RawPath               string            `json:"rawPath"`
	RawQueryString        string            `json:"rawQueryString"`
	Headers               map[string]string `json:"headers"`
	QueryStringParameters map[string]string `json:"queryStringParameters,omitempty"`
	PathParameters        map[string]string `json:"pathParameters,omitempty"`
	Body                  string            `json:"body"`
	IsBase64Encoded       bool              `json:"isBase64Encoded"`
	RequestContext        struct {
		HTTP struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"http"`
	} `json:"requestContext"`
}

type proxyResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

type route struct {
	Pattern  string // doubles as the ServeMux pattern and the RouteKey
	Upstream string // logical upstream name ("game", "set", "card")
}

// paramNames extracts wildcard names from a pattern like
// "GET /v1/games/{game}/sets/{set}".
func paramNames(pattern string) []string {
	var names []string
	for _, seg := range strings.Split(pattern, "/") {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			names = append(names, strings.Trim(seg, "{}"))
		}
	}
	return names
}

// buildEvent wraps an incoming HTTP request into the proxy event the Lambda
// handlers expect.
func buildEvent(r *http.Request, routeKey string, params []string) (proxyEvent, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return proxyEvent{}, err
	}
	ev := proxyEvent{
		Version:        "2.0",
		RouteKey:       routeKey,
		RawPath:        r.URL.Path,
		RawQueryString: r.URL.RawQuery,
		Headers:        map[string]string{},
		Body:           string(body),
	}
	ev.RequestContext.HTTP.Method = r.Method
	ev.RequestContext.HTTP.Path = r.URL.Path
	for name, vals := range r.Header {
		// API Gateway lowercases header names; handlers rely on that.
		ev.Headers[strings.ToLower(name)] = strings.Join(vals, ",")
	}
	if q := r.URL.Query(); len(q) > 0 {
		ev.QueryStringParameters = map[string]string{}
		for k, vals := range q {
			ev.QueryStringParameters[k] = strings.Join(vals, ",")
		}
	}
	if len(params) > 0 {
		ev.PathParameters = map[string]string{}
		for _, p := range params {
			ev.PathParameters[p] = r.PathValue(p)
		}
	}
	return ev, nil
}

// invokeURL is the Runtime Interface Emulator's invoke endpoint.
func invokeURL(upstream string) string {
	return strings.TrimSuffix(upstream, "/") + "/2015-03-31/functions/function/invocations"
}

func newGateway(table []route, upstreams map[string]string) (http.Handler, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	mux := http.NewServeMux()

	for _, rt := range table {
		upstream, ok := upstreams[rt.Upstream]
		if !ok {
			return nil, fmt.Errorf("route %q references unknown upstream %q", rt.Pattern, rt.Upstream)
		}
		routeKey, params, target := rt.Pattern, paramNames(rt.Pattern), invokeURL(upstream)
		mux.HandleFunc(rt.Pattern, func(w http.ResponseWriter, r *http.Request) {
			ev, err := buildEvent(r, routeKey, params)
			if err != nil {
				http.Error(w, `{"error":"bad request body"}`, http.StatusBadRequest)
				return
			}
			payload, err := json.Marshal(ev)
			if err != nil {
				http.Error(w, `{"error":"event encode failed"}`, http.StatusInternalServerError)
				return
			}
			res, err := client.Post(target, "application/json", bytes.NewReader(payload))
			if err != nil {
				http.Error(w, `{"error":"lambda unreachable — is docker compose up?"}`, http.StatusBadGateway)
				return
			}
			defer res.Body.Close()
			var out proxyResponse
			if err := json.NewDecoder(res.Body).Decode(&out); err != nil || out.StatusCode == 0 {
				http.Error(w, `{"error":"lambda returned an unexpected payload"}`, http.StatusBadGateway)
				return
			}
			for k, v := range out.Headers {
				w.Header().Set(k, v)
			}
			w.WriteHeader(out.StatusCode)
			_, _ = io.WriteString(w, out.Body)
		})
	}
	return withCORS(mux), nil
}

// withCORS mirrors the API Gateway CORS config so browser apps on the Vite
// dev servers (different origin) can call the gateway directly.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "authorization, content-type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
