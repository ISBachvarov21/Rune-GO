package httphandler

import (
	`encoding/json`
	`fmt`
	"net/http"
	`reflect`
)

type Context struct {
	w http.ResponseWriter
	r *http.Request
}

type Response struct {
	statusCode int
}

type HTTPFunc func(*Context) (Response, error)

type HTTPHandler struct {
	routes map[string]map[string]HTTPFunc
}

// todo: wrap to remove w from argument list
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// search for handler inside route->method
	contextMethods, ok := h.routes[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}

	handler, ok := contextMethods[r.Method]
	if !ok {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := &Context{w: w, r: r}

	// todo: handle errors
	handler(ctx)

	// todo: if not a match try searching for a match in dynamic paths
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
	// func(...) runehttp.Response
	if t.NumOut() != 1 {
		return fmt.Errorf("handler must return one value")
	}

	// Supported:
	// func (*Context) Response
	// func (Serializable) Response
	// func (*Context, Serializable) Response
	if t.NumIn() < 1 || t.NumIn() > 2 {
		return fmt.Errorf("handler must only accept one or two arguments")
	}

	fn := reflect.ValueOf(handler)

	firstArg := t.In(0)
	hasContext := firstArg != reflect.TypeOf((*Context)(nil))

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
		if bodyType.Kind() == reflect.Ptr {
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

			if bodyType.Kind() == reflect.Ptr {
				body = reflect.New(bodyType.Elem())
			} else {
				body = reflect.New(bodyType)
			}

			if err := json.NewDecoder(c.r.Body).Decode(body.Interface()); err != nil {
				http.Error(
					c.w,
					"Invalid JSON body",
					http.StatusBadRequest,
				)
				response := Response{}
				return
			}

		}

	}

	if h.routes == nil {
		h.routes = map[string]map[string]http.HandlerFunc{}
	}
	if h.routes[path] == nil {
		h.routes[path] = map[string]http.HandlerFunc{}
	}
	h.routes[path][method] = handler
}
