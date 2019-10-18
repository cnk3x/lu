package lu

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
)

type (
	//Router 路由
	Router struct {
		prefix      string
		handlers    *tree
		groups      *tree
		assets      *tree
		middlewares []Middleware
		template    Template
	}

	//HandlerFunc 执行函数
	HandlerFunc func(Context)

	//Middleware 中间件
	Middleware interface {
		Apply(next HandlerFunc) HandlerFunc
	}

	//MiddlewareFunc 中间件方法
	MiddlewareFunc func(next HandlerFunc) HandlerFunc

	//Template 模板引擎
	Template interface {
		ExecuteTemplate(w io.Writer, name string, data interface{}) error
	}

	pathHandler struct {
		get  HandlerFunc
		post HandlerFunc
		any  HandlerFunc
	}
)

//New 新建
func New() *Router {
	return &Router{
		handlers:    newRadix(),
		groups:      newRadix(),
		assets:      newRadix(),
		middlewares: []Middleware{},
	}
}

//SetTemplate 设置模板引擎
func (router *Router) SetTemplate(template Template) {
	router.template = template
}

//Use 使用中间件
func (router *Router) Use(middlewares ...Middleware) {
	router.middlewares = append(router.middlewares, middlewares...)
}

//Group 路由分组
func (router *Router) Group(path string, middlewares ...Middleware) *Router {
	path = router.subPath(path)
	newSR := New()
	newSR.prefix = path
	newSR.template = router.template
	newSR.Use(router.middlewares...)
	newSR.Use(middlewares...)
	router.groups.Insert(path, newSR)
	return newSR
}

//Handle Handle
func (router *Router) Handle(method string, path string, handlerFunc HandlerFunc) {
	path = router.subPath(path)
	log.Printf("handle %5s -> %s\n", method, path)
	item := router.findHandler(path)
	if item == nil {
		item = &pathHandler{}
		router.handlers.Insert(path, item)
	}
	switch strings.ToUpper(method) {
	case http.MethodGet:
		item.get = handlerFunc
	case http.MethodPost:
		item.post = handlerFunc
	default:
		item.any = handlerFunc
	}
}

//Assets Handle
func (router *Router) Assets(path string, dir string) {
	path = router.subPath(path)
	log.Printf("static %s -> %s\n", path, dir)
	router.assets.Insert(path, dir)
}

//ServeHTTP 实现 http Handler接口
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			_, f, l, _ := runtime.Caller(0)
			http.Error(w, fmt.Sprintf("%s:%d %#v", f, l, err), http.StatusInternalServerError)
		}
	}()

	c := newContext(router, r)
	defer c.release()
	router.serveContext(c)

	for n := range c.header {
		w.Header().Set(n, c.header.Get(n))
	}

	if c.response != nil {
		c.response.Render(c, w)
	}
}

func (router *Router) serveContext(c Context) {
	routerPath := c.Path()

	//先查精确匹配
	if item := router.findHandler(routerPath); item != nil {
		if h := item.findMethod(c.Method()); h != nil {
			// l := len(router.middlewares)
			for _, mw := range router.middlewares {
				h = mw.Apply(h) //router.middlewares[l-i-1] 倒排
			}
			h(c)
			return
		}
		c.StatusText(http.StatusMethodNotAllowed)
		return
	}

	//再查分组
	if group := router.findGroup(routerPath); group != nil {
		group.serveContext(c)
		return
	}

	//再查分组
	if prefix, dir := router.findAssets(routerPath); dir != "" {
		path := filepath.Join(dir, strings.TrimPrefix(strings.TrimPrefix(c.Path(), prefix), "/"))
		_, name := filepath.Split(path)
		c.File(path, name)
		return
	}

	//找不到，404
	c.StatusText(http.StatusNotFound)
}

//精确匹配
func (router *Router) findHandler(path string) *pathHandler {
	if i, ok := router.handlers.Get(strings.ToLower(path)); ok {
		return i.(*pathHandler)
	}
	return nil
}

//查前缀
func (router *Router) findGroup(path string) *Router {
	if _, i, ok := router.groups.Prefix(strings.ToLower(path)); ok {
		return i.(*Router)
	}
	return nil
}

//查前缀
func (router *Router) findAssets(path string) (string, string) {
	if root, i, ok := router.assets.Prefix(strings.ToLower(path)); ok {
		return root, i.(string)
	}
	return "", ""
}

func (router *Router) subPath(path string) string {
	return "/" + strings.ToLower(strings.Trim(router.prefix+strings.ReplaceAll(path, " ", ""), "/"))
}

func (item *pathHandler) findMethod(method string) HandlerFunc {
	var h HandlerFunc
	switch strings.ToUpper(method) {
	case http.MethodGet:
		h = item.get
	case http.MethodPost:
		h = item.post
	}
	if h == nil {
		h = item.any
	}
	return h
}

//Apply middleware Apply
func (m MiddlewareFunc) Apply(next HandlerFunc) HandlerFunc {
	return m(next)
}
