package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/liondadev/sx-host/internal/betterlog"
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

func writeResponse(w http.ResponseWriter, r *http.Request, status int, message string, data interface{}) error {
	if status < 200 || status > 200 {
		var passedData []any

		fields, ok := data.(jMap)
		if ok {
			for key, val := range fields {
				fmt.Println(key, val)
				passedData = append(passedData, key, val)
			}
		}

		_ = betterlog.Error(r, message, passedData...)
	}

	w.WriteHeader(status)
	return writeJson(w, jMap{
		"status":  status,
		"message": message,
		"data":    data,
	})
}
