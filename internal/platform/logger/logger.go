package logger

import (
	"log"
	"net/http"
)

func RequestError(r *http.Request, scope string, err error) {
	if err == nil {
		return
	}

	log.Printf("[%s] %s %s: %v", scope, r.Method, r.URL.Path, err)
}
