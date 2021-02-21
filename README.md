# `gows` (Go Websocket)
Websocket servers in Go (Golang) made simple!

`gows` is a high-level abstraction of a Websocket server that hides away all the boring an complicated details of dealing with a websocket connection. Under the hood, the `gorilla/websocket` package is used. When using `gows`, Ping/Pong is automatically handled and you can read and write messages with ease.

## Example
The following is an example of a websocket server that simply echoes whatever is being sent from the client:
```go
func echo(w http.ResponseWriter, r *http.Request) {
	ws, err := gows.Upgrade(w, r)
	if err != nil {
		// error handling...
	}

	for {
		msg := <-ws.Read()
		ws.WriteText(msg.Data)
	}
}
```