package responder

import "net/http"

type Option func(*Meta)

type PanicFn func(http.ResponseWriter, *http.Request, error)

func DefaultPanicFn(w http.ResponseWriter, r *http.Request, err error) {
	panic(err)
}
