package runehttp

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type Context struct {
	W  http.ResponseWriter
	R  *http.Request
	PS httprouter.Params
}

// Param returns the value of the named route parameter, e.g. "id" for "/users/:id".
func (c *Context) Param(name string) string {
	return c.PS.ByName(name)
}
