package runehttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/julienschmidt/httprouter"
)

type Context struct {
	w  http.ResponseWriter
	r  *http.Request
	ps httprouter.Params
}

// Param returns the value of the named route parameter, e.g. "id" for "/users/:id".
func (c *Context) Param(name string) string {
	return c.ps.ByName(name)
}

type Response struct {
	statusCode int
	header     http.Header
	body       []byte
}
type HTTPFunc func(*Context) (Response, error)

type HTTPHandler struct {
	router *httprouter.Router
}

func NewHTTPHandler() *HTTPHandler {
	return &HTTPHandler{router: httprouter.New()}
}

// todo: wrap to remove w from argument list
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *HTTPHandler) AddRoute(method string, path string, handler any) error {
	t := reflect.TypeOf(handler)

	if t.Kind() != reflect.Func {
		return fmt.Errorf("handler must be a function")
	}
	if reflect.ValueOf(handler).IsNil() {
		return fmt.Errorf("handler must not be nil")
	}

	// Only support:
	// func(...) (runehttp.Response, error)
	if t.NumOut() != 2 {
		return fmt.Errorf("handler must return two values")
	}

	// Supported:
	// func (*Context) (runehttp.Response, error)
	// func (Serializable) (runehttp.Response, error)
	// func (*Context, Serializable) (runehttp.Response, error)
	if t.NumIn() < 1 || t.NumIn() > 2 {
		return fmt.Errorf("handler must only accept one or two arguments")
	}

	fn := reflect.ValueOf(handler)

	firstArg := t.In(0)
	hasContext := firstArg == reflect.TypeFor[*Context]()

	var bodyType reflect.Type

	switch {
	case hasContext:
		if t.NumIn() == 2 {
			bodyType = t.In(1)
		}
	case t.NumIn() == 1:
		bodyType = firstArg
	default:
		return fmt.Errorf("first argument must be *Context or a serializable struct")
	}

	if bodyType != nil {
		if bodyType.Kind() == reflect.Pointer {
			if bodyType.Elem().Kind() != reflect.Struct {
				return fmt.Errorf("body argument must be a struct or a pointer to a struct")
			}
		} else if bodyType.Kind() != reflect.Struct {
			return fmt.Errorf("body argument must be a struct or a pointer to a struct")
		}
	}

	wrapped := func(c *Context) (Response, error) {
		args := make([]reflect.Value, 0, t.NumIn())

		if hasContext {
			args = append(args, reflect.ValueOf(c))
		}

		if bodyType != nil {
			var body reflect.Value

			if bodyType.Kind() == reflect.Pointer {
				body = reflect.New(bodyType.Elem())
			} else {
				body = reflect.New(bodyType)
			}

			if err := json.NewDecoder(c.r.Body).Decode(body.Interface()); err != nil {
				response := BadRequest(nil)
				return response, nil
			}

			if bodyType.Kind() != reflect.Pointer {
				body = body.Elem()
			}

			args = append(args, body)
		}

		values := fn.Call(args)
		return values[0].Interface().(Response), values[1].Interface().(error)
	}

	h.router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		c := &Context{w: w, r: r, ps: ps}

		resp, err := wrapped(c)

		if err != nil {
			resp = InternalServerError(nil)
		}

		for k, vs := range resp.header {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}

		w.WriteHeader(resp.statusCode)

		if resp.statusCode != 204 && resp.statusCode != 304 {
			w.Write(resp.body)
		}
	})

	return nil
}

func InternalServerError(body interface{}) Response {
	r := Response{statusCode: http.StatusInternalServerError}
	r.header = http.Header{}

	if body != nil {
		data, err := json.Marshal(body)
		if err == nil {
			r.body = data
			r.header.Add("Content-Type", "application/json")
		} else {
			r.body = []byte(err.Error())
			r.header.Add("Content-Type", "text/plain")
		}
	}
	return r
}

func BadRequest(body interface{}) Response {
	r := Response{statusCode: http.StatusBadRequest}
	r.header = http.Header{}

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return InternalServerError(err)

		}
		r.body = data
		r.header.Add("Content-Type", "application/json")
	}

	return r
}

func JSON(obj interface{}) Response {
	r := Response{}
	r.statusCode = http.StatusOK
	r.header = http.Header{}
	r.header.Add("content-type", "application/json")

	body, err := json.Marshal(obj)
	if err != nil {
		return BadRequest(err)
	}
	r.body = body

	return r
}

func Created(body interface{}) Response {
	r := Response{statusCode: http.StatusCreated}
	r.header = http.Header{}

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return InternalServerError(err)
		}
		r.body = data
		r.header.Add("content-type", "application/json")
	}

	return r
}

func NoContent() Response {
	r := Response{statusCode: http.StatusNoContent}
	return r
}

func OK(body interface{}) Response {
	r := Response{statusCode: http.StatusOK}
	r.header = http.Header{}
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return InternalServerError(err)
		}
		r.body = data
		r.header.Add("content-type", "application/json")
	}
	return r
}

func Unauthorized(body interface{}) Response {
	r := Response{statusCode: http.StatusUnauthorized}
	r.header = http.Header{}
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return InternalServerError(err)
		}
		r.body = data
		r.header.Add("content-type", "application/json")
	}
	return r
}

func Forbidden(body interface{}) Response {
	r := Response{statusCode: http.StatusForbidden}
	r.header = http.Header{}
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return InternalServerError(err)
		}
		r.body = data
		r.header.Add("content-type", "application/json")
	}
	return r
}

func NotFound(body interface{}) Response {
	r := Response{statusCode: http.StatusNotFound}
	r.header = http.Header{}
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return InternalServerError(err)
		}
		r.body = data
		r.header.Add("content-type", "application/json")
	}
	return r
}

func Conflict(body interface{}) Response {
	r := Response{statusCode: http.StatusConflict}
	r.header = http.Header{}
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return InternalServerError(err)
		}
		r.body = data
		r.header.Add("content-type", "application/json")
	}
	return r
}

func Text(s string) Response {
	r := Response{statusCode: http.StatusOK}
	r.header = http.Header{}
	r.header.Add("content-type", "text/plain; charset=utf-8")
	r.body = []byte(s)
	return r
}

func HTML(s string) Response {
	r := Response{statusCode: http.StatusOK}
	r.header = http.Header{}
	r.header.Add("content-type", "text/html; charset=utf-8")
	r.body = []byte(s)
	return r
}

func Redirect(url string) Response {
	r := Response{statusCode: http.StatusFound}
	r.header = http.Header{}
	r.header.Add("Location", url)
	return r
}

// Custom is a generic escape hatch for responses that don't fit the
// standard constructors (file downloads, CSV exports, etc.).
// contentType may be empty if there is no body.
func Custom(statusCode int, contentType string, body []byte) Response {
	r := Response{statusCode: statusCode}
	r.header = http.Header{}
	if contentType != "" {
		r.header.Add("Content-Type", contentType)
	}
	r.body = body
	return r
}
