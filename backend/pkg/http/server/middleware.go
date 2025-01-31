package http_server

import (
	"net/http"
)

const timeLayout = "02/Jan/2006:15:04:05 -0700"

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}
