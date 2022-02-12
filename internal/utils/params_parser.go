package utils

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"net/http"
	"net/url"
	"strings"
)

func GetQuery(req *http.Request) map[string]interface{} {
	m := make(map[string]interface{})

	query, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		return m
	}
	if len(query) > 0 {
		for key, strArr := range query {
			m[key] = strings.Join(strArr, ",")
		}
	}
	return m
}

func ParameterParser(c *gin.Context) map[string]interface{} {
	m := make(map[string]interface{})
	split := strings.Split(c.FullPath(), `/`)
	params := strings.Split(c.Request.URL.Path, `/`)
	for i, preKey := range split {
		if strings.Contains(preKey, ":") {
			key := strings.ReplaceAll(preKey, ":", "")
			m[key] = params[i]
		}
	}
	if c.Request.Method == http.MethodGet {

		//整合query参数
		queryParams := GetQuery(c.Request)
		for key, val := range queryParams {
			m[key] = val
		}
	} else if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut {
		postMap := make(map[string]interface{})
		err := c.ShouldBindBodyWith(&postMap, binding.JSON)
		if err == nil {
			for key, val := range postMap {
				m[key] = val
			}
		}
	}
	return m
}
