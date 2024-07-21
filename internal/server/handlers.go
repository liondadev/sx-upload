package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/liondadev/sx-host/internal/baseurl"
	"github.com/liondadev/sx-host/internal/config"
	"github.com/liondadev/sx-host/internal/id"
)

type filesHandler struct {
	db   *sqlx.DB
	conf *config.Config
}

func (f *filesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		_ = writeResponse(w, http.StatusBadRequest, "This route expects a GET request", jMap{})
		return
	}

	apiKey := r.Header.Get("X-Sx-Api-Key")
	if apiKey == "" {
		_ = writeResponse(w, http.StatusUnauthorized, "No APIKey specified (X-SX-API-KEY)", jMap{})
		return
	}

	userId, found := f.conf.Keys[apiKey]
	if !found {
		_ = writeResponse(w, http.StatusUnauthorized, "Invalid APIKey", jMap{})
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
		_ = writeResponse(w, http.StatusUnauthorized, "Failed to perform select.", jMap{})
		return
	}

	_ = writeResponse(w, http.StatusOK, "", result)
}

type authHandler struct {
	db   *sqlx.DB
	conf *config.Config
}

func (a *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		_ = writeResponse(w, http.StatusBadRequest, "This route expects a POST request", jMap{})
		return
	}

	apiKey := r.Header.Get("X-Sx-Api-Key")
	if apiKey == "" {
		_ = writeResponse(w, http.StatusUnauthorized, "No APIKey specified (X-SX-API-KEY)", jMap{})
		return
	}

	us, err := a.conf.UserFromKey(apiKey)
	if err != nil {
		_ = writeResponse(w, http.StatusUnauthorized, "Invalid APIKey", jMap{})
		return
	}

	_ = writeResponse(w, http.StatusOK, "", jMap{
		"user": us,
	})
}

type uploadHandler struct {
	db   *sqlx.DB
	conf *config.Config
}

func (u *uploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		_ = writeResponse(w, http.StatusBadRequest, "This route expects a POST request", jMap{})
		return
	}

	apiKey := r.Header.Get("X-SX-API-KEY")
	if apiKey == "" {
		_ = writeResponse(w, http.StatusUnauthorized, "No APIKey specified (X-SX-API-KEY)", jMap{})
		return
	}

	us, err := u.conf.UserFromKey(apiKey)
	if err != nil {
		_ = writeResponse(w, http.StatusUnauthorized, "Invalid APIKey", jMap{})
		return
	}

	f, h, err := r.FormFile("file")
	if err != nil {
		_ = writeResponse(w, http.StatusBadRequest, "No file passed ('file' field)", jMap{})
		return
	}

	if h.Size > us.MaxUploadSize {
		_ = writeResponse(w, http.StatusBadRequest, "File too large", jMap{
			"max_size": us.MaxUploadSize,
		})
		return
	}

	ext := path.Ext(h.Filename)

	c, err := io.ReadAll(f)
	if err != nil {
		_ = writeResponse(w, http.StatusInternalServerError, "Failed to read content of file into buffer", jMap{})
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
		_ = writeResponse(w, http.StatusInternalServerError, "Failed to save file to SQLite", jMap{})
		return
	}

	_ = writeResponse(w, http.StatusCreated, "Created!", jMap{
		"link":   baseurl.GetBaseUrl() + "/f/" + fileId + ext,
		"delete": baseurl.GetBaseUrl() + "/del?f=" + fileId + "&t=" + deleteToken,
	})
}

type viewHandler struct {
	db   *sqlx.DB
	conf *config.Config
}

func (v *viewHandler) handleGet(w http.ResponseWriter, r *http.Request, fileId string) {
	type row struct {
		Blob []byte `db:"blob"`
	}

	var result row
	if err := v.db.Get(&result, `SELECT blob FROM "files" WHERE "id" = ? LIMIT 1`, fileId); err != nil {
		_ = writeResponse(w, http.StatusNotFound, "File not found.", jMap{})
		return
	}

	w.WriteHeader(200)
	w.Write(result.Blob)
}

