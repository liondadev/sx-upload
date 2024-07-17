package server

import (
	"encoding/json"
	"net/http"
)

type jMap map[string]interface{}

func writeJson(w http.ResponseWriter, body interface{}) error {
	cont, err := json.Marshal(body)
	if err != nil {
		return err
	}

	_, err = w.Write(cont)
	return err
}

func writeResponse(w http.ResponseWriter, status int, message string, data interface{}) error {
	w.WriteHeader(status)
	return writeJson(w, jMap{
		"status":  status,
		"message": message,
		"data":    data,
	})
}
