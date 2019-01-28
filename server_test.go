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

	http.Handle("/wrpc", s)
	go http.ListenAndServe(":8112", nil)
	time.Sleep(time.Second)

	c := NewClient("http://localhost:8112/wrpc", "mofon")
	ret, err := c.Call(context.TODO(), "echo", "hello", "world")
	if err != nil {
		t.Error(err)
	}

	fmt.Println(ret)
}
