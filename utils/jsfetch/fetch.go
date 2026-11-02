//go:build tinygo && js

package jsfetch

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"syscall/js"
)

// TinyGo's net/http has no js fetch transport; this is the subset the data loader needs.
type fetchTransport struct{}

func (fetchTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	init := js.Global().Get("Object").New()
	init.Set("method", req.Method)
	headers := js.Global().Get("Object").New()
	for k, v := range req.Header {
		if len(v) > 0 {
			headers.Set(k, v[0])
		}
	}
	init.Set("headers", headers)

	type result struct {
		res *http.Response
		err error
	}
	done := make(chan result, 1)
	var onBody, onRes, onErr js.Func
	var status int
	var header http.Header
	onErr = js.FuncOf(func(_ js.Value, args []js.Value) any {
		msg := "fetch failed"
		if len(args) > 0 {
			msg = args[0].Call("toString").String()
		}
		done <- result{err: errors.New(msg)}
		return nil
	})
	onBody = js.FuncOf(func(_ js.Value, args []js.Value) any {
		u8 := js.Global().Get("Uint8Array").New(args[0])
		b := make([]byte, u8.Get("length").Int())
		js.CopyBytesToGo(b, u8)
		done <- result{res: &http.Response{
			Status:        http.StatusText(status),
			StatusCode:    status,
			Header:        header,
			Body:          io.NopCloser(bytes.NewReader(b)),
			ContentLength: int64(len(b)),
			Request:       req,
		}}
		return nil
	})
	onRes = js.FuncOf(func(_ js.Value, args []js.Value) any {
		r := args[0]
		status = r.Get("status").Int()
		header = http.Header{}
		if ct := r.Get("headers").Call("get", "content-type"); ct.Type() == js.TypeString {
			header.Set("Content-Type", ct.String())
		}
		r.Call("arrayBuffer").Call("then", onBody, onErr)
		return nil
	})
	js.Global().Call("fetch", req.URL.String(), init).Call("then", onRes, onErr)
	r := <-done
	onBody.Release()
	onRes.Release()
	onErr.Release()
	return r.res, r.err
}

func init() {
	http.DefaultClient.Transport = fetchTransport{}
}
