package main

import (
	"flag"
	"wechat/hub"
	"net/http"
	"log"
)

var addr = flag.String("tcp", ":9503", "http server address")

func main() {
	flag.Parse()

	chatHub := hub.NewChatHub()
	go chatHub.Run()

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeChat(chatHub, w, r)
	})
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("http ListenAndServe error:", err)
		return
	}
}

func serveHome(w http.ResponseWriter, r *http.Request)  {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", 404)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	http.ServeFile(w, r, "static/home.html")
}

