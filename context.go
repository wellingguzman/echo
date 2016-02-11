package echo

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"path/filepath"
	"time"

	"github.com/labstack/echo/engine"
	"github.com/labstack/echo/logger"

	"net/url"

	"bytes"

	netContext "golang.org/x/net/context"
	"golang.org/x/net/websocket"
)

type (
	// Context represents context for the current request. It holds request and
	// response objects, path parameters, data and registered handler.
	Context interface {
		netContext.Context
		Request() engine.Request
		Response() engine.Response
		Socket() *websocket.Conn
		Path() string
		P(int) string
		Param(string) string
		Query(string) string
		Form(string) string
		Set(string, interface{})
		Get(string) interface{}
		Bind(interface{}) error
		Render(int, string, interface{}) error
		HTML(int, string) error
		String(int, string) error
		JSON(int, interface{}) error
		JSONBlob(int, []byte) error
		JSONP(int, string, interface{}) error
		XML(int, interface{}) error
		XMLBlob(int, []byte) error
		File(string, string, bool) error
		NoContent(int) error
		Redirect(int, string) error
		Error(err error)
		Logger() logger.Logger
		Object() *context
	}

	context struct {
		request  engine.Request
		response engine.Response
		socket   *websocket.Conn
		path     string
		pnames   []string
		pvalues  []string
		query    url.Values
		store    store
		echo     *Echo
	}

	store map[string]interface{}
)

// NewContext creates a Context object.
func NewContext(req engine.Request, res engine.Response, e *Echo) Context {
	return &context{
		request:  req,
		response: res,
		echo:     e,
		pvalues:  make([]string, *e.maxParam),
		store:    make(store),
	}
}

func (c *context) Deadline() (deadline time.Time, ok bool) {
	return
}

func (c *context) Done() <-chan struct{} {
	return nil
}

func (c *context) Err() error {
	return nil
}

func (c *context) Value(key interface{}) interface{} {
	return nil
}

// Request returns *http.Request.
func (c *context) Request() engine.Request {
	return c.request
}

// Response returns *Response.
func (c *context) Response() engine.Response {
	return c.response
}

// Socket returns *websocket.Conn.
func (c *context) Socket() *websocket.Conn {
	return c.socket
}

// Path returns the registered path for the handler.
func (c *context) Path() string {
	return c.path
}

// P returns path parameter by index.
func (c *context) P(i int) (value string) {
	l := len(c.pnames)
	if i < l {
		value = c.pvalues[i]
	}
	return
}

// Param returns path parameter by name.
func (c *context) Param(name string) (value string) {
	l := len(c.pnames)
	for i, n := range c.pnames {
		if n == name && i < l {
			value = c.pvalues[i]
			break
		}
	}
	return
}

// Query returns query parameter by name.
func (c *context) Query(name string) string {
	return c.request.URL().QueryValue(name)
}

// Form returns form parameter by name.
func (c *context) Form(name string) string {
	return c.request.FormValue(name)
}

// Get retrieves data from the context.
func (c *context) Get(key string) interface{} {
	return c.store[key]
}

// Set saves data in the context.
func (c *context) Set(key string, val interface{}) {
	if c.store == nil {
		c.store = make(store)
	}
	c.store[key] = val
}

// Bind binds the request body into specified type `i`. The default binder does
// it based on Content-Type header.
func (c *context) Bind(i interface{}) error {
	return c.echo.binder.Bind(c.request, i)
}

// Render renders a template with data and sends a text/html response with status
// code. Templates can be registered using `Echo.SetRenderer()`.
func (c *context) Render(code int, name string, data interface{}) (err error) {
	if c.echo.renderer == nil {
		return RendererNotRegistered
	}
	buf := new(bytes.Buffer)
	if err = c.echo.renderer.Render(buf, name, data); err != nil {
		return
	}
	c.response.Header().Set(ContentType, TextHTMLCharsetUTF8)
	c.response.WriteHeader(code)
	c.response.Write(buf.Bytes())
	return
}

