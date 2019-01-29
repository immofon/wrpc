package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-gomail/gomail"
	"github.com/immofon/wrpc"
)

func main() {
	s := wrpc.NewServer()
	s.Auth = func(r wrpc.Req) bool {
		fmt.Printf("[Auth] :%s: %s\n", r.Method, r.Token)
		return true
	}
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
