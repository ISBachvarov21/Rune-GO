package runehttp

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	statusCode int
	header     http.Header
	body       []byte
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
