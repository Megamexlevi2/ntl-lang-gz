package std

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"lunex/internal/runtime"
	shared "lunex/internal/std/shared"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func httpHeadersToObj(h http.Header) *runtime.Value {
	obj := make(map[string]*runtime.Value)
	for k, vals := range h {
		if len(vals) > 0 {
			obj[strings.ToLower(k)] = runtime.StringVal(strings.Join(vals, ", "))
		}
	}
	return runtime.ObjectVal(obj)
}

func httpBuildResponse(w http.ResponseWriter, status int, body string, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	fmt.Fprint(w, body)
}

func httpRequestObj(r *http.Request, body string) *runtime.Value {
	queryObj := make(map[string]*runtime.Value)
	for k, vals := range r.URL.Query() {
		if len(vals) > 0 {
			queryObj[k] = runtime.StringVal(vals[0])
		}
	}
	paramsObj := make(map[string]*runtime.Value)
	return runtime.ObjectVal(map[string]*runtime.Value{
		"method":  runtime.StringVal(r.Method),
		"url":     runtime.StringVal(r.URL.String()),
		"path":    runtime.StringVal(r.URL.Path),
		"query":   runtime.ObjectVal(queryObj),
		"params":  runtime.ObjectVal(paramsObj),
		"headers": httpHeadersToObj(r.Header),
		"body":    runtime.StringVal(body),
		"ip":      runtime.StringVal(r.RemoteAddr),
		"host":    runtime.StringVal(r.Host),
	})
}

type httpServer struct {
	handler func(req *runtime.Value, res *runtime.Value) (*runtime.Value, error)
	mu      sync.Mutex
	routes  []httpRoute
}

type httpRoute struct {
	method  string
	pattern string
	handler *runtime.Value
	isUse   bool
}

func (s *httpServer) match(method, path string) (*runtime.Value, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.routes {
		if r.method != "" && r.method != method {
			continue
		}
		params, ok := matchRoute(r.pattern, path)
		if ok {
			return r.handler, params
		}
	}
	return nil, nil
}

