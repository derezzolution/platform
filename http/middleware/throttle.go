package middleware

import (
	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"

	"log"
	"net/http"
)

// ThrottleHandler controls the number of requests that should be throttled to
// the server.
func ThrottleHandler(h http.Handler) http.Handler {
	throttleStore, err := memstore.New(65536)
	if err != nil {
		log.Fatal(err)
	}

	return throttled.RateLimit(throttled.PerMin(30),
		&throttled.VaryBy{RemoteAddr: true},
		throttleStore).Throttle(h)
}
