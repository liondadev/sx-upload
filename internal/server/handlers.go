package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/liondadev/sx-host/internal/betterlog"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/liondadev/sx-host/internal/baseurl"
	"github.com/liondadev/sx-host/internal/config"
	"github.com/liondadev/sx-host/internal/id"
)

const (
	ErrExpectedMethodGet       = "Received invalid method, expected GET"
	ErrExpectedMethodPost      = "Received invalid method, expected POST"
	ErrExpectedMethodGetOrPut  = "Received invalid method, expected GET or PUT"
	ErrNoAPIKeySpecified       = "No APIKey specified (X-SX-API-KEY)"
	ErrNoFilePassed            = "No file provided in request (file form param)"
	ErrFileToLarge             = "File too large"
	ErrInvalidAPIKey           = "Invalid API Key"
	ErrResourceNotFound        = "Resource not found"
	ErrInvalidArguments        = "Invalid # of arguments expected"
	ErrExpectedApplicationJson = "Expected Content-Type of application/json"
	ErrSQLError                = "Encountered SQL error"
	ErrFailedReadBytes         = "Failed to read content to bytes"
	ErrFailedParse             = "Failed to parse body"
	ErrNameNonEmpty            = "'name' field must be non-empty"
	ErrFailCloseFile           = "Failed to close file"
)

type filesHandler struct {
	db   *sqlx.DB
	conf *config.Config
}

func (f *filesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		_ = writeResponse(w, r, http.StatusBadRequest, ErrExpectedMethodGet, jMap{})
		return
	}

	apiKey := r.Header.Get("X-Sx-Api-Key")
	if apiKey == "" {
		_ = writeResponse(w, r, http.StatusUnauthorized, ErrNoAPIKeySpecified, jMap{})
		return
	}

	userId, found := f.conf.Keys[apiKey]
	if !found {

		_ = writeResponse(w, r, http.StatusUnauthorized, ErrInvalidAPIKey, jMap{})
		return
	}

	type row struct {
		Id               string `db:"id" json:"id"`
		Ext              string `db:"ext" json:"ext"`
		DeleteToken      string `db:"delete_token" json:"delete_token"`
		OriginalFilename string `db:"original_filename" json:"original_filename"`
	}

	var result []row
	if err := f.db.Select(&result, `SELECT "id", "ext", "delete_token", "original_filename" FROM "files" where "user_id" = ?`, userId); err != nil {
		_ = betterlog.Error(r, "Failed to select file list", "err", err)
		_ = writeResponse(w, r, http.StatusUnauthorized, ErrSQLError, jMap{})
		return
	}

	_ = writeResponse(w, r, http.StatusOK, "", result)
}

type authHandler struct {
	db   *sqlx.DB
	conf *config.Config
}

func (a *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		_ = writeResponse(w, r, http.StatusBadRequest, ErrExpectedMethodGet, jMap{})
		return
	}

	apiKey := r.Header.Get("X-Sx-Api-Key")
	if apiKey == "" {
		_ = writeResponse(w, r, http.StatusUnauthorized, ErrNoAPIKeySpecified, jMap{})
		return
	}

	us, err := a.conf.UserFromKey(apiKey)
	if err != nil {
		_ = writeResponse(w, r, http.StatusUnauthorized, ErrInvalidAPIKey, jMap{})
		return
	}

	_ = writeResponse(w, r, http.StatusOK, "", jMap{
		"user": us,
	})
}

type uploadHandler struct {
	db   *sqlx.DB
	conf *config.Config
}

