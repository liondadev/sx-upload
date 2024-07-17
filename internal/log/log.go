package log

import (
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
)

func WrapHandler(l *slog.Logger, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		rid := rand.Intn(99999999)

		uri := r.URL.String()
		method := r.Method

		l.Info(fmt.Sprintf("Handling request %d.", rid), "uri", uri, "method", method)

		h.ServeHTTP(w, r)

		l.Info(fmt.Sprintf("Finished handling request %d.", rid))
	}

	return http.HandlerFunc(fn)
}
