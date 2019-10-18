package lu

import (
	"net/http"
	"path"
	"strconv"
)

//Cors Cors
func Cors() Middleware {
	return &CorsMw{AllowCredentials: true, MaxAge: 600}
}

//CorsMw CorsMw
type CorsMw struct {
	Origins          []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	MaxAge           int
}

//Apply Apply
func (m *CorsMw) Apply(nxt HandlerFunc) HandlerFunc {
	return func(c Context) {
		origin := c.HeaderGet("Origin")
		method := c.Method()
		if origin != "" && stringSliceContains(m.Origins, origin) {
			if len(m.AllowMethods) == 0 {
				if method != "OPTIONS" {
					m.AllowMethods = []string{method}
				} else {
					m.AllowMethods = []string{"GET", "POST", "OPTIONS", "PUT", "PATCH"}
				}
			}

			c.HeaderSet("Access-Control-Allow-Origin", origin)
			c.HeaderSet("Access-Control-Allow-Credentials", strconv.FormatBool(m.AllowCredentials))
			c.HeaderSet("Access-Control-Allow-Methods", m.AllowMethods...)
			c.HeaderSet("Access-Control-Allow-Headers", m.AllowHeaders...)
			c.HeaderSet("Access-Control-Max-Age", strconv.Itoa(m.MaxAge))

			if method == "OPTIONS" {
				c.StatusText(http.StatusNoContent)
				return
			}
		}
		nxt(c)
	}
}

func stringSliceContains(stringSlice []string, item string) bool {
	if len(stringSlice) == 0 {
		return true
	}
	for _, s := range stringSlice {
		if s == item || s == "*" {
			return true
		}
		if ok, _ := path.Match(s, item); ok {
			return true
		}
	}
	return false
}
