// Package web provides a web.go compitable layer for reusing the code written with
// hoisie's `web.go` framework. Basiclly this package add web.Context to
// martini's dependency injection system.
package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	htemplate "html/template"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-martini/martini"
	"github.com/gobuild/log"
)

// A Context object is created for every incoming HTTP request, and is
// passed to handlers as an optional first argument. It provides information
// about the request, including the http.Request object, the GET and POST params,
// and acts as a Writer for the response.
type Context struct {
	Request      *http.Request
	Params       map[string]string
	cookieSecret string
	http.ResponseWriter
	*renderer

	// Flash *Flash
	Data map[string]interface{}
}

// new func added
// Query querys form parameter.
func (ctx *Context) Query(name string) string {
	ctx.Request.ParseForm()
	return ctx.Request.Form.Get(name)
}

// HasError returns true if error occurs in form validation.
func (ctx *Context) HasApiError() bool {
	hasErr, ok := ctx.Data["HasError"]
	if !ok {
		return false
	}
	return hasErr.(bool)
}

func (ctx *Context) GetErrMsg() string {
	return ctx.Data["ErrorMsg"].(string)
}

// HasError returns true if error occurs in form validation.
func (ctx *Context) HasError() bool {
	hasErr, ok := ctx.Data["HasError"]
	if !ok {
		return false
	}
	// ctx.Flash.ErrorMsg = ctx.Data["ErrorMsg"].(string)
	// ctx.Data["Flash"] = ctx.Flash
	return hasErr.(bool)
}

// HTML calls render.HTML underlying but reduce one argument.
func (ctx *Context) HTML(status int, name string, htmlOpt ...HTMLOptions) {
	ctx.renderer.HTML(status, name, ctx.Data, htmlOpt...)
}

// // RenderWithErr used for page has form validation but need to prompt error to users.
// func (ctx *Context) RenderWithErr(msg, tpl string, form auth.Form) {
// 	if form != nil {
// 		auth.AssignForm(form, ctx.Data)
// 	}
// 	// ctx.Flash.ErrorMsg = msg
// 	// ctx.Data["Flash"] = ctx.Flash
// 	ctx.HTML(200, tpl)
// }

// Handle handles and logs error by given status.
func (ctx *Context) Handle(status int, desc string, err error) {
	if err != nil {
		log.Error("%s: %v", desc, err)
		if martini.Dev != martini.Prod {
			ctx.Data["ErrorMsg"] = err
		}
	}

	ctx.Data["Description"] = "desc:" + desc
	ctx.Data["Status"] = status
	switch status {
	case 404:
		ctx.Data["Title"] = "Page Not Found"
	case 500:
		ctx.Data["Title"] = "Internal Server Error"
	default:
		ctx.Data["Title"] = fmt.Sprintf("Status - %d", status)
	}
	ctx.HTML(status, fmt.Sprintf("status/%dx", status/10))
}

func (ctx *Context) ServeFile(file string, names ...string) {
	var name string
	if len(names) > 0 {
		name = names[0]
	} else {
		name = filepath.Base(file)
	}
	ctx.Header().Set("Content-Description", "File Transfer")
	ctx.Header().Set("Content-Type", "application/octet-stream")
	ctx.Header().Set("Content-Disposition", "attachment; filename="+name)
	ctx.Header().Set("Content-Transfer-Encoding", "binary")
	ctx.Header().Set("Expires", "0")
	ctx.Header().Set("Cache-Control", "must-revalidate")
	ctx.Header().Set("Pragma", "public")
	http.ServeFile(ctx.ResponseWriter, ctx.Request, file)
}

func (ctx *Context) ServeContent(name string, r io.ReadSeeker, params ...interface{}) {
	modtime := time.Now()
	for _, p := range params {
		switch v := p.(type) {
		case time.Time:
			modtime = v
		}
	}
	ctx.Header().Set("Content-Description", "File Transfer")
	ctx.Header().Set("Content-Type", "application/octet-stream")
	ctx.Header().Set("Content-Disposition", "attachment; filename="+name)
	ctx.Header().Set("Content-Transfer-Encoding", "binary")
	ctx.Header().Set("Expires", "0")
	ctx.Header().Set("Cache-Control", "must-revalidate")
	ctx.Header().Set("Pragma", "public")
	http.ServeContent(ctx.ResponseWriter, ctx.Request, name, modtime, r)
}

// type Flash struct {
// 	url.Values
// 	ErrorMsg, SuccessMsg string
// }

// func (f *Flash) Error(msg string) {
// 	f.Set("error", msg)
// 	f.ErrorMsg = msg
// }

// func (f *Flash) Success(msg string) {
// 	f.Set("success", msg)
// 	f.SuccessMsg = msg
// }

// end of added --

// if cookie secret is set to "", then SetSecureCookie would not work
func ContextWithCookieSecret(secret string, options ...Options) martini.Handler {
	// render
	opt := prepareOptions(options)
	cs := prepareCharset(opt.Charset)
	t := compile(opt)

	return func(w http.ResponseWriter, req *http.Request, mc martini.Context) {
		// render
		var tc *htemplate.Template
		if martini.Env == martini.Dev {
			tc = compile(opt)
		} else {
			tc, _ = t.Clone()
		}
		render := &renderer{w, req, tc, opt, cs}

		// Request      *http.Request
		// 	Params       map[string]string
		// 	cookieSecret string
		// 	http.ResponseWriter
		// 	*renderer

		// 	Flash *Flash
		// 	Data  map[string]interface{}
		ctx := &Context{
			Request:        req,
			Params:         map[string]string{},
			cookieSecret:   secret,
			ResponseWriter: w,
			renderer:       render,
			// Flash:          new(Flash),
			Data: make(map[string]interface{}),
		}
		//set some default headers
		tm := time.Now().UTC()

		//ignore errors from ParseForm because it's usually harmless.
		req.ParseForm()
		if len(req.Form) > 0 {
			for k, v := range req.Form {
				ctx.Params[k] = v[0]
			}
		}
		ctx.SetHeader("Date", webTime(tm), true)
		//Set the default content-type
		ctx.SetHeader("Content-Type", "text/html; charset=utf-8", true)
		// set martini context for web.Context
		mc.Map(ctx)
	}
}

