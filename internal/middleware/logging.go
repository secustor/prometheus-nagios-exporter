package middleware

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func Logging() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				log.WithFields(log.Fields{
					"event":         "HTTP_REQUEST",
					"method":        r.Method,
					"url":           r.URL,
					"remoteAddress": r.RemoteAddr,
				}).Info()
			}()

			next.ServeHTTP(w, r)
		})
	}
}
