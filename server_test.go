package wrpc

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	s := NewServer()
	s.Auth = func(r Req) bool {
		fmt.Println("auth:", r)
		return true
	}
	s.HandleFunc("echo", func(r Req) Resp {
		return Ret(StatusOK, r.Args...)
	})
	s.Alias("echo", "alias_echo")

	http.Handle("/wrpc", s)
	go http.ListenAndServe(":8112", nil)
	time.Sleep(time.Millisecond * 10)

	c := NewClient("http://localhost:8112/wrpc", "mofon")
	ret, err := c.Call(context.TODO(), "echo", "hello", "world")
	if err != nil {
		t.Error(err)
	}
	if ret.Status != StatusOK || ret.Rets[0] != "hello" || ret.Rets[1] != "world" {
		t.Error(ret)
	}

	ret, err = c.Call(context.TODO(), "alias_echo", "hello", "world")
	if err != nil {
		t.Error(err)
	}
	if ret.Status != StatusOK || ret.Rets[0] != "hello" || ret.Rets[1] != "world" {
		t.Error(ret)
	}

}
