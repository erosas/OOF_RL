package core

import (
	"net"
	"net/http"
	"net/url"

	"OOF_RL/internal/httputil"
)

// LocalhostGuard rejects requests that did not originate from this machine's
// browser or webview, even though the listener is already bound to 127.0.0.1:
//
//   - Host header must be a loopback name. A remote website can reach the
//     listener via DNS rebinding (its hostname re-resolves to 127.0.0.1 after
//     the page loads), gaining full same-origin access to the API. Such
//     requests carry the attacker's hostname in Host.
//   - State-changing requests with an explicit non-loopback Origin are
//     rejected. Browsers send cross-origin POSTs without a CORS preflight for
//     "simple" content types, and the JSON decoders here ignore Content-Type,
//     so any website could otherwise trigger /api/config, /api/settings, and
//     plugin POST routes. An absent Origin (webview, curl) is allowed.
func LocalhostGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackHost(r.Host) {
			httputil.JSONError(w, http.StatusForbidden, "forbidden host")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			if origin := r.Header.Get("Origin"); origin != "" && !isLoopbackOrigin(origin) {
				httputil.JSONError(w, http.StatusForbidden, "forbidden origin")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func isLoopbackHost(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport // no port present
	}
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func isLoopbackOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	h := u.Hostname()
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}