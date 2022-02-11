package internal

import (
	"bytes"
	"github.com/gin-gonic/gin"
)

// ResponseBodyWriter do change transform io writer
type ResponseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r ResponseBodyWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}
