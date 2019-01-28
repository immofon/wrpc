package wrpc

import (
	"io/ioutil"
	"net/http"
	"strings"
)

type Status string

const (
	StatusOK Status = "ok" // ok,have rets for details.

	// err with message,ret [code,message], and message could be read by user
	StatusError Status = "err"

	StatusInternalServerError Status = "ierr" // internal server error,no rets.
	StatusAuth                Status = "auth" // have to auth first,no rets.
	StatusBan                 Status = "ban"  // be forbidden access,no rets.
)

type Req struct {
	Token  string
	Method string
	Args   []string
}

func Ret(s Status, rets ...string) Resp {
	return Resp{
		Status: s,
		Rets:   rets,
	}
}

type Resp struct {
	Status Status
	Rets   []string
}

func (ret Resp) WriteTo(w http.ResponseWriter) {
	switch ret.Status {
	case StatusOK:
		w.WriteHeader(http.StatusOK)
	case StatusError:
		w.WriteHeader(http.StatusBadRequest)
	case StatusAuth:
		w.WriteHeader(http.StatusUnauthorized)
	case StatusBan:
		w.WriteHeader(http.StatusForbidden)
	case StatusInternalServerError:
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	if len(ret.Rets) == 0 {
		w.Write([]byte(ret.Status))
	} else {
		data := strings.Join([]string{string(ret.Status), strings.Join(ret.Rets, "\x1F")}, "\x1F")
		w.Write([]byte(data))
	}
}

type AuthFunc func(Req) bool

type Handler interface {
	WrpcCall(Req) Resp
}

type HandleFunc func(Req) Resp

func (fn HandleFunc) WrpcCall(r Req) Resp {
	return fn(r)
}

type Server struct {
	// read-only after init stage
	handlers map[string]Handler

	// read-only
	Auth             AuthFunc
	MaxContentLength int64
}

const DefaultMaxContentLength = 65535

func NewServer() *Server {
	return &Server{
		handlers: make(map[string]Handler),

		Auth:             func(_ Req) bool { return true },
		MaxContentLength: DefaultMaxContentLength,
	}
}

func (s *Server) Handler(method string, handler Handler) {
	if handler == nil {
		panic("handler should not nil")
	}

	s.handlers[method] = handler
}
func (s *Server) HandleFunc(method string, fn HandleFunc) {
	if fn == nil {
		panic("handleFunc should not nil")
	}

	s.Handler(method, fn)
}

// protocal:
// POST
// -> token|method{|args}
// <- status{|rets}
// tip: '|' represents '\x1F'
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// the method must be post
	if r.Method != "POST" {
		Ret(StatusInternalServerError).WriteTo(w)
		return
	}

	// MUST set ContentLength and (TODO) Less than MaxContentLength
	if r.ContentLength > s.MaxContentLength {
		Ret(StatusInternalServerError).WriteTo(w)
		return

	}

	raw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Ret(StatusInternalServerError).WriteTo(w)
		return

	}

	data := strings.Split(string(raw), "\x1F")
	var token string
	var method string
	var args []string
	if len(data) > 1 {
		token = data[0]
		method = data[1]
	} else {
		Ret(StatusInternalServerError).WriteTo(w)
		return
	}
	if len(data) > 2 {
		args = data[2:]
	}

	handler := s.handlers[method]
	if handler == nil {
		Ret(StatusInternalServerError).WriteTo(w)
		return

	}

	req := Req{
		Token:  token,
		Method: method,
		Args:   args,
	}
	if !s.Auth(req) {
		Ret(StatusAuth).WriteTo(w)
		return
	}

	ret := handler.WrpcCall(req)
	ret.WriteTo(w)
}
