package runehttp

import (
	"net/http"

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
