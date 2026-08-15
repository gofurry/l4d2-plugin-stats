package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func TestOpenIDVerifierAcceptsOfficialHTTPClaimedIDAfterProviderVerification(t *testing.T) {
	var posted url.Values
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		posted, err = url.ParseQuery(string(body))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.WriteString(w, "ns:http://specs.openid.net/auth/2.0\nis_valid:true\n")
	}))
	defer provider.Close()

	settings := store.SiteSettings{PublicOrigin: "https://stats.example.com"}
	verifier, err := newOpenIDVerifier(settings, provider.URL, provider.Client())
	if err != nil {
		t.Fatal(err)
	}
	values := callbackValues(provider.URL, "http://steamcommunity.com/openid/id/76561198000000000")
	identity, err := verifier.VerifyValues(context.Background(), values)
	if err != nil {
		t.Fatal(err)
	}
	if identity.SteamID != "76561198000000000" || identity.State != "test-state" {
		t.Fatalf("identity=%+v", identity)
	}
	if posted.Get("openid.mode") != "check_authentication" || posted.Get("openid.claimed_id") != values.Get("openid.claimed_id") {
		t.Fatalf("verification payload=%v", posted)
	}
}

func TestOpenIDVerifierRejectsUnsignedHTTPClaimedID(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "is_valid:true\n")
	}))
	defer provider.Close()
	verifier, err := newOpenIDVerifier(store.SiteSettings{PublicOrigin: "https://stats.example.com"}, provider.URL, provider.Client())
	if err != nil {
		t.Fatal(err)
	}
	values := callbackValues(provider.URL, "http://steamcommunity.com/openid/id/76561198000000000")
	values.Set("openid.signed", "op_endpoint,identity,return_to")
	if _, err := verifier.VerifyValues(context.Background(), values); err == nil || !strings.Contains(err.Error(), "claimed_id is not signed") {
		t.Fatalf("error=%v", err)
	}
}

func TestOpenIDVerifierRejectsProviderInvalidHTTPClaimedID(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "is_valid:false\n")
	}))
	defer provider.Close()
	verifier, err := newOpenIDVerifier(store.SiteSettings{PublicOrigin: "https://stats.example.com"}, provider.URL, provider.Client())
	if err != nil {
		t.Fatal(err)
	}
	values := callbackValues(provider.URL, "http://steamcommunity.com/openid/id/76561198000000000")
	if _, err := verifier.VerifyValues(context.Background(), values); err == nil || !strings.Contains(err.Error(), "provider rejected") {
		t.Fatalf("error=%v", err)
	}
}

func TestSteamOpenIDHTTPClientUsesConfiguredProxyURL(t *testing.T) {
	proxied := false
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxied = true
		if r.URL.String() != "http://steam.invalid/openid/login" {
			t.Fatalf("proxy request URL=%q", r.URL.String())
		}
		_, _ = io.WriteString(w, "is_valid:true\n")
	}))
	defer proxy.Close()
	client, err := steamOpenIDHTTPClient(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}
	settings := store.SiteSettings{PublicOrigin: "https://stats.example.com"}
	verifier, err := newOpenIDVerifier(settings, "http://steam.invalid/openid/login", client)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := verifier.VerifyValues(context.Background(), callbackValues("http://steam.invalid/openid/login", "http://steamcommunity.com/openid/id/76561198000000000"))
	if err != nil {
		t.Fatal(err)
	}
	if !proxied || identity.SteamID != "76561198000000000" {
		t.Fatalf("proxied=%t identity=%+v", proxied, identity)
	}
}

func TestNormalizeSteamOpenIDProxyURL(t *testing.T) {
	valid := map[string]string{
		"":                              "",
		"127.0.0.1:7890":                "http://127.0.0.1:7890",
		"http://proxy.example.com:8080": "http://proxy.example.com:8080",
		"https://user:pass@proxy.example.com:8443": "https://user:pass@proxy.example.com:8443",
		"socks5://10.0.0.8:1080":                   "socks5://10.0.0.8:1080",
	}
	for input, want := range valid {
		got, _, err := normalizeSteamOpenIDProxyURL(input)
		if err != nil || got != want {
			t.Fatalf("normalize %q=%q err=%v, want %q", input, got, err, want)
		}
	}
	for _, input := range []string{"ftp://proxy.example.com:21", "http://", "http://proxy.example.com/path", "http://proxy.example.com?x=1", "http://proxy.example.com#x", strings.Repeat("a", 2049)} {
		if _, _, err := normalizeSteamOpenIDProxyURL(input); err == nil {
			t.Fatalf("proxy URL %q should fail", input)
		}
	}
}

func TestSafeSteamReturnTo(t *testing.T) {
	valid := map[string]string{
		"":                         "/player",
		"/player":                  "/player",
		"/player?tab=achievements": "/player?tab=achievements",
	}
	for input, want := range valid {
		got, ok := safeSteamReturnTo(input)
		if !ok || got != want {
			t.Fatalf("safeSteamReturnTo(%q)=%q,%t want %q,true", input, got, ok, want)
		}
	}
	for _, input := range []string{"https://evil.example/player", "//evil.example/player", "/admin", "/player#fragment", "player"} {
		if got, ok := safeSteamReturnTo(input); ok {
			t.Fatalf("unsafe return %q accepted as %q", input, got)
		}
	}
}

func callbackValues(endpoint, claimedID string) url.Values {
	return url.Values{
		"openid.ns":          {"http://specs.openid.net/auth/2.0"},
		"openid.mode":        {"id_res"},
		"openid.op_endpoint": {endpoint},
		"openid.claimed_id":  {claimedID},
		"openid.identity":    {claimedID},
		"openid.return_to":   {"https://stats.example.com/api/v1/steam/callback?state=test-state"},
		"openid.signed":      {"op_endpoint,claimed_id,identity,return_to,response_nonce,assoc_handle"},
		"openid.sig":         {"signed-by-provider"},
	}
}
