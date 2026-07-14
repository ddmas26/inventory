package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ipinfo/go/v2/ipinfo"
)

type timeHandler struct {
	format string
}

type locHandler struct {
}

func (th timeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tm := time.Now().Format(th.format)
	w.Write([]byte("The time is: " + tm))
}

func (lc locHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	client := ipinfo.NewClient(nil, nil, "4e3a22d87202d4")

	// Passing nil gets info for your own current IP
	info, err := client.GetIPInfo(nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("City: %s, Country: %s\n", info.City, info.CountryName)
	w.Write([]byte("The city is: " + info.City))

}

func main() {
	mux := http.NewServeMux()

	// Initialise the timeHandler in exactly the same way we would any normal
	// struct.
	th := timeHandler{format: time.RFC1123}
	lc := locHandler{}

	// Like the previous example, we use the mux.Handle() function to register
	// this with our ServeMux.
	mux.Handle("/time", th)
	mux.Handle("/loc", lc)

	log.Print("Listening...")
	http.ListenAndServe(":3000", mux)
}
