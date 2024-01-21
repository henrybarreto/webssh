package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"syscall/js"

	"golang.org/x/crypto/ssh"
	"nhooyr.io/websocket"
)

func connect(ctx context.Context, server string) (net.Conn, error) {
	fmt.Println("Connecting to", server)
	wconn, _, err := websocket.Dial(ctx, server, nil)
	fmt.Println("Dial")
	if err != nil {
		fmt.Println("Error dialing websocket", err)

		return nil, err
	}

	return websocket.NetConn(ctx, wconn, websocket.MessageBinary), nil
}

func login(ctx context.Context, conn net.Conn, username string, password string) (*ssh.Session, error) {
	sconn, chs, reqs, err := ssh.NewClientConn(conn, "tcp", &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})

	fmt.Println("NewClientConn")

	if err != nil {
		fmt.Println("Error creating ssh client", err)

		return nil, err
	}

	cli := ssh.NewClient(sconn, chs, reqs)

	return cli.NewSession()
}

func main() {
	fmt.Println("Started")

	ctx := context.Background()

	conn, err := connect(ctx, "ws://localhost:8081/")
	if err != nil {
		fmt.Println(err)

		return
	}
	defer conn.Close()

	sess, err := login(ctx, conn, "root", "root")
	if err != nil {
		fmt.Println(err)

		return
	}
	defer sess.Close()

	backend := js.Global().Get("backend")
	fmt.Println(backend)

	if err := sess.RequestPty("xterm", 40, 120, ssh.TerminalModes{}); err != nil {
		sess.Close()
		fmt.Println(err)
		fmt.Println("Error requesting pty")

		return
	}

	stdin, err := sess.StdinPipe()
	if err != nil {
		fmt.Println(err)

		return
	}

	stdout, err := sess.StdoutPipe()
	if err != nil {
		fmt.Println(err)

		return
	}

	stderr, err := sess.StderrPipe()
	if err != nil {
		fmt.Println(err)

		return
	}

	stdcombined := io.MultiReader(stdout, stderr)

	if err := sess.Shell(); err != nil {
		fmt.Println(err)

		return
	}

	backend.Set("resizeCallback", js.FuncOf(func(this js.Value, args []js.Value) any {
		sess.WindowChange(args[0].Int(), args[1].Int())

		return nil
	}))

	backend.Set("writeCallback", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		fmt.Println("emitData")

		stdin.Write([]byte(args[0].String()))

		return nil
	}))

	go func() {
		// https://datatracker.ietf.org/doc/html/rfc4253#section-6.1
		buffer := make([]byte, 35000)

		for {
			fmt.Println("Reading")

			n, err := stdcombined.Read(buffer)
			fmt.Println(n)
			if err != nil {
				fmt.Println(err)

				return
			}

			backend.Call("onDataCallback", string(buffer[:n]))
		}
	}()

	// Prevent the program from exiting.
	// Note: the exported func should be released if you don't need it any more,
	// and let the program exit after then. To simplify this demo, this is
	// omitted. See https://pkg.go.dev/syscall/js#Func.Release for more information.
	select {}
}