func (u *uploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		_ = writeResponse(w, r, http.StatusBadRequest, ErrExpectedMethodPost, jMap{})
		return
	}

	apiKey := r.Header.Get("X-SX-API-KEY")
	if apiKey == "" {
		_ = writeResponse(w, r, http.StatusUnauthorized, ErrNoAPIKeySpecified, jMap{})
		return
	}

	us, err := u.conf.UserFromKey(apiKey)
	if err != nil {
		_ = writeResponse(w, r, http.StatusUnauthorized, ErrInvalidAPIKey, jMap{})
		return
	}

	f, h, err := r.FormFile("file")
	if err != nil {
		_ = writeResponse(w, r, http.StatusBadRequest, ErrNoFilePassed, jMap{})
		return
	}

	if h.Size > us.MaxUploadSize {
		_ = writeResponse(w, r, http.StatusBadRequest, ErrFileToLarge, jMap{
			"max_size": us.MaxUploadSize,
		})
		return
	}

	ext := path.Ext(h.Filename)

	c, err := io.ReadAll(f)
	if err != nil {
		_ = writeResponse(w, r, http.StatusInternalServerError, "Failed to read content of file into buffer", jMap{})
		return
	}

	fileId := id.New(8)
	deleteToken := id.New(32)

	_, err = u.db.Exec(
		`INSERT INTO "files" ("id", "user_id", "delete_token", "ext", "original_filename", "blob") VALUES (?, ?, ?, ?, ?, ?)`,
		fileId,
		u.conf.Keys[apiKey],
		deleteToken,
		ext,
		h.Filename,
		c,
	)

	if err != nil {
		fmt.Println(err)
		_ = writeResponse(w, r, http.StatusInternalServerError, "Failed to save file to SQLite", jMap{})
		return
	}

	_ = writeResponse(w, r, http.StatusCreated, "Created!", jMap{
		"link":   baseurl.GetBaseUrl() + "/f/" + fileId + ext,
		"delete": baseurl.GetBaseUrl() + "/del?f=" + fileId + "&t=" + deleteToken,
	})
}

type viewHandler struct {
	db   *sqlx.DB
	conf *config.Config
}

func (v *viewHandler) handleGet(w http.ResponseWriter, r *http.Request, fileId string) {
	_ = r

	type row struct {
		Blob []byte `db:"blob"`
	}

	var result row
	if err := v.db.Get(&result, `SELECT blob FROM "files" WHERE "id" = ? LIMIT 1`, fileId); err != nil {
		_ = betterlog.Error(r, "Failed to SELECT file for viewing", "err", err)
		_ = writeResponse(w, r, http.StatusNotFound, ErrResourceNotFound, jMap{})
		return
	}

	w.WriteHeader(200)
	_, _ = w.Write(result.Blob)
}

func (v *viewHandler) handlePut(w http.ResponseWriter, r *http.Request, fileId string) {
	if r.Header.Get("Content-Type") != "application/json" {
		_ = writeResponse(w, r, http.StatusBadRequest, ErrExpectedApplicationJson, jMap{})
		return
	}

	apiKey := r.Header.Get("X-SX-API-KEY")
	if apiKey == "" {
		_ = writeResponse(w, r, http.StatusUnauthorized, ErrNoAPIKeySpecified, jMap{})
		return
	}

	userId, found := v.conf.Keys[apiKey]
	if !found {
		_ = writeResponse(w, r, http.StatusUnauthorized, ErrInvalidAPIKey, jMap{})
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		_ = betterlog.Error(r, "Failed to read request body into bytes", "err", err)
		_ = writeResponse(w, r, http.StatusInternalServerError, ErrFailedReadBytes, jMap{})
		return
	}

	type Body struct {
		Name string `json:"name"`
	}
	var body Body
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		_ = writeResponse(w, r, http.StatusBadRequest, ErrFailedParse, jMap{})
		return
	}
	body.Name = strings.TrimSpace(body.Name)

	if body.Name == "" {
		_ = writeResponse(w, r, http.StatusBadRequest, ErrNameNonEmpty, jMap{})
		return
	}

	result, err := v.db.Exec("UPDATE `files` SET `original_filename` = ? WHERE `id` = ? AND `user_id` = ?", body.Name, fileId, userId)
	if err != nil {
		_ = betterlog.Error(r, "Error when renaming file", "fileId", fileId, "err", err)
		_ = writeResponse(w, r, http.StatusNotFound, ErrResourceNotFound, jMap{})
		return
	}

	n, err := result.RowsAffected()
	if err != nil || n < 1 {
		_ = writeResponse(w, r, http.StatusNotFound, ErrResourceNotFound, jMap{})
		return
	}

	_ = writeResponse(w, r, http.StatusOK, "", jMap{})
}

