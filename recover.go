package lu

import (
	"fmt"
	"net/http"
)

//Recover 异常恢复中间件
func Recover() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) {
			defer func() {
				if err := recover(); err != nil {
					c.String(http.StatusInternalServerError, fmt.Sprintf("%v", err))
				}
			}()
			next(c)
		}
	}
}
