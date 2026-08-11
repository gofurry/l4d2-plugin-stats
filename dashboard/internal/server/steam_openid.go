package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	steamopenid "github.com/gofurry/steam-go/addons/openid"
)

const steamOpenIDEndpoint = "https://steamcommunity.com/openid/login"

var openIDSteamID64Pattern = regexp.MustCompile(`^[0-9]{17}$`)

type openIDVerifier struct {
	primary  *steamopenid.Verifier
	returnTo *url.URL
	endpoint *url.URL
	client   *http.Client
}

func openidVerifier(settings store.SiteSettings) (*openIDVerifier, error) {
	return newOpenIDVerifier(settings, steamOpenIDEndpoint, http.DefaultClient)
}

func newOpenIDVerifier(settings store.SiteSettings, endpoint string, client *http.Client) (*openIDVerifier, error) {
	origin := strings.TrimSuffix(settings.PublicOrigin, "/")
	returnTo, err := url.Parse(origin + "/api/v1/steam/callback")
	if err != nil {
		return nil, fmt.Errorf("parse Steam OpenID return_to: %w", err)
	}
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse Steam OpenID endpoint: %w", err)
	}
	primary, err := steamopenid.NewVerifier(
		steamopenid.Config{Realm: origin + "/", ReturnTo: returnTo.String()},
		steamopenid.WithEndpoint(endpoint),
		steamopenid.WithHTTPClient(client),
	)
	if err != nil {
		return nil, err
	}
	return &openIDVerifier{primary: primary, returnTo: returnTo, endpoint: endpointURL, client: client}, nil
}

func (v *openIDVerifier) LoginURL(state string) (string, error) {
	return v.primary.LoginURL(state)
}

func (v *openIDVerifier) VerifyValues(ctx context.Context, values url.Values) (*steamopenid.Identity, error) {
	claimedID, err := url.Parse(values.Get("openid.claimed_id"))
	if err != nil || !strings.EqualFold(claimedID.Scheme, "http") {
		return v.primary.VerifyValues(ctx, values)
	}
	return v.verifyOfficialHTTPClaimedID(ctx, values, claimedID)
}

// Steamworks documents the claimed ID as an HTTP URL even though the OpenID
// provider endpoint itself is HTTPS. The claimed ID is signed data and is never
// requested by the dashboard, so accepting this official form does not create
// a plaintext network request or skip provider verification.
func (v *openIDVerifier) verifyOfficialHTTPClaimedID(ctx context.Context, values url.Values, claimedID *url.URL) (*steamopenid.Identity, error) {
	if values.Get("openid.mode") != "id_res" {
		return nil, fmt.Errorf("unexpected openid.mode %q", values.Get("openid.mode"))
	}
	if values.Get("openid.identity") != values.Get("openid.claimed_id") {
		return nil, fmt.Errorf("openid.identity does not match openid.claimed_id")
	}
	if err := validateSteamClaimedID(claimedID); err != nil {
		return nil, err
	}
	if !sameOpenIDEndpoint(values.Get("openid.op_endpoint"), v.endpoint) {
		return nil, fmt.Errorf("unexpected openid.op_endpoint")
	}
	if err := requireSignedOpenIDFields(values.Get("openid.signed")); err != nil {
		return nil, err
	}
	state, err := v.verifyReturnTo(values.Get("openid.return_to"))
	if err != nil {
		return nil, err
	}

	checkValues := cloneURLValues(values)
	checkValues.Set("openid.mode", "check_authentication")
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(checkCtx, http.MethodPost, v.endpoint.String(), strings.NewReader(checkValues.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build Steam OpenID verification request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("verify Steam OpenID response: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, (64<<10)+1))
	if err != nil {
		return nil, fmt.Errorf("read Steam OpenID verification response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Steam OpenID verification returned HTTP %d", resp.StatusCode)
	}
	if len(body) > 64<<10 || !openIDProviderAccepted(string(body)) {
		return nil, fmt.Errorf("Steam OpenID provider rejected authentication response")
	}

	return &steamopenid.Identity{
		SteamID:   strings.TrimPrefix(claimedID.Path, "/openid/id/"),
		ClaimedID: claimedID.String(),
		State:     state,
	}, nil
}

func validateSteamClaimedID(claimedID *url.URL) error {
	if claimedID == nil || !strings.EqualFold(claimedID.Scheme, "http") || !strings.EqualFold(claimedID.Host, "steamcommunity.com") || claimedID.User != nil || claimedID.RawQuery != "" || claimedID.Fragment != "" {
		return fmt.Errorf("invalid Steam claimed_id")
	}
	steamID := strings.TrimPrefix(claimedID.Path, "/openid/id/")
	if !openIDSteamID64Pattern.MatchString(steamID) || claimedID.Path != "/openid/id/"+steamID {
		return fmt.Errorf("invalid SteamID64 in claimed_id")
	}
	return nil
}

func sameOpenIDEndpoint(raw string, expected *url.URL) bool {
	actual, err := url.Parse(raw)
	if err != nil || actual.Scheme != expected.Scheme || !strings.EqualFold(actual.Host, expected.Host) || actual.User != nil || actual.RawQuery != "" || actual.Fragment != "" {
		return false
	}
	return strings.TrimSuffix(actual.Path, "/") == strings.TrimSuffix(expected.Path, "/")
}

func requireSignedOpenIDFields(raw string) error {
	signed := make(map[string]bool)
	for _, field := range strings.Split(raw, ",") {
		signed[strings.TrimSpace(field)] = true
	}
	for _, required := range []string{"op_endpoint", "claimed_id", "identity", "return_to"} {
		if !signed[required] {
			return fmt.Errorf("openid.%s is not signed", required)
		}
	}
	return nil
}

func (v *openIDVerifier) verifyReturnTo(raw string) (string, error) {
	actual, err := url.Parse(raw)
	if err != nil || actual.Scheme == "" || actual.Host == "" {
		return "", fmt.Errorf("invalid openid.return_to")
	}
	state := actual.Query().Get("state")
	if state == "" {
		return "", fmt.Errorf("openid.return_to has no state")
	}
	expected := *v.returnTo
	query := expected.Query()
	query.Set("state", state)
	expected.RawQuery = query.Encode()
	if actual.String() != expected.String() {
		return "", fmt.Errorf("openid.return_to mismatch")
	}
	return state, nil
}

func cloneURLValues(values url.Values) url.Values {
	cloned := make(url.Values, len(values))
	for key, items := range values {
		cloned[key] = append([]string(nil), items...)
	}
	return cloned
}

func openIDProviderAccepted(body string) bool {
	for _, line := range strings.Split(body, "\n") {
		if strings.EqualFold(strings.TrimSpace(line), "is_valid:true") {
			return true
		}
	}
	return false
}