func splitPathParts(s string) []string {
	trimmed := strings.Trim(s, "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

func matchRoute(pattern, path string) (map[string]string, bool) {
	if pattern == path {
		return map[string]string{}, true
	}
	pParts := splitPathParts(pattern)
	uParts := splitPathParts(path)
	if len(pParts) != len(uParts) {
		if len(pParts) == 0 || pParts[len(pParts)-1] != "*" {
			return nil, false
		}
	}
	params := make(map[string]string)
	for i, pp := range pParts {
		if pp == "*" {
			return params, true
		}
		if i >= len(uParts) {
			return nil, false
		}
		up := uParts[i]
		if strings.HasPrefix(pp, ":") {
			params[pp[1:]] = up
		} else if pp != up {
			return nil, false
		}
	}
	return params, true
}

func matchPrefix(pattern, path string) bool {
	if pattern == "/" {
		return true
	}
	prefix := strings.TrimRight(pattern, "/")
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func buildResObj(w http.ResponseWriter, r *http.Request) (resObj *runtime.Value, isSent func() bool) {
	sent := false
	pendingStatus := 0
	pendingHeaders := map[string]string{}

	flushHeaders := func() {
		for k, v := range pendingHeaders {
			w.Header().Set(k, v)
		}
		pendingHeaders = map[string]string{}
	}

	isSent = func() bool { return sent }

	var resMap map[string]*runtime.Value

	resMap = map[string]*runtime.Value{
		"json": runtime.FuncVal(&runtime.Function{Name: "json", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if sent {
				return runtime.Undefined, nil
			}
			sent = true
			status := 200
			if pendingStatus > 0 {
				status = pendingStatus
			}
			if len(a) > 1 && a[1].Tag == runtime.TypeNumber {
				status = int(a[1].ToNumber())
			}
			flushHeaders()
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(status)
			data := ""
			if len(a) > 0 {
				data = shared.ValueToJSON(a[0])
			}
			fmt.Fprint(w, data)
			return runtime.Undefined, nil
		}}),

		"send": runtime.FuncVal(&runtime.Function{Name: "send", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if sent {
				return runtime.Undefined, nil
			}
			sent = true
			status := 200
			if pendingStatus > 0 {
				status = pendingStatus
			}
			body := ""
			ct := "text/plain; charset=utf-8"
			if len(a) > 0 {
				if a[0].Tag == runtime.TypeObject || a[0].Tag == runtime.TypeArray {
					body = shared.ValueToJSON(a[0])
					ct = "application/json; charset=utf-8"
				} else {
					body = a[0].ToString()
					if strings.Contains(body, "<") && strings.Contains(body, ">") {
						ct = "text/html; charset=utf-8"
					}
				}
			}
			if len(a) > 1 && a[1].Tag == runtime.TypeNumber {
				status = int(a[1].ToNumber())
			}
			flushHeaders()
			httpBuildResponse(w, status, body, ct)
			return runtime.Undefined, nil
		}}),

		"text": runtime.FuncVal(&runtime.Function{Name: "text", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if sent {
				return runtime.Undefined, nil
			}
			sent = true
			status := 200
			if pendingStatus > 0 {
				status = pendingStatus
			}
			body := ""
			if len(a) > 0 {
				body = a[0].ToString()
			}
			if len(a) > 1 && a[1].Tag == runtime.TypeNumber {
				status = int(a[1].ToNumber())
			}
			flushHeaders()
			httpBuildResponse(w, status, body, "text/plain; charset=utf-8")
			return runtime.Undefined, nil
		}}),

		"html": runtime.FuncVal(&runtime.Function{Name: "html", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if sent {
				return runtime.Undefined, nil
			}
			sent = true
			status := 200
			if pendingStatus > 0 {
				status = pendingStatus
			}
			body := ""
			if len(a) > 0 {
				body = a[0].ToString()
			}
			if len(a) > 1 && a[1].Tag == runtime.TypeNumber {
				status = int(a[1].ToNumber())
			}
			flushHeaders()
			httpBuildResponse(w, status, body, "text/html; charset=utf-8")
			return runtime.Undefined, nil
		}}),

		"redirect": runtime.FuncVal(&runtime.Function{Name: "redirect", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if sent {
				return runtime.Undefined, nil
			}
			sent = true
			url := "/"
			status := 302
			if len(a) > 0 {
				url = a[0].ToString()
			}
			if len(a) > 1 && a[1].Tag == runtime.TypeNumber {
				status = int(a[1].ToNumber())
			}
			flushHeaders()
			http.Redirect(w, r, url, status)
			return runtime.Undefined, nil
		}}),

		"status": runtime.FuncVal(&runtime.Function{Name: "status", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if !sent && len(a) > 0 {
				pendingStatus = int(a[0].ToNumber())
			}
			return runtime.ObjectVal(resMap), nil
		}}),

		"setHeader": runtime.FuncVal(&runtime.Function{Name: "setHeader", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if !sent && len(a) >= 2 {
				pendingHeaders[a[0].ToString()] = a[1].ToString()
			}
			return runtime.ObjectVal(resMap), nil
		}}),

		"getHeader": runtime.FuncVal(&runtime.Function{Name: "getHeader", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(a) > 0 {
				return runtime.StringVal(w.Header().Get(a[0].ToString())), nil
			}
			return runtime.Null, nil
		}}),

		"removeHeader": runtime.FuncVal(&runtime.Function{Name: "removeHeader", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if !sent && len(a) > 0 {
				delete(pendingHeaders, a[0].ToString())
				w.Header().Del(a[0].ToString())
			}
			return runtime.ObjectVal(resMap), nil
		}}),

		"cookie": runtime.FuncVal(&runtime.Function{Name: "cookie", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if sent || len(a) < 2 {
				return runtime.ObjectVal(resMap), nil
			}
			name := a[0].ToString()
			value := a[1].ToString()
			cookie := &http.Cookie{Name: name, Value: value, Path: "/"}
			if len(a) > 2 && a[2].Tag == runtime.TypeObject {
				opts := a[2].ObjVal
				if v, ok := opts["maxAge"]; ok {
					cookie.MaxAge = int(v.ToNumber())
				}
				if v, ok := opts["path"]; ok {
					cookie.Path = v.ToString()
				}
				if v, ok := opts["domain"]; ok {
					cookie.Domain = v.ToString()
				}
				if v, ok := opts["secure"]; ok {
					cookie.Secure = v.Tag == runtime.TypeBool && v.BoolVal
				}
				if v, ok := opts["httpOnly"]; ok {
					cookie.HttpOnly = v.Tag == runtime.TypeBool && v.BoolVal
				}
				if v, ok := opts["sameSite"]; ok {
					switch strings.ToLower(v.ToString()) {
					case "strict":
						cookie.SameSite = http.SameSiteStrictMode
					case "lax":
						cookie.SameSite = http.SameSiteLaxMode
					case "none":
						cookie.SameSite = http.SameSiteNoneMode
					}
				}
			}
			http.SetCookie(w, cookie)
			return runtime.ObjectVal(resMap), nil
		}}),

		"clearCookie": runtime.FuncVal(&runtime.Function{Name: "clearCookie", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if !sent && len(a) > 0 {
				http.SetCookie(w, &http.Cookie{
					Name:    a[0].ToString(),
					Value:   "",
					Path:    "/",
					MaxAge:  -1,
					Expires: time.Unix(0, 0),
				})
			}
			return runtime.ObjectVal(resMap), nil
		}}),

		"end": runtime.FuncVal(&runtime.Function{Name: "end", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if !sent {
				sent = true
				status := 200
				if pendingStatus > 0 {
					status = pendingStatus
				}
				body := ""
				if len(a) > 0 {
					body = a[0].ToString()
				}
				flushHeaders()
				w.WriteHeader(status)
				if body != "" {
					fmt.Fprint(w, body)
				}
			}
			return runtime.Undefined, nil
		}}),
	}

	resObj = runtime.ObjectVal(resMap)
	return resObj, isSent
}

