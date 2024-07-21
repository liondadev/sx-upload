package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/liondadev/sx-host/internal/config"
	"github.com/liondadev/sx-host/internal/log"
)

type Server struct {
	db   *sqlx.DB
	mux  *http.ServeMux
	conf *config.Config
}

func NewServer(db *sqlx.DB, conf *config.Config) *Server {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: false}))

	mux := http.NewServeMux()

	mux.Handle("/upload", log.WrapHandler(logger, &uploadHandler{db: db, conf: conf}))
	mux.Handle("/f/", log.WrapHandler(logger, &viewHandler{db: db, conf: conf}))
	mux.Handle("/del", log.WrapHandler(logger, &deleteHandler{db: db}))
	mux.Handle("/export", log.WrapHandler(logger, &exportHandler{db: db, conf: conf}))
	mux.Handle("/test-auth", log.WrapHandler(logger, &authHandler{db: db, conf: conf}))
	mux.Handle("/files", log.WrapHandler(logger, &filesHandler{db: db, conf: conf}))

	var staticPath = "./static"
	pathEnv, ok := os.LookupEnv("SX_STATIC_DIR")
	if ok {
		staticPath = pathEnv
	}

	mux.Handle("/", http.FileServer(http.Dir(staticPath)))

	return &Server{db: db, mux: mux, conf: conf}
}

func (s *Server) Start(port int) error {
	return http.ListenAndServe(fmt.Sprintf(":%d", port), s.mux)
}
