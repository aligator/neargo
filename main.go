package main

import (
	"log"
	"net/http"

	"github.com/aligator/neargo/datasource/geonames"
	"github.com/aligator/neargo/server"
	"github.com/rs/cors"
	"github.com/spf13/pflag"
)

func main() {
	gn := geonames.Geonames{}
	gn.PFlag()
	host := pflag.String("host", "0.0.0.0:3141", "Host and Port to listen on.")
	origins := pflag.StringSlice("origins", []string{"*"}, `Comma separated CORS Origins`)
	pflag.Parse()

	neargo := server.Neargo{
		Source: gn,
	}
	err := neargo.Init()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Serving on " + *host)
	err = http.ListenAndServe(*host, cors.New(cors.Options{
		AllowedOrigins: *origins,
		AllowedMethods: []string{http.MethodHead, http.MethodOptions, http.MethodGet},
	}).Handler(neargo))

	log.Fatal(err)
}
