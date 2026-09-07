package api

import (
	"net/http"
)

func OKResponse(w http.ResponseWriter, data any) {
}

func ErrorResponse(w http.ResponseWriter, status int, message string) {
}
