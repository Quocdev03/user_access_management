package service

import (
	"net"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestLocalMailSMTP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	got := make(chan string, 1)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			got <- err.Error()
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		buf := make([]byte, 4096)
		_, _ = conn.Write([]byte("220 ok\r\n"))
		_, _ = conn.Read(buf) // EHLO
		_, _ = conn.Write([]byte("250 ok\r\n"))
		_, _ = conn.Read(buf) // MAIL
		_, _ = conn.Write([]byte("250 ok\r\n"))
		_, _ = conn.Read(buf) // RCPT
		_, _ = conn.Write([]byte("250 ok\r\n"))
		_, _ = conn.Read(buf) // DATA
		_, _ = conn.Write([]byte("354 go\r\n"))
		var body strings.Builder
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				body.Write(buf[:n])
				if strings.Contains(body.String(), "\r\n.\r\n") {
					break
				}
			}
			if err != nil {
				break
			}
		}
		_, _ = conn.Write([]byte("250 ok\r\n"))
		_, _ = conn.Read(buf)
		_, _ = conn.Write([]byte("221 bye\r\n"))
		got <- body.String()
	}()

	m := &localMail{host: "127.0.0.1", port: port, from: "a@b.c", logger: zap.NewNop()}
	if err := m.SendEmail("u@e.com", "Subj", "<p>x</p>"); err != nil {
		t.Fatal(err)
	}
	select {
	case raw := <-got:
		if !strings.Contains(raw, "Subject: Subj") || !strings.Contains(raw, "<p>x</p>") {
			t.Fatalf("bad payload: %q", raw)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}
