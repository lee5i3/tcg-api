package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeRIE captures the proxy event and answers like a Lambda would.
func fakeRIE(t *testing.T, capture *proxyEvent, respond proxyResponse) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2015-03-31/functions/function/invocations" {
			t.Errorf("invoke path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(capture); err != nil {
			t.Fatalf("decode event: %v", err)
		}
		_ = json.NewEncoder(w).Encode(respond)
	}))
}

func testGateway(t *testing.T, capture *proxyEvent, respond proxyResponse) http.Handler {
	t.Helper()
	rie := fakeRIE(t, capture, respond)
	t.Cleanup(rie.Close)
	gw, err := newGateway(routeTable, map[string]string{
		"game": rie.URL, "set": rie.URL, "card": rie.URL, "auth": rie.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	return gw
}

func TestGatewayTranslatesRequest(t *testing.T) {
	var got proxyEvent
	gw := testGateway(t, &got, proxyResponse{
		StatusCode: 201,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       `{"id":"set-1"}`,
	})

	req := httptest.NewRequest("POST", "/v1/games/pokemon/sets?verbose=1", strings.NewReader(`{"key":"sv3pt5"}`))
	req.Header.Set("Authorization", "Bearer sekrit")
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)

	if got.RouteKey != "POST /v1/games/{game}/sets" {
		t.Errorf("routeKey = %q", got.RouteKey)
	}
	if got.PathParameters["game"] != "pokemon" {
		t.Errorf("pathParameters = %v", got.PathParameters)
	}
	if got.Headers["authorization"] != "Bearer sekrit" {
		t.Errorf("headers = %v (want lowercased authorization)", got.Headers)
	}
	if got.QueryStringParameters["verbose"] != "1" {
		t.Errorf("query = %v", got.QueryStringParameters)
	}
	if got.Body != `{"key":"sv3pt5"}` {
		t.Errorf("body = %q", got.Body)
	}
	if rec.Code != 201 || !strings.Contains(rec.Body.String(), "set-1") {
		t.Errorf("response = %d %q", rec.Code, rec.Body.String())
	}
}

func TestGatewayMultiParamRoute(t *testing.T) {
	var got proxyEvent
	gw := testGateway(t, &got, proxyResponse{StatusCode: 200, Body: `{"cards":[]}`})

	req := httptest.NewRequest("GET", "/v1/games/pokemon/sets/sv3pt5/cards", nil)
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)

	if got.RouteKey != "GET /v1/games/{game}/sets/{set}/cards" {
		t.Errorf("routeKey = %q", got.RouteKey)
	}
	if got.PathParameters["game"] != "pokemon" || got.PathParameters["set"] != "sv3pt5" {
		t.Errorf("pathParameters = %v", got.PathParameters)
	}
}

func TestGatewayUnknownRoute404s(t *testing.T) {
	var got proxyEvent
	gw := testGateway(t, &got, proxyResponse{StatusCode: 200})
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, httptest.NewRequest("GET", "/nope", nil))
	if rec.Code != 404 {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestGatewayCORSPreflight(t *testing.T) {
	var got proxyEvent
	gw := testGateway(t, &got, proxyResponse{StatusCode: 200})
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, httptest.NewRequest("OPTIONS", "/v1/games", nil))
	if rec.Code != 204 {
		t.Errorf("preflight status = %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS headers")
	}
}

func TestGatewayLambdaDown(t *testing.T) {
	gw, err := newGateway(routeTable, map[string]string{
		"game": "http://127.0.0.1:1", "set": "http://127.0.0.1:1", "card": "http://127.0.0.1:1", "auth": "http://127.0.0.1:1",
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/games", nil))
	if rec.Code != 502 {
		t.Errorf("status = %d, want 502", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "docker compose") {
		t.Errorf("body should hint at compose: %s", body)
	}
}

func TestParamNames(t *testing.T) {
	got := paramNames("GET /v1/games/{game}/sets/{set}/cards")
	if len(got) != 2 || got[0] != "game" || got[1] != "set" {
		t.Errorf("paramNames = %v", got)
	}
	if paramNames("GET /healthz") != nil {
		t.Error("no params expected")
	}
}