func serveStaticFile(w http.ResponseWriter, r *http.Request, dir string) bool {
	clean := filepath.Clean(r.URL.Path)
	if clean == "/" {
		clean = "/index.html"
	}
	fullPath := filepath.Join(dir, filepath.FromSlash(clean))

	rel, err := filepath.Rel(dir, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return false
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			indexPath := filepath.Join(fullPath, "index.html")
			info2, err2 := os.Stat(indexPath)
			if err2 != nil || info2.IsDir() {
				return false
			}
			fullPath = indexPath
		} else {
			return false
		}
	} else if info.IsDir() {
		indexPath := filepath.Join(fullPath, "index.html")
		info2, err2 := os.Stat(indexPath)
		if err2 != nil || info2.IsDir() {
			return false
		}
		fullPath = indexPath
	}

	ext := strings.ToLower(filepath.Ext(fullPath))
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		switch ext {
		case ".js":
			ct = "application/javascript; charset=utf-8"
		case ".css":
			ct = "text/css; charset=utf-8"
		case ".html", ".htm":
			ct = "text/html; charset=utf-8"
		case ".json":
			ct = "application/json; charset=utf-8"
		case ".svg":
			ct = "image/svg+xml"
		case ".ico":
			ct = "image/x-icon"
		case ".png":
			ct = "image/png"
		case ".jpg", ".jpeg":
			ct = "image/jpeg"
		case ".gif":
			ct = "image/gif"
		case ".woff":
			ct = "font/woff"
		case ".woff2":
			ct = "font/woff2"
		case ".ttf":
			ct = "font/ttf"
		case ".map":
			ct = "application/json"
		default:
			ct = "application/octet-stream"
		}
	}

	f, err := os.Open(fullPath)
	if err != nil {
		return false
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return false
	}

	w.Header().Set("Content-Type", ct)
	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
	return true
}