func (v *viewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		_ = writeResponse(w, r, http.StatusBadRequest, ErrExpectedMethodGetOrPut, jMap{})
		return
	}

	// Get file ID (required for both handlers)
	parts := strings.Split(r.URL.Path[1:], "/") // [1:] removes the first trailing slash
	if len(parts) != 2 {
		_ = writeResponse(w, r, http.StatusNotFound, ErrInvalidArguments, jMap{})
		return
	}
	fileName := parts[1]
	fileId := fileName[0 : len(fileName)-len(path.Ext(fileName))]

	switch r.Method {
	case http.MethodGet:
		v.handleGet(w, r, fileId)
		return
	case http.MethodPut:
		v.handlePut(w, r, fileId)
		return
	}

	_ = writeResponse(w, r, http.StatusBadRequest, ErrExpectedMethodGetOrPut, jMap{})
}

type deleteHandler struct {
	db *sqlx.DB
}

func (d *deleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		_ = writeResponse(w, r, http.StatusBadRequest, ErrExpectedMethodGet, jMap{})
		return
	}

	toks, ok := r.URL.Query()["t"]
	if !ok {
		_ = writeResponse(w, r, http.StatusNotFound, ErrInvalidAPIKey, jMap{})
		return
	}

	fileIds, ok := r.URL.Query()["f"]
	if !ok {
		_ = writeResponse(w, r, http.StatusNotFound, ErrResourceNotFound, jMap{})
		return
	}

	tok := toks[0]
	fileId := fileIds[0]

	result, err := d.db.Exec(`DELETE FROM "files" WHERE "id" = ? AND "delete_token" = ?`, fileId, tok)
	if err != nil {
		_ = betterlog.Error(r, "Failed to delete from files table", "err", err, "fileId", fileId, "deleteToken", tok)
		_ = writeResponse(w, r, http.StatusNotFound, ErrSQLError, jMap{})
		return
	}

	affect, err := result.RowsAffected()
	if err != nil {
		_ = writeResponse(w, r, http.StatusNotFound, ErrInvalidAPIKey, jMap{})
		return
	}

	if affect < 1 {
		_ = writeResponse(w, r, http.StatusNotFound, ErrInvalidAPIKey, jMap{})
		return
	}

	http.Redirect(w, r, baseurl.GetBaseUrl(), http.StatusTemporaryRedirect)
}

type exportHandler struct {
	db   *sqlx.DB
	conf *config.Config
}

func (d *exportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		_ = writeResponse(w, r, http.StatusBadRequest, ErrExpectedMethodGet, jMap{})
		return
	}

	apiKey := r.Header.Get("X-SX-API-KEY")
	if apiKey == "" {
		_ = writeResponse(w, r, http.StatusUnauthorized, ErrNoAPIKeySpecified, jMap{})
		return
	}

	userId, ok := d.conf.Keys[apiKey]
	if !ok {
		_ = writeResponse(w, r, http.StatusUnauthorized, ErrInvalidAPIKey, jMap{})
		return
	}

	type row struct {
		Id   string `db:"id"`
		Ext  string `db:"ext"`
		Blob []byte `db:"blob"`
	}
	var results []row
	if err := d.db.Select(&results, `SELECT "id", "ext", "blob" FROM "files" WHERE "user_id" = ?`, userId); err != nil {
		_ = betterlog.Error(r, "Failed to select files for export", "userId", userId)
		_ = writeResponse(w, r, http.StatusUnauthorized, ErrSQLError, jMap{})
		return
	}

	zipBuf := new(bytes.Buffer)
	zW := zip.NewWriter(zipBuf)

	for _, entry := range results {
		f, err := zW.Create(entry.Id + entry.Ext)
		if err != nil {
			_ = betterlog.Error(r, "Failed to create file in zip export file", "id", entry.Id, "err", err)
			continue
		}

		_, err = f.Write(entry.Blob)
		if err != nil {
			_ = betterlog.Error(r, "Failed to write file in zip export file", "id", entry.Id, "err", err)
			continue
		}
	}

	err := zW.Close()
	if err != nil {
		_ = betterlog.Error(r, "Failed to close export zip file", "err", err)
		_ = writeResponse(w, r, http.StatusUnauthorized, ErrFailCloseFile, jMap{})
		return
	}

	w.WriteHeader(200)
	io.Copy(w, zipBuf)
}
