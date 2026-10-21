# Rune-GO
## What is it?
Rune-GO is a Golang rewrite of my (relatively) old C++ HTTP server framework, Rune. It focuses on a good developer experience through a high-level HTTP API.
## What does it feature?
This framework features a hot reload mechanism of the server, automatic body serialization into structs, request responses using returns, and an automatic route registration feature via code generation
## What is it NOT?
This is not a framework tuned for execution speed or a low memory footprint, although it uses [julienschmidt's httprouter](https://github.com/julienschmidt/httprouter) for fast routing.

## Syntax examples
Enough talking general things, let's look at where this framework shines - the developer experience it provides.
```go
// RUNE GET /
func GetRoot(c *runehttp.Context) (runehttp.Response, error) {
  return runehttp.Text("Hello World!"), nil
}
```

```go
type Serializable struct {
  SerializableField string `json:"sf"`
}

// RUNE POST /serializable
func PostSerializable(s Serializable) (runehttp.Response, error) {
  fmt.Println(s.SerializableField)
  return runehttp.OK(nil), nil
}
```

```go
// RUNE POST /post
func CreatePost(c *runehttp.Context, p PostIM) (runehttp.Response, error) {
  // Accessing original http.Request to get client authorization token from headers
  token := c.R.Header.Get("Authorization")

  // Authorize client using the token

  // Validate post fields

  // Register post in db
  postEntry, err := db.Posts.CreatePost(p)

  if err != nil {
    details := struct {
      Message string `json:"message"`
      Error string `json:"error"`
    }{
      Message: "Error during post creation. Try again later.",
      Error: err.Error(),
    }
	// Return InternalServerError with a json body containing the error details
    return runehttp.InternalServerError(details), nil
  }

  return runehttp.JSON(postEntry.(PostVM)), nil
}
```
As you see by these examples, several aspects of the normal Golang HTTP API have been abstracted. Let's go over them one by one.

## Endpoint registration
One registers an endpoint by giving the respective endpoint handler a comment containing a RUNE directive of the following syntax: `RUNE <METHOD> <PATH>`. Doesn't matter where in the comment it is, could be at the start, could be at the end, could be somewhere in the middle, go wild.

## Accessing original HTTP request, writer and julienschmidt/httprouter's request params
Accessing any of the aforementioned objects is done through the *runehttp.Context argument. The original request is the `R *http.Request` field, the writer is the `W http.ResponseWriter` field, and the request params are in the `PS httprouter.Params` field.

## Accessing request body
You can access the request body as is from the context, but in case you don't want to parse JSON into a struct manually, you can just include said struct in the endpoint arguments. Doing so creates a value of the specified struct type, deserializes the JSON body into it, and passes that value to the handler

## Returning a response to the client
The whole http.ResponseWriter way of returning a response is outdated. Why not `return` a response and a nillable error instead? Constructing a response can be done using the response constructor helpers, such as runehttp.OK(), runehttp.JSON(), etc., etc.
If error is not nil then the HTTP handler returns an InternalServerError with an empty body.

