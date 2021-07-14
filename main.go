package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/aligator/neargo/datasource/geonames"
	"github.com/aligator/neargo/neargo"
)

func main() {
	gn := geonames.Geonames{}
	gn.Flag()
	host := flag.String("host", "0.0.0.0:3141", "Host and Port to listen on.")
	flag.Parse()

	neargo := neargo.Neargo{
		Source: gn,
	}
	err := neargo.Init()
	if err != nil {
		panic(err)
	}

	log.Println("Serving on " + *host)
	err = http.ListenAndServe(*host, neargo)
	panic(err)
}
