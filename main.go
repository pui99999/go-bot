package main

import (
	"flag"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"os"
	"os/signal"
)

type JsonRPC2 struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Result  interface{} `json:"result"`
}

type SubscribeParams struct {
	Channel string `json:"channel"`
}

type ExecutionsMessage struct {
	Channel string      `json:"channel"`
	Message []Execution `json:"message"`
}

type Execution struct {
	ID                         int     `json:"id"`
	Side                       string  `json:"side"`
	Price                      int     `json:"price"`
	Size                       float64 `json:"size"`
	ExecDate                   string  `json:"exec_date"`
	BuyChildOrderAcceptanceID  string  `json:"buy_child_order_acceptance_id"`
	SellChildOrderAcceptanceID string  `json:"sell_child_order_acceptance_id"`
}

var addr = flag.String("addr", "ws.lightstream.bitflyer.com", "https service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "wss", Host: *addr, Path: "/json-rpc"}
	log.Printf("connecting to %s", u.String())

	// note: reconnection handling needed.
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	if err := c.WriteJSON(&JsonRPC2{Version: "2.0", Method: "subscribe", Params: &SubscribeParams{"lightning_executions_FX_BTC_JPY"}}); err != nil {
		log.Fatal("subscribe:", err)
		return
	}

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			message := new(JsonRPC2)
			if err := c.ReadJSON(message); err != nil {
				log.Println("read:", err)
				return
			}

			if message.Method == "channelMessage" {
				message := message.Params.(map[string]interface{})["message"]
				executions := message.([]interface{})
				for _, execution := range executions {
					e := execution.(map[string]interface{})
					log.Println(e["side"], e["price"], e["size"], e["exec_date"])
				}
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			<-done
			return
		}
	}
}
