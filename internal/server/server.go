package server

import (
	"log"
	"net/http"

	"github.com/olahol/melody"
)

func Run() {
	m := melody.New()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		m.HandleRequest(w, r)
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		m.Broadcast(msg)
	})

	log.Println("Listening on :9876")
	log.Fatal(http.ListenAndServe(":9876", nil))
}

