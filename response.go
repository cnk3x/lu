package lu

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type response interface {
	Render(c Context, w http.ResponseWriter)
}

type statusTextResp struct {
	status int
}

func (resp *statusTextResp) Render(c Context, w http.ResponseWriter) {
	http.Error(w, http.StatusText(resp.status), resp.status)
}

type encoderResponse struct {
	status int
	format string
	data   interface{}
}

func (resp *encoderResponse) decterFormat(c Context) {
	if resp.format == "" {
		accept := c.HeaderGet("Accept")
		switch {
		case strings.Contains(accept, "xml"):
			resp.format = "xml"
		case strings.Contains(accept, "json") || strings.Contains(accept, "javascript"):
			resp.format = "json"
		}
	}
}

func (resp *encoderResponse) Render(c Context, w http.ResponseWriter) {
	b := &bytes.Buffer{}
	contentType := "text/plain;charset=utf-8"

	resp.decterFormat(c)

	switch resp.format {
	case "xml":
		if err := xml.NewEncoder(b).Encode(resp.data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		contentType = "text/xml;charset=utf-8"
	case "json":
		if err := json.NewEncoder(b).Encode(resp.data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		contentType = "application/json;charset=utf-8"
	default:
		if _, err := fmt.Fprintf(b, "%#v", resp.data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(resp.status)
	w.Header().Set("Content-Type", contentType)
	w.Write(b.Bytes())
}

type viewResponse struct {
	status int
	name   string
	data   interface{}
}

func (resp *viewResponse) templateRender(c Context, w http.ResponseWriter) ([]byte, error) {
	b := &bytes.Buffer{}
	t := c.Template()
	if t != nil {
		return nil, t.ExecuteTemplate(b, resp.name, resp.data)
	}
	return b.Bytes(), nil
}

func (resp *viewResponse) Render(c Context, w http.ResponseWriter) {
	var data []byte
	if resp.name != "" {
		w.Header().Set("Content-Type", "text/plain;charset=utf-8")
		var err error
		if data, err = resp.templateRender(c, w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		b := &bytes.Buffer{}
		contentType := "text/plain;charset=utf-8"
		accept := c.HeaderGet("Accept")

		switch {
		case strings.Contains(accept, "xml"):
			if err := xml.NewEncoder(b).Encode(resp.data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			contentType = "text/xml;charset=utf-8"
		case strings.Contains(accept, "json") || strings.Contains(accept, "javascript"):
			if err := json.NewEncoder(b).Encode(resp.data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			contentType = "application/json;charset=utf-8"
		default:
			if _, err := fmt.Fprintf(b, "%#v", resp.data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		w.Header().Set("Content-Type", contentType)
		data = b.Bytes()
	}

	w.WriteHeader(resp.status)
	w.Write(data)
}

type binaryResponse struct {
	status      int
	binary      []byte
	contentType string
	attachName  string
}

func (resp *binaryResponse) Render(c Context, w http.ResponseWriter) {
	w.WriteHeader(resp.status)
	w.Header().Set("Content-Type", resp.contentType)
	if _, err := w.Write(resp.binary); err != nil {
		log.Println(err)
	}
}

type fileResponse struct {
	path        string
	attachName  string
	fatalPath   string
	inFatalCall bool
}

func (resp *fileResponse) handle404(c Context, w http.ResponseWriter) {
	if !resp.inFatalCall && resp.fatalPath != "" {
		resp.path = resp.fatalPath
		resp.inFatalCall = true
		resp.Render(c, w)
		return
	}
	http.NotFound(w, c.Request())
}

func (resp *fileResponse) Render(c Context, w http.ResponseWriter) {
	if _, name := filepath.Split(resp.path); name[0] == '_' || name[0] == '.' {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	f, err := os.Stat(resp.path)
	if err != nil {
		if os.IsNotExist(err) { //找不到404
			resp.handle404(c, w)
			return
		}
		if os.IsPermission(err) { //无权限 403
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError) //其他错误，内部错误
		return
	}

	if f.IsDir() { //如果是文件夹
		// http.Error(w, err.Error(), http.StatusForbidden)
		resp.path = filepath.Join(resp.path, "index.html")
		resp.Render(c, w)
		return
	}

	fs, err := os.Open(resp.path)
	if err != nil { //打开文件出错， 500
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer fs.Close()
	http.ServeContent(w, c.Request(), resp.attachName, f.ModTime(), fs)
}

type redirectResponse struct {
	status int
	to     string
}

func (resp *redirectResponse) Render(c Context, w http.ResponseWriter) {
	http.Redirect(w, c.Request(), resp.to, resp.status)
}