func httpServerVal(s *httpServer) *runtime.Value {
	var obj *runtime.Value

	serverMap := map[string]*runtime.Value{
		"listen": runtime.FuncVal(&runtime.Function{Name: "listen", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			port := 3000
			host := "0.0.0.0"
			var onStart *runtime.Value

			if len(args) > 0 {
				port = int(args[0].ToNumber())
			}
			if len(args) > 1 {
				if args[1].Tag == runtime.TypeString {
					host = args[1].StrVal
				} else if args[1].Tag == runtime.TypeFunction {
					onStart = args[1]
				}
			}
			if len(args) > 2 && args[2].Tag == runtime.TypeFunction {
				onStart = args[2]
			}

			addr := fmt.Sprintf("%s:%d", host, port)
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return runtime.Null, err
			}

			if onStart != nil && runtime.CallFunction != nil {
				runtime.CallFunction(onStart, []*runtime.Value{runtime.NumberVal(float64(port))})
			}

			srv := &http.Server{}

			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				bodyBytes, _ := io.ReadAll(r.Body)
				body := string(bodyBytes)
				reqObj := httpRequestObj(r, body)

				var routeHandler *runtime.Value
				var params map[string]string
				var middlewares []httpRoute

				s.mu.Lock()
				for _, route := range s.routes {
					if route.isUse {
						if matchPrefix(route.pattern, r.URL.Path) {
							middlewares = append(middlewares, route)
						}
						continue
					}
					if route.method != "" && route.method != r.Method {
						continue
					}
					p, ok := matchRoute(route.pattern, r.URL.Path)
					if ok {
						routeHandler = route.handler
						params = p
						break
					}
				}
				s.mu.Unlock()

				if params != nil {
					paramsObj := make(map[string]*runtime.Value)
					for k, v := range params {
						paramsObj[k] = runtime.StringVal(v)
					}
					reqObj.ObjVal["params"] = runtime.ObjectVal(paramsObj)
				}

				resObj, isSent := buildResObj(w, r)

				var callErr error
				var callRes *runtime.Value

				for _, mw := range middlewares {
					if isSent() {
						break
					}
					if mw.handler != nil && mw.handler.Tag == runtime.TypeObject {
						if sd, ok := mw.handler.ObjVal["staticDir"]; ok {
							if serveStaticFile(w, r, sd.StrVal) {
								return
							}
							continue
						}
					}
					if mw.handler != nil && mw.handler.Tag == runtime.TypeFunction && runtime.CallFunction != nil {
						callRes, callErr = runtime.CallFunction(mw.handler, []*runtime.Value{reqObj, resObj})
						_ = callErr
					}
				}

				if isSent() {
					return
				}

				if routeHandler != nil && runtime.CallFunction != nil {
					callRes, callErr = runtime.CallFunction(routeHandler, []*runtime.Value{reqObj, resObj})
				} else if s.handler != nil {
					callRes, callErr = s.handler(reqObj, resObj)
				}

				_ = callErr

				if !isSent() {
					if callRes != nil && callRes.Tag != runtime.TypeUndefined && callRes.Tag != runtime.TypeNull {
						if callRes.Tag == runtime.TypeString {
							ct := "text/plain; charset=utf-8"
							if strings.Contains(callRes.StrVal, "<") && strings.Contains(callRes.StrVal, ">") {
								ct = "text/html; charset=utf-8"
							}
							httpBuildResponse(w, 200, callRes.StrVal, ct)
						} else if callRes.Tag == runtime.TypeObject || callRes.Tag == runtime.TypeArray {
							httpBuildResponse(w, 200, shared.ValueToJSON(callRes), "application/json; charset=utf-8")
						} else {
							httpBuildResponse(w, 200, callRes.ToString(), "text/plain; charset=utf-8")
						}
					} else {
						httpBuildResponse(w, 404, `{"error":"not found"}`, "application/json; charset=utf-8")
					}
				}
			})

			srv.Handler = mux

			runtime.KeepAliveAdd()
			go func() {
				defer runtime.KeepAliveDone()
				srv.Serve(ln)
			}()

			return runtime.ObjectVal(map[string]*runtime.Value{
				"port": runtime.NumberVal(float64(port)),
				"close": runtime.FuncVal(&runtime.Function{Name: "close", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					srv.Shutdown(ctx)
					return runtime.Undefined, nil
				}}),
			}), nil
		}}),

		"get": runtime.FuncVal(&runtime.Function{Name: "get", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s.mu.Lock()
			pattern := ""
			if len(args) > 0 {
				pattern = args[0].ToString()
			}
			var handler *runtime.Value
			if len(args) > 1 {
				handler = args[1]
			}
			s.routes = append(s.routes, httpRoute{method: "GET", pattern: pattern, handler: handler})
			s.mu.Unlock()
			return obj, nil
		}}),

		"post": runtime.FuncVal(&runtime.Function{Name: "post", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s.mu.Lock()
			pattern := ""
			if len(args) > 0 {
				pattern = args[0].ToString()
			}
			var handler *runtime.Value
			if len(args) > 1 {
				handler = args[1]
			}
			s.routes = append(s.routes, httpRoute{method: "POST", pattern: pattern, handler: handler})
			s.mu.Unlock()
			return obj, nil
		}}),

		"put": runtime.FuncVal(&runtime.Function{Name: "put", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s.mu.Lock()
			pattern := ""
			if len(args) > 0 {
				pattern = args[0].ToString()
			}
			var handler *runtime.Value
			if len(args) > 1 {
				handler = args[1]
			}
			s.routes = append(s.routes, httpRoute{method: "PUT", pattern: pattern, handler: handler})
			s.mu.Unlock()
			return obj, nil
		}}),

		"patch": runtime.FuncVal(&runtime.Function{Name: "patch", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s.mu.Lock()
			pattern := ""
			if len(args) > 0 {
				pattern = args[0].ToString()
			}
			var handler *runtime.Value
			if len(args) > 1 {
				handler = args[1]
			}
			s.routes = append(s.routes, httpRoute{method: "PATCH", pattern: pattern, handler: handler})
			s.mu.Unlock()
			return obj, nil
		}}),

		"delete": runtime.FuncVal(&runtime.Function{Name: "delete", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s.mu.Lock()
			pattern := ""
			if len(args) > 0 {
				pattern = args[0].ToString()
			}
			var handler *runtime.Value
			if len(args) > 1 {
				handler = args[1]
			}
			s.routes = append(s.routes, httpRoute{method: "DELETE", pattern: pattern, handler: handler})
			s.mu.Unlock()
			return obj, nil
		}}),

		"all": runtime.FuncVal(&runtime.Function{Name: "all", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s.mu.Lock()
			pattern := ""
			if len(args) > 0 {
				pattern = args[0].ToString()
			}
			var handler *runtime.Value
			if len(args) > 1 {
				handler = args[1]
			}
			s.routes = append(s.routes, httpRoute{method: "", pattern: pattern, handler: handler})
			s.mu.Unlock()
			return obj, nil
		}}),

		"use": runtime.FuncVal(&runtime.Function{Name: "use", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s.mu.Lock()
			pattern := "/"
			var handler *runtime.Value
			if len(args) == 1 {
				handler = args[0]
			} else if len(args) >= 2 {
				pattern = args[0].ToString()
				handler = args[1]
			}
			if handler != nil {
				s.routes = append(s.routes, httpRoute{method: "", pattern: pattern, handler: handler, isUse: true})
			}
			s.mu.Unlock()
			return obj, nil
		}}),
	}

	obj = runtime.ObjectVal(serverMap)
	return obj
}

