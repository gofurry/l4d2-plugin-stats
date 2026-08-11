package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
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

func TestSteamOpenIDHTTPClientUsesConfiguredLoopbackProxy(t *testing.T) {
	proxied := false
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxied = true
		if r.URL.String() != "http://steam.invalid/openid/login" {
			t.Fatalf("proxy request URL=%q", r.URL.String())
		}
		_, _ = io.WriteString(w, "is_valid:true\n")
	}))
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.ParseInt(proxyURL.Port(), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	client, err := steamOpenIDHTTPClient(port)
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

func TestSteamOpenIDHTTPClientRejectsInvalidProxyPort(t *testing.T) {
	for _, port := range []int64{-1, 65536} {
		if _, err := steamOpenIDHTTPClient(port); err == nil {
			t.Fatalf("proxy port %d should fail", port)
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
