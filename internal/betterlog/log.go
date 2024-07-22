package betterlog

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/liondadev/sx-host/internal/id"
	"github.com/urfave/negroni"
)

const RequestIdKey = "RequestID"

type ContextKey string

func WrapHandler(l *slog.Logger, h http.Handler) http.Handler {

	fn := func(w http.ResponseWriter, r *http.Request) {
		rid := id.New(32)

		uri := r.URL.String()
		method := r.Method

		r = r.WithContext(context.WithValue(r.Context(), RequestIdKey, rid))

		Info(r, "Handling request", "uri", uri, "method", method)

		lgw := negroni.NewResponseWriter(w)
		h.ServeHTTP(lgw, r)

		status := lgw.Status()
		size := lgw.Size()

		Info(r, "Finished handling request", "status", status, "responseSize", size)
	}

	return http.HandlerFunc(fn)
}

func GetRequestId(r *http.Request) (string, error) {
	id := r.Context().Value(RequestIdKey)
	if id == nil {
		return "", errors.New("no request id specified")
	}

	rid, ok := id.(string)
	if !ok {
		return "", errors.New("failed to convert request id value from context into string")
	}

	return rid, nil
}

func Info(r *http.Request, msg string, args ...any) error {
	rid, err := GetRequestId(r)
	if err != nil {
		return err
	}

	slog.Info(msg, append([]any{"requestId", rid}, args...)...)

	return nil
}

func Error(r *http.Request, msg string, args ...any) error {
	rid, err := GetRequestId(r)
	if err != nil {
		return err
	}

	slog.Error(msg, append([]any{"requestId", rid}, args...)...)

	return nil
}
