package main

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"nhooyr.io/websocket"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Hello from server")

		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"*"},
		})
		if err != nil {
			fmt.Println(err)

			return
		}
		fmt.Println("Accept")
		defer c.Close(websocket.StatusInternalError, "The sky is falling")

		ctx := r.Context()

		fmt.Println("Context")

		conn := websocket.NetConn(ctx, c, websocket.MessageBinary)
		defer conn.Close()

		fmt.Println("NetConn")

		fmt.Println(">>>>>")

		nconn, err := net.Dial("tcp", "localhost:22")
		if err != nil {
			fmt.Println("Error no dial")
			fmt.Println(err)

			return
		}

		fmt.Println(nconn)

		go func() {
			io.Copy(conn, nconn)
		}()
		go func() {
			io.Copy(nconn, conn)
		}()

		select {}
	})

	fmt.Println("Server started at port 8081")

	http.ListenAndServe(":8081", nil)
}
