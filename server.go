package wrpc

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

const UnitSep = "\x1F"

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

func (r Req) Encode() string {
	if len(r.Args) == 0 {
		return strings.Join([]string{r.Token, r.Method}, UnitSep)
	}
	return strings.Join([]string{r.Token, r.Method,
		strings.Join(r.Args, UnitSep),
	}, UnitSep)
}

type Resp struct {
	Status Status
	Rets   []string
}

func Ret(s Status, rets ...string) Resp {
	return Resp{
		Status: s,
		Rets:   rets,
	}
}

func (resp Resp) OK() bool {
	return resp.Status == StatusOK
}
func (resp Resp) Error(err error, expect_len int) error {
	if err != nil {
		return err
	}

	if expect_len < 0 {
		expect_len = len(resp.Rets)
	}

	if resp.OK() {
		if len(resp.Rets) == expect_len {
			return nil
		} else {
			return errors.New("error: [wrpc|rets.length]")
		}
	}

	e := strings.Join(resp.Rets, " ")
	return errors.New(fmt.Sprintf("error: [wrpc|%s] %s", string(resp.Status), e))
}

func (ret Resp) WriteTo(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	if len(ret.Rets) == 0 {
		io.Copy(w, strings.NewReader(string(ret.Status)))
	} else {
		data := strings.Join([]string{string(ret.Status), strings.Join(ret.Rets, UnitSep)}, UnitSep)
		io.Copy(w, strings.NewReader(data))
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

type ServerStatus struct {
	Count int64
}
type ServerStatusFunc func(*ServerStatus)

type Server struct {
	// statistics
	status    *ServerStatus         // read/write via status_ch
	status_ch chan ServerStatusFunc // never close

	// read-only after init stage
	handlers map[string]Handler

	// read-only
	Auth             AuthFunc
	MaxContentLength int64
}

const DefaultMaxContentLength = 65535
const defaultServerStatusChanSize = 100

func NewServer() *Server {
	s := &Server{
		status:    &ServerStatus{Count: 0},
		status_ch: make(chan ServerStatusFunc, defaultServerStatusChanSize),

		handlers: make(map[string]Handler),

		Auth:             func(_ Req) bool { return true },
		MaxContentLength: DefaultMaxContentLength,
	}
	go s.statisticsLoop()

	return s
}

func (s *Server) statisticsLoop() {
	for fn := range s.status_ch {
		fn(s.status)
	}
}

func (s *Server) Status() ServerStatus {
	ch := make(chan ServerStatus, 1)
	defer close(ch)
	s.status_ch <- func(ss *ServerStatus) {
		ch <- *ss
	}
	return <-ch
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
// tip: '|' represents UnitSep '\x1F'
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// the method must be post
	if r.Method != "POST" {
		Ret(StatusInternalServerError, "method").WriteTo(w)
		return
	}

	// MUST set ContentLength and (TODO) Less than MaxContentLength
	if r.ContentLength > s.MaxContentLength {
		Ret(StatusInternalServerError, "content length").WriteTo(w)
		return
	}

	raw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Ret(StatusInternalServerError, "read body").WriteTo(w)
		return

	}

	data := strings.Split(string(raw), UnitSep)
	var token string
	var method string
	var args []string
	if len(data) > 1 {
		token = data[0]
		method = data[1]
	} else {
		Ret(StatusInternalServerError, "encode request").WriteTo(w)
		return
	}
	if len(data) > 2 {
		args = data[2:]
	}

	ret := s.Call(Req{
		Token:  token,
		Method: method,
		Args:   args,
	})
	ret.WriteTo(w)

	s.status_ch <- func(ss *ServerStatus) {
		ss.Count++
	}
}

func (s *Server) Alias(method, alias string) {
	h := s.handlers[method]
	if h == nil {
		panic("wrpc.alias: not registed method:" + method)
	}

	s.handlers[alias] = h
}

func (s *Server) Call(r Req) Resp {
	if !s.Auth(r) {
		return Ret(StatusAuth)
	}

	return s.CallWithoutAuth(r)
}
func (s *Server) CallWithoutAuth(r Req) (resp Resp) {
	defer func() {
		if r := recover(); r != nil {
			resp = Ret(StatusInternalServerError, "panic")
		}
	}()

	handler := s.handlers[r.Method]
	if handler == nil {
		return Ret(StatusInternalServerError, "not found method")
	}

	return handler.WrpcCall(r)
}
