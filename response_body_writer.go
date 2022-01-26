package gin_cache

import (
	"bytes"
	"github.com/gin-gonic/gin"
)

type ResponseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (this ResponseBodyWriter) Write(b []byte) (int, error) {
	this.body.Write(b)
	return this.ResponseWriter.Write(b)
}
