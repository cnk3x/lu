package lu

import (
	"net/http"
	"path/filepath"
	"strings"
	"sync"
)

var cPool = &sync.Pool{New: func() interface{} { return &srContext{} }}

//Context 上下文对象
type Context interface {
	Request() *http.Request //请求
	Template() Template     //模板

	StatusText(status int)                                  //输出状态文本
	String(status int, content string)                      //输出文本
	File(path, name string)                                 //输出文件
	Blob(status int, data []byte, contentType, name string) //输出原始二进制内容
	AutoEncode(status int, data interface{})                //输出自动格式化JSON数据，根据Accept
	JSON(status int, data interface{})                      //输出JSON数据
	XML(status int, data interface{})                       //输出XML数据
	View(status int, name string, data interface{})         //输出模板数据
	Redirect(status int, to string)                         //跳转

	Method() string //方法
	Host() string   //主机（不含端口）
	Path() string   //路径
	RealIP() string //获取真实IP

	HeaderSet(name string, values ...string)
	HeaderGet(name string) string
}

func newContext(router *Router, r *http.Request) *srContext {
	c := cPool.Get().(*srContext)
	c.reset()
	c.request = r
	c.router = router
	return c
}

type srContext struct {
	router   *Router
	response response
	request  *http.Request
	header   http.Header
}

func (c *srContext) reset() {
	c.response = nil
	c.request = nil

	if c.header == nil {
		c.header = make(http.Header)
	}

	for n := range c.header {
		c.header.Del(n)
	}
}

func (c *srContext) release() {
	c.reset()
	cPool.Put(c)
}

func (c *srContext) Request() *http.Request {
	return c.request
}

func (c *srContext) Template() Template {
	return c.router.template
}

func (c *srContext) StatusText(status int) {
	c.Blob(status, []byte(http.StatusText(status)), "text/plain;charset=utf8", "")
}

func (c *srContext) String(status int, content string) {
	c.Blob(status, []byte(content), "text/plain;charset=utf8", "")
}

func (c *srContext) File(path, name string) {
	if name == "" {
		_, name = filepath.Split(path)
	}
	c.response = &fileResponse{path: path, attachName: name}
}

func (c *srContext) Blob(status int, data []byte, contentType, name string) {
	c.response = &binaryResponse{
		status:      status,
		binary:      data,
		contentType: contentType,
		attachName:  name,
	}
}

func (c *srContext) AutoEncode(status int, data interface{}) {
	c.response = &encoderResponse{format: "auto", data: data, status: status}
}

func (c *srContext) JSON(status int, data interface{}) {
	c.response = &encoderResponse{format: "json", data: data, status: status}
}

func (c *srContext) XML(status int, data interface{}) {
	c.response = &encoderResponse{format: "xml", data: data, status: status}
}

func (c *srContext) View(status int, name string, data interface{}) {
	c.response = &viewResponse{status: status, name: name, data: data}
}

func (c *srContext) Redirect(status int, to string) {
	c.response = &redirectResponse{
		status: status,
		to:     to,
	}
}

func (c *srContext) Path() string {
	return "/" + strings.Trim(strings.ReplaceAll(c.request.URL.Path, " ", ""), "/")
}

func (c *srContext) Method() string {
	return c.request.Method
}

func (c *srContext) Host() string {
	hostport := c.request.Host
	if hostport[0] == '[' {
		end := strings.IndexRune(hostport, ']')
		if end < 0 {
			return ""
		}
		return hostport[1:end]
	}
	end := strings.IndexRune(hostport, ':')
	if end < 1 {
		return ""
	}
	return hostport[:end]
}

func (c *srContext) RealIP() string {
	return realip(c.request)
}

func (c *srContext) HeaderGet(name string) string {
	return c.request.Header.Get(name)
}

func (c *srContext) HeaderSet(name string, values ...string) {
	if len(values) > 0 {
		c.header[name] = values
	} else {
		c.header.Del(name)
	}
}