// HTML sends an HTTP response with status code.
func (c *context) HTML(code int, html string) (err error) {
	c.response.Header().Set(ContentType, TextHTMLCharsetUTF8)
	c.response.WriteHeader(code)
	c.response.Write([]byte(html))
	return
}

// String sends a string response with status code.
func (c *context) String(code int, s string) (err error) {
	c.response.Header().Set(ContentType, TextPlainCharsetUTF8)
	c.response.WriteHeader(code)
	c.response.Write([]byte(s))
	return
}

// JSON sends a JSON response with status code.
func (c *context) JSON(code int, i interface{}) (err error) {
	b, err := json.Marshal(i)
	if c.echo.Debug() {
		b, err = json.MarshalIndent(i, "", "  ")
	}
	if err != nil {
		return err
	}
	return c.JSONBlob(code, b)
}

// JSONBlob sends a JSON blob response with status code.
func (c *context) JSONBlob(code int, b []byte) (err error) {
	c.response.Header().Set(ContentType, ApplicationJSONCharsetUTF8)
	c.response.WriteHeader(code)
	c.response.Write(b)
	return
}

// JSONP sends a JSONP response with status code. It uses `callback` to construct
// the JSONP payload.
func (c *context) JSONP(code int, callback string, i interface{}) (err error) {
	b, err := json.Marshal(i)
	if err != nil {
		return err
	}
	c.response.Header().Set(ContentType, ApplicationJavaScriptCharsetUTF8)
	c.response.WriteHeader(code)
	c.response.Write([]byte(callback + "("))
	c.response.Write(b)
	c.response.Write([]byte(");"))
	return
}

// XML sends an XML response with status code.
func (c *context) XML(code int, i interface{}) (err error) {
	b, err := xml.Marshal(i)
	if c.echo.Debug() {
		b, err = xml.MarshalIndent(i, "", "  ")
	}
	if err != nil {
		return err
	}
	return c.XMLBlob(code, b)
}

// XMLBlob sends a XML blob response with status code.
func (c *context) XMLBlob(code int, b []byte) (err error) {
	c.response.Header().Set(ContentType, ApplicationXMLCharsetUTF8)
	c.response.WriteHeader(code)
	c.response.Write([]byte(xml.Header))
	c.response.Write(b)
	return
}

// File sends a response with the content of the file. If `attachment` is set
// to true, the client is prompted to save the file with provided `name`,
// name can be empty, in that case name of the file is used.
func (c *context) File(path, name string, attachment bool) (err error) {
	dir, file := filepath.Split(path)
	if attachment {
		c.response.Header().Set(ContentDisposition, "attachment; filename="+name)
	}
	if err = c.echo.serveFile(dir, file, c); err != nil {
		c.response.Header().Del(ContentDisposition)
	}
	return
}

// NoContent sends a response with no body and a status code.
func (c *context) NoContent(code int) error {
	c.response.WriteHeader(code)
	return nil
}

// Redirect redirects the request using http.Redirect with status code.
func (c *context) Redirect(code int, url string) error {
	if code < http.StatusMultipleChoices || code > http.StatusTemporaryRedirect {
		return InvalidRedirectCode
	}
	// TODO: v2
	// http.Redirect(c.response, c.request, url, code)
	return nil
}

// Error invokes the registered HTTP error handler. Generally used by middleware.
func (c *context) Error(err error) {
	c.echo.httpErrorHandler(err, c)
}

// Logger returns the `Logger` instance.
func (c *context) Logger() logger.Logger {
	return c.echo.logger
}

// Object returns the `context` object.
func (c *context) Object() *context {
	return c
}

func (c *context) reset(req engine.Request, res engine.Response, e *Echo) {
	c.request = req
	c.response = res
	c.query = nil
	c.store = nil
	c.echo = e
}
