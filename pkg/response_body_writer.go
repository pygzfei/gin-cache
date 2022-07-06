package pkg

import (
	"bytes"
	"github.com/gin-gonic/gin"
)

// ResponseBodyWriter do change transform io writer
type ResponseBodyWriter struct {
	gin.ResponseWriter
	Body *bytes.Buffer
}

func (r ResponseBodyWriter) Write(b []byte) (int, error) {
	r.Body.Write(b)
	return r.ResponseWriter.Write(b)
}
