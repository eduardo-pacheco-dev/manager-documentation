package httpapi

import (
	"crypto/tls"
	"net/http/httptest"
	"testing"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/config"
)

func TestSecureCookieFollowsRequestScheme(t *testing.T) {
	enabled := config.Config{SecureCookies: true}
	disabled := config.Config{SecureCookies: false}

	plain := httptest.NewRequest("GET", "http://example.test/", nil)
	if secureCookie(enabled, plain) {
		t.Fatal("cookie must not be Secure on plain HTTP")
	}
	if secureCookie(disabled, plain) {
		t.Fatal("cookie must not be Secure when disabled")
	}

	proxiedTLS := httptest.NewRequest("GET", "http://example.test/", nil)
	proxiedTLS.Header.Set("X-Forwarded-Proto", "https")
	if !secureCookie(enabled, proxiedTLS) {
		t.Fatal("cookie must be Secure when the proxy reports HTTPS")
	}

	directTLS := httptest.NewRequest("GET", "https://example.test/", nil)
	directTLS.TLS = &tls.ConnectionState{}
	if !secureCookie(enabled, directTLS) {
		t.Fatal("cookie must be Secure on direct TLS")
	}
}
