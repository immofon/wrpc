package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-gomail/gomail"
	"github.com/immofon/wrpc"
)

func main() {
	cmd := os.Getenv("cmd")
	switch cmd {
	case "daemon":
		daemon()
	case "watch":
		watch()
	case "test":
		test()
	default:
		client()
	}
}

func watch() {
	c := wrpc.NewClient("http://localhost:8112/wrpc", "mofon")

	for {
		resp, err := c.Call(context.TODO(), "status/count")
		if err != nil {
			panic(err)
		}

		fmt.Println(resp.Rets[0])
		time.Sleep(time.Second)
	}
}
func test() {
	panic("TODO")
}
func client() {
	panic("TODO")
}

func daemon() {
	s := wrpc.NewServer()
	s.Auth = func(r wrpc.Req) bool {
		fmt.Printf("[Auth] :%s: %s\n", r.Method, r.Token)
		return true
	}
	s.HandleFunc("status/count", func(r wrpc.Req) wrpc.Resp {
		ss := s.Status()
		return wrpc.Ret(wrpc.StatusOK, strconv.FormatInt(ss.Count, 10))
	})

	s.HandleFunc("time", func(r wrpc.Req) wrpc.Resp {
		return wrpc.Ret(wrpc.StatusOK, time.Now().String())
	})
	s.HandleFunc("time/unix", func(r wrpc.Req) wrpc.Resp {
		return wrpc.Ret(wrpc.StatusOK, fmt.Sprint(time.Now().Unix()))
	})

	// mail/notify: to type("") body(text)
	s.HandleFunc("mail/notify", func(r wrpc.Req) wrpc.Resp {
		if len(r.Args) != 3 {
			return wrpc.Ret(wrpc.StatusError, "args")
		}

		var (
			to    = r.Args[0]
			ntype = r.Args[1]
			body  = r.Args[2]
		)

		err := SendMailNotify(to, ntype, body)
		if err != nil {
			return wrpc.Ret(wrpc.StatusInternalServerError)
		}
		return wrpc.Ret(wrpc.StatusOK)
	})

	http.Handle("/wrpc", s)
	fmt.Println("listen: http://localhost:8112/wrpc")
	http.ListenAndServe(":8112", nil)
}

func SendMailNotify(to, type_ string, body string) error {
	m := gomail.NewMessage()

	from := "immofon@163.com"
	m.SetHeader("From", from)
	m.SetHeader("To", to)

	m.SetHeader("Subject", fmt.Sprintf("[通知] %s", type_))
	m.SetBody("text/plain", body)

	username := os.Getenv("MAIL_USERNAME")
	password := os.Getenv("MAIL_PASSWORD")
	d := gomail.NewDialer("smtp.163.com", 465, username, password)

	// Send the email to Bob, Cora and Dan.
	return d.DialAndSend(m)
}