func (v *viewHandler) handlePut(w http.ResponseWriter, r *http.Request, fileId string) {
	if r.Header.Get("Content-Type") != "application/json" {
		_ = writeResponse(w, http.StatusBadRequest, "This request expects Content-Type of application/json", jMap{})
		return
	}

	apiKey := r.Header.Get("X-SX-API-KEY")
	if apiKey == "" {
		_ = writeResponse(w, http.StatusUnauthorized, "No APIKey specified (X-SX-API-KEY)", jMap{})
		return
	}

	userId, found := v.conf.Keys[apiKey]
	if !found {
		_ = writeResponse(w, http.StatusUnauthorized, "Invalid API Key", jMap{})
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		_ = writeResponse(w, http.StatusInternalServerError, "Internal Server Error while reading body bytes", jMap{})
		return
	}

	type Body struct {
		Name string `json:"name"`
	}
	var body Body
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		_ = writeResponse(w, http.StatusBadRequest, "Failed to parse body.", jMap{})
		return
	}
	body.Name = strings.TrimSpace(body.Name)

	if body.Name == "" {
		_ = writeResponse(w, http.StatusBadRequest, "'name' field must be non-empty", jMap{})
		return
	}

	result, err := v.db.Exec("UPDATE `files` SET `original_filename` = ? WHERE `id` = ? AND `user_id` = ? LIMIT 1", body.Name, fileId, userId)
	if err != nil {
		_ = writeResponse(w, http.StatusInternalServerError, "Internal Server Error when making HTTP request.", jMap{})
		return
	}

	n, err := result.RowsAffected()
	if err != nil || n < 1 {
		_ = writeResponse(w, http.StatusInternalServerError, "Either no rows were affected or we failed to get the # of rows affected.", jMap{})
		return
	}

	_ = writeResponse(w, http.StatusOK, "", jMap{})
}

func (v *viewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		_ = writeResponse(w, http.StatusBadRequest, "This route expects a GET or PUT request", jMap{})
		return
	}

	// Get file ID (required for both handlers)
	parts := strings.Split(r.URL.Path[1:], "/") // [1:] removes the first trailing slash
	if len(parts) != 2 {
		_ = writeResponse(w, http.StatusNotFound, "Invalid # of arguments (2 expected)", jMap{})
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

	_ = writeResponse(w, http.StatusBadRequest, "This route expects a GET or PUT request", jMap{})
}

type deleteHandler struct {
	db *sqlx.DB
}

func (d *deleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		_ = writeResponse(w, http.StatusBadRequest, "This route expects a GET request", jMap{})
		return
	}

	toks, ok := r.URL.Query()["t"]
	if !ok {
		_ = writeResponse(w, http.StatusNotFound, "Invalid token", jMap{})
		return
	}

	fileIds, ok := r.URL.Query()["f"]
	if !ok {
		_ = writeResponse(w, http.StatusNotFound, "Invalid token", jMap{})
		return
	}

	tok := toks[0]
	fileId := fileIds[0]

	result, err := d.db.Exec(`DELETE FROM "files" WHERE "id" = ? AND "delete_token" = ?`, fileId, tok)
	if err != nil {
		fmt.Println(err)
		_ = writeResponse(w, http.StatusNotFound, "Failed to remove image", jMap{})
		return
	}

	affect, err := result.RowsAffected()
	if err != nil {
		_ = writeResponse(w, http.StatusNotFound, "Failed to fetch rows affected", jMap{})
		return
	}

	if affect < 1 {
		_ = writeResponse(w, http.StatusNotFound, "Invalid image ID", jMap{})
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
		_ = writeResponse(w, http.StatusBadRequest, "This route expects a GET request", jMap{})
		return
	}

	apiKey := r.Header.Get("X-SX-API-KEY")
	if apiKey == "" {
		_ = writeResponse(w, http.StatusUnauthorized, "No APIKey specified (X-SX-API-KEY)", jMap{})
		return
	}

	userId, ok := d.conf.Keys[apiKey]
	if !ok {
		_ = writeResponse(w, http.StatusUnauthorized, "Invalid APIKey", jMap{})
		return
	}

	type row struct {
		Id   string `db:"id"`
		Ext  string `db:"ext"`
		Blob []byte `db:"blob"`
	}
	var results []row
	if err := d.db.Select(&results, `SELECT "id", "ext", "blob" FROM "files" WHERE "user_id" = ?`, userId); err != nil {
		fmt.Println(err)
		_ = writeResponse(w, http.StatusUnauthorized, "Failed to select files to export.", jMap{})
		return
	}

	zipBuf := new(bytes.Buffer)
	zW := zip.NewWriter(zipBuf)

	for _, entry := range results {
		f, err := zW.Create(entry.Id + entry.Ext)
		if err != nil {
			fmt.Printf("Failed to add %s to zip file: %s", entry.Id, err)
			continue
		}

		_, err = f.Write(entry.Blob)
		if err != nil {
			fmt.Printf("Failed to write to file %s: %s", entry.Id, err)
			continue
		}
	}

	err := zW.Close()
	if err != nil {
		_ = writeResponse(w, http.StatusUnauthorized, "Failed to close zip file.", jMap{})
		return
	}

	w.WriteHeader(200)
	io.Copy(w, zipBuf)
}
