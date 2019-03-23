package main

import (
	"flag"
	"fmt"
	"github.com/andrew-d/go-termutil"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

func usage() string {
	return "usage: stun [local port]:[remote address]:[remote port]... [ssh server address]:[port]"
}

type session struct {
	local      string
	remoteAddr string
	listener   net.Listener
}

func main() {
	var methods []ssh.AuthMethod
	password := flag.String("pwd", "", "password")

	flag.Parse()

	if len(os.Args) < 3 {
		fmt.Println(usage())
		return
	}
	fmt.Println(*password)
	os.Args = os.Args[1:]

	var sessions []session
	var sshServer string
	var username string
	for i, arg := range os.Args {
		if i == len(os.Args)-1 {
			ss := strings.Split(arg, "@")
			if len(ss) == 2 {
				username = ss[0]
				sshServer = ss[1]
			} else {
				sshServer = arg
			}
		} else {
			ss := strings.Split(arg, ":")
			sessions = append(sessions, session{":" + ss[0], strings.Join(ss[1:], ":"), nil})
		}
	}

	if username == "" {
		fmt.Print("enter your name:")
		var s string
		_, err := fmt.Scanln(&s)
		if err != nil {
			fmt.Println(err)
			return
		}
		username = s
	}
	if *password == "" {
		b, err := termutil.GetPass("enter your password:", os.Stdout.Fd(), os.Stdin.Fd())
		if err != nil {
			fmt.Println(err)
			return
		}
		*password = string(b)
	}

	methods = append(methods, ssh.Password(*password))

	sshconfig := &ssh.ClientConfig{
		User: username,
		Auth: methods,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		BannerCallback: func(message string) error {
			fmt.Println(message)
			return nil
		},
	}

	sshClient, err := ssh.Dial("tcp", sshServer+":22", sshconfig)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Println("connect ssh server successfully")
	for _, s := range sessions {
		l, err := net.Listen("tcp", s.local)
		if err != nil {
			log.Printf("listen error:%s", err)
			continue
		}
		s.listener = l
		go tunnel(s, sshClient)
		log.Printf("tunnel %s -> %s built", s.local, s.remoteAddr)
	}

	select {}
}

func tunnel(s session, client *ssh.Client) {

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("accept error:%s", err)
		}
		go func() {
			remote, err := client.Dial("tcp", s.remoteAddr)
			if err != nil {
				log.Printf("dial %s error", s.remoteAddr)
				return
			}
			log.Printf("%s -> %s connected.", conn.RemoteAddr(), s.remoteAddr)
			go func() {
				io.Copy(remote, conn)
				remote.Close()
			}()
			io.Copy(conn, remote)
			conn.Close()
		}()
	}
}
