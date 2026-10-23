package runehttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"slices"

	"github.com/julienschmidt/httprouter"
)

type HTTPFunc func(*Context) (Response, error)

type HTTPHandler struct {
	router *httprouter.Router
}

func NewHTTPHandler() *HTTPHandler {
	return &HTTPHandler{router: httprouter.New()}
}

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
	//if t.NumIn() < 1 || t.NumIn() > 2 {
	//	return fmt.Errorf("handler must only accept one or two arguments")
	//}

	hasContext := false
	contextIndex := -1
	hasBody := false
	bodyIndex := -1
	var bodyType reflect.Type

	for i := 0; i < t.NumIn(); i++ {
		arg := t.In(i)

		if arg == reflect.TypeFor[*Context]() {
			hasContext = true
			contextIndex = i
		} else if arg.Kind() == reflect.Struct {
			hasBody = true
			bodyIndex = i
			bodyType = arg
		} else if arg.Kind() == reflect.Ptr && arg.Elem().Kind() == reflect.Struct {
			hasBody = true
			bodyIndex = i
			bodyType = arg
		} else {
			return fmt.Errorf("handler argument %d must be *Context or a struct type", i)
		}
	}

	fn := reflect.ValueOf(handler)

	wrapped := func(c *Context) (Response, error) {
		args := make([]reflect.Value, 0, t.NumIn())

		if hasContext {
			args = slices.Insert(args, contextIndex, reflect.ValueOf(c))
		}

		if hasBody {
			var body reflect.Value

			if bodyType.Kind() == reflect.Pointer {
				body = reflect.New(bodyType.Elem())
			} else {
				body = reflect.New(bodyType)
			}

			if err := json.NewDecoder(c.R.Body).Decode(body.Interface()); err != nil {
				response := BadRequest(nil)
				return response, nil
			}

			if bodyType.Kind() != reflect.Pointer {
				body = body.Elem()
			}

			args = slices.Insert(args, bodyIndex, body)
		}

		values := fn.Call(args)
		respInterface := values[0].Interface()
		if respInterface == nil {
			panic("handler must return a Response")
		}
		resp := respInterface.(Response)
		errInterface := values[1].Interface()
		if errInterface != nil {
			err := errInterface.(error)
			return resp, err
		}
		return resp, nil
	}

	h.router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		c := &Context{W: w, R: r, PS: ps}

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