func HttpModule() *runtime.Value {
	newClient := func(timeoutMs float64) *http.Client {
		t := 30 * time.Second
		if timeoutMs > 0 {
			t = time.Duration(timeoutMs) * time.Millisecond
		}
		return &http.Client{Timeout: t}
	}

	doRequest := func(method, url string, opts *runtime.Value) (*runtime.Value, error) {
		var body io.Reader
		headers := map[string]string{}
		timeoutMs := float64(0)

		if opts != nil && opts.Tag == runtime.TypeObject {
			if b, ok := opts.ObjVal["body"]; ok && b != nil {
				var bodyStr string
				if b.Tag == runtime.TypeObject || b.Tag == runtime.TypeArray {
					bodyStr = shared.ValueToJSON(b)
					headers["Content-Type"] = "application/json"
				} else {
					bodyStr = b.ToString()
				}
				body = bytes.NewBufferString(bodyStr)
			}
			if h, ok := opts.ObjVal["headers"]; ok && h != nil && h.Tag == runtime.TypeObject {
				for k, v := range h.ObjVal {
					headers[k] = v.ToString()
				}
			}
			if t, ok := opts.ObjVal["timeout"]; ok && t != nil {
				timeoutMs = t.ToNumber()
			}
		}

		client := newClient(timeoutMs)

		req, err := http.NewRequest(method, url, body)
		if err != nil {
			return runtime.ObjectVal(map[string]*runtime.Value{
				"ok":    runtime.False,
				"error": runtime.StringVal(err.Error()),
			}), nil
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			return runtime.ObjectVal(map[string]*runtime.Value{
				"ok":    runtime.False,
				"error": runtime.StringVal(err.Error()),
			}), nil
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		bodyStr := string(respBody)
		ct := resp.Header.Get("Content-Type")

		var parsedBody *runtime.Value
		if strings.Contains(ct, "application/json") {
			parsed, err := shared.ParseJSON(bodyStr)
			if err == nil {
				parsedBody = parsed
			} else {
				parsedBody = runtime.StringVal(bodyStr)
			}
		} else {
			parsedBody = runtime.StringVal(bodyStr)
		}

		return runtime.ObjectVal(map[string]*runtime.Value{
			"ok":      runtime.BoolVal(resp.StatusCode >= 200 && resp.StatusCode < 300),
			"status":  runtime.NumberVal(float64(resp.StatusCode)),
			"headers": httpHeadersToObj(resp.Header),
			"body":    parsedBody,
			"text":    runtime.StringVal(bodyStr),
			"json": runtime.FuncVal(&runtime.Function{Name: "json", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				v, err := shared.ParseJSON(bodyStr)
				if err != nil {
					return runtime.Null, nil
				}
				return v, nil
			}}),
		}), nil
	}

	return runtime.ObjectVal(map[string]*runtime.Value{
		"request": runtime.FuncVal(&runtime.Function{Name: "request", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("http.request: method and url required")
			}
			var opts *runtime.Value
			if len(args) > 2 {
				opts = args[2]
			}
			return doRequest(args[0].ToString(), args[1].ToString(), opts)
		}}),

		"get": runtime.FuncVal(&runtime.Function{Name: "get", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("http.get: url required")
			}
			var opts *runtime.Value
			if len(args) > 1 {
				opts = args[1]
			}
			return doRequest("GET", args[0].ToString(), opts)
		}}),

		"post": runtime.FuncVal(&runtime.Function{Name: "post", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("http.post: url required")
			}
			var opts *runtime.Value
			if len(args) > 1 {
				opts = args[1]
			}
			return doRequest("POST", args[0].ToString(), opts)
		}}),

		"put": runtime.FuncVal(&runtime.Function{Name: "put", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("http.put: url required")
			}
			var opts *runtime.Value
			if len(args) > 1 {
				opts = args[1]
			}
			return doRequest("PUT", args[0].ToString(), opts)
		}}),

		"patch": runtime.FuncVal(&runtime.Function{Name: "patch", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("http.patch: url required")
			}
			var opts *runtime.Value
			if len(args) > 1 {
				opts = args[1]
			}
			return doRequest("PATCH", args[0].ToString(), opts)
		}}),

		"delete": runtime.FuncVal(&runtime.Function{Name: "delete", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("http.delete: url required")
			}
			var opts *runtime.Value
			if len(args) > 1 {
				opts = args[1]
			}
			return doRequest("DELETE", args[0].ToString(), opts)
		}}),

		"head": runtime.FuncVal(&runtime.Function{Name: "head", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("http.head: url required")
			}
			return doRequest("HEAD", args[0].ToString(), nil)
		}}),

		"createServer": runtime.FuncVal(&runtime.Function{Name: "createServer", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s := &httpServer{}
			if len(args) > 0 && args[0].Tag == runtime.TypeFunction {
				fn := args[0]
				s.handler = func(req *runtime.Value, res *runtime.Value) (*runtime.Value, error) {
					if runtime.CallFunction != nil {
						return runtime.CallFunction(fn, []*runtime.Value{req, res})
					}
					return runtime.Undefined, nil
				}
			}
			return httpServerVal(s), nil
		}}),

		"listen": runtime.FuncVal(&runtime.Function{Name: "listen", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("http.listen: server and port required")
			}
			server := args[0]
			if listen, ok := server.ObjVal["listen"]; ok {
				return runtime.CallFunction(listen, args[1:])
			}
			return runtime.Undefined, nil
		}}),

		"json": runtime.FuncVal(&runtime.Function{Name: "json", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Undefined, nil
			}
			res := args[0]
			data := args[1]
			status := 200
			if len(args) > 2 {
				status = int(args[2].ToNumber())
			}
			if jsonFn, ok := res.ObjVal["json"]; ok {
				return runtime.CallFunction(jsonFn, []*runtime.Value{data, runtime.NumberVal(float64(status))})
			}
			return runtime.Undefined, nil
		}}),

		"text": runtime.FuncVal(&runtime.Function{Name: "text", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Undefined, nil
			}
			res := args[0]
			if textFn, ok := res.ObjVal["text"]; ok {
				return runtime.CallFunction(textFn, args[1:])
			}
			return runtime.Undefined, nil
		}}),

		"html": runtime.FuncVal(&runtime.Function{Name: "html", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Undefined, nil
			}
			res := args[0]
			if htmlFn, ok := res.ObjVal["html"]; ok {
				return runtime.CallFunction(htmlFn, args[1:])
			}
			return runtime.Undefined, nil
		}}),

		"redirect": runtime.FuncVal(&runtime.Function{Name: "redirect", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Undefined, nil
			}
			res := args[0]
			if redFn, ok := res.ObjVal["redirect"]; ok {
				return runtime.CallFunction(redFn, args[1:])
			}
			return runtime.Undefined, nil
		}}),

		"parseBody": runtime.FuncVal(&runtime.Function{Name: "parseBody", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, nil
			}
			req := args[0]
			if body, ok := req.ObjVal["body"]; ok {
				if body.Tag == runtime.TypeString {
					v, err := shared.ParseJSON(body.StrVal)
					if err == nil {
						return v, nil
					}
					return body, nil
				}
				return body, nil
			}
			return runtime.Null, nil
		}}),

		"serveStatic": runtime.FuncVal(&runtime.Function{Name: "serveStatic", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			dir := "."
			if len(args) > 0 {
				dir = args[0].ToString()
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"staticDir": runtime.StringVal(dir),
			}), nil
		}}),

		"parseURL": runtime.FuncVal(&runtime.Function{Name: "parseURL", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ObjectVal(nil), nil
			}
			rawURL := args[0].ToString()
			protocol := ""
			host := ""
			path := rawURL
			query := ""
			if idx := strings.Index(rawURL, "://"); idx >= 0 {
				protocol = rawURL[:idx]
				rest := rawURL[idx+3:]
				if slash := strings.Index(rest, "/"); slash >= 0 {
					host = rest[:slash]
					path = rest[slash:]
				} else {
					host = rest
					path = "/"
				}
			}
			if idx := strings.Index(path, "?"); idx >= 0 {
				query = path[idx+1:]
				path = path[:idx]
			}
			queryObj := make(map[string]*runtime.Value)
			for _, part := range strings.Split(query, "&") {
				kv := strings.SplitN(part, "=", 2)
				if len(kv) == 2 && kv[0] != "" {
					queryObj[kv[0]] = runtime.StringVal(kv[1])
				}
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"protocol": runtime.StringVal(protocol),
				"host":     runtime.StringVal(host),
				"path":     runtime.StringVal(path),
				"query":    runtime.ObjectVal(queryObj),
				"search":   runtime.StringVal(query),
			}), nil
		}}),

		"buildURL": runtime.FuncVal(&runtime.Function{Name: "buildURL", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			base := args[0].ToString()
			if len(args) > 1 && args[1].Tag == runtime.TypeObject {
				params := []string{}
				for k, v := range args[1].ObjVal {
					params = append(params, urlEncode(k)+"="+urlEncode(v.ToString()))
				}
				if len(params) > 0 {
					if strings.Contains(base, "?") {
						base += "&" + strings.Join(params, "&")
					} else {
						base += "?" + strings.Join(params, "&")
					}
				}
			}
			return runtime.StringVal(base), nil
		}}),

		"encode": runtime.FuncVal(&runtime.Function{Name: "encode", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(urlEncode(args[0].ToString())), nil
		}}),

		"decode": runtime.FuncVal(&runtime.Function{Name: "decode", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(urlDecode(args[0].ToString())), nil
		}}),

		"statusText": runtime.FuncVal(&runtime.Function{Name: "statusText", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(http.StatusText(int(args[0].ToNumber()))), nil
		}}),
	})
}

func urlEncode(s string) string {
	var out strings.Builder
	for _, c := range s {
		switch {
		case (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~':
			out.WriteRune(c)
		default:
			b := []byte(string(c))
			for _, bb := range b {
				out.WriteString(fmt.Sprintf("%%%02X", bb))
			}
		}
	}
	return out.String()
}

func urlDecode(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			val, err := strconv.ParseInt(s[i+1:i+3], 16, 32)
			if err == nil {
				out.WriteRune(rune(val))
				i += 2
				continue
			}
		} else if s[i] == '+' {
			out.WriteByte(' ')
			continue
		}
		out.WriteByte(s[i])
	}
	return out.String()
}

func init() {
	_ = json.Marshal
}