// internal utility methods
func webTime(t time.Time) string {
	ftime := t.Format(time.RFC1123)
	if strings.HasSuffix(ftime, "UTC") {
		ftime = ftime[0:len(ftime)-3] + "GMT"
	}
	return ftime
}

// WriteString writes string data into the response object.
func (ctx *Context) WriteString(content string) {
	ctx.ResponseWriter.Write([]byte(content))
}

// Abort is a helper method that sends an HTTP header and an optional
// body. It is useful for returning 4xx or 5xx errors.
// Once it has been called, any return value from the handler will
// not be written to the response.
func (ctx *Context) Abort(status int, body string) {
	ctx.ResponseWriter.WriteHeader(status)
	ctx.ResponseWriter.Write([]byte(body))
}

// Redirect is a helper method for 3xx redirects.
func (ctx *Context) Redirect(status int, url_ string) {
	ctx.ResponseWriter.Header().Set("Location", url_)
	ctx.ResponseWriter.WriteHeader(status)
	ctx.ResponseWriter.Write([]byte("Redirecting to: " + url_))
}

// Notmodified writes a 304 HTTP response
func (ctx *Context) NotModified() {
	ctx.ResponseWriter.WriteHeader(304)
}

// NotFound writes a 404 HTTP response
func (ctx *Context) NotFound(message string) {
	ctx.ResponseWriter.WriteHeader(404)
	ctx.ResponseWriter.Write([]byte(message))
}

//Unauthorized writes a 401 HTTP response
func (ctx *Context) Unauthorized() {
	ctx.ResponseWriter.WriteHeader(401)
}

//Forbidden writes a 403 HTTP response
func (ctx *Context) Forbidden() {
	ctx.ResponseWriter.WriteHeader(403)
}

// ContentType sets the Content-Type header for an HTTP response.
// For example, ctx.ContentType("json") sets the content-type to "application/json"
// If the supplied value contains a slash (/) it is set as the Content-Type
// verbatim. The return value is the content type as it was
// set, or an empty string if none was found.
func (ctx *Context) ContentType(val string) string {
	var ctype string
	if strings.ContainsRune(val, '/') {
		ctype = val
	} else {
		if !strings.HasPrefix(val, ".") {
			val = "." + val
		}
		ctype = mime.TypeByExtension(val)
	}
	if ctype != "" {
		ctx.Header().Set("Content-Type", ctype)
	}
	return ctype
}

// SetHeader sets a response header. If `unique` is true, the current value
// of that header will be overwritten . If false, it will be appended.
func (ctx *Context) SetHeader(hdr string, val string, unique bool) {
	if unique {
		ctx.Header().Set(hdr, val)
	} else {
		ctx.Header().Add(hdr, val)
	}
}

// SetCookie adds a cookie header to the response.
func (ctx *Context) SetCookie(cookie *http.Cookie) {
	ctx.SetHeader("Set-Cookie", cookie.String(), false)
}

func getCookieSig(key string, val []byte, timestamp string) string {
	hm := hmac.New(sha1.New, []byte(key))

	hm.Write(val)
	hm.Write([]byte(timestamp))

	hex := fmt.Sprintf("%02x", hm.Sum(nil))
	return hex
}

// NewCookie is a helper method that returns a new http.Cookie object.
// Duration is specified in seconds. If the duration is zero, the cookie is permanent.
// This can be used in conjunction with ctx.SetCookie.
func NewCookie(name string, value string, age int64) *http.Cookie {
	var utctime time.Time
	if age == 0 {
		// 2^31 - 1 seconds (roughly 2038)
		utctime = time.Unix(2147483647, 0)
	} else {
		utctime = time.Unix(time.Now().Unix()+age, 0)
	}
	return &http.Cookie{Name: name, Value: value, Expires: utctime}
}

func (ctx *Context) SetSecureCookie(name string, val string, age int64) {
	//base64 encode the val
	if len(ctx.cookieSecret) == 0 {
		return
	}
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	encoder.Write([]byte(val))
	encoder.Close()
	vs := buf.String()
	vb := buf.Bytes()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sig := getCookieSig(ctx.cookieSecret, vb, timestamp)
	cookie := strings.Join([]string{vs, timestamp, sig}, "|")
	ctx.SetCookie(NewCookie(name, cookie, age))
}

func (ctx *Context) GetSecureCookie(name string) (string, bool) {
	for _, cookie := range ctx.Request.Cookies() {
		if cookie.Name != name {
			continue
		}

		parts := strings.SplitN(cookie.Value, "|", 3)

		val := parts[0]
		timestamp := parts[1]
		sig := parts[2]

		if getCookieSig(ctx.cookieSecret, []byte(val), timestamp) != sig {
			return "", false
		}

		ts, _ := strconv.ParseInt(timestamp, 0, 64)

		if time.Now().Unix()-31*86400 > ts {
			return "", false
		}

		buf := bytes.NewBufferString(val)
		encoder := base64.NewDecoder(base64.StdEncoding, buf)

		res, _ := ioutil.ReadAll(encoder)
		return string(res), true
	}
	return "", false
}
