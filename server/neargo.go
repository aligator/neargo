package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

// Geo is the model used to store the geo-information.
type Geo struct {
	CountryCode string
	PostalCode  string
	PlaceName   string
	AdminName1  string
	AdminCode1  string
	AdminName2  string
	AdminCode2  string
	AdminName3  string
	AdminCode3  string
	Latitude    float64
	Longitude   float64
}

// GeoSource provides a method to load the Geo data from somewhere.
type GeoSource interface {
	GetGeoData() ([]Geo, error)
}

// Neargo is a simple webservice which provides a proximity
// search by postal code.
type Neargo struct {
	// Source can be set to any implementation of GeoSource which defines where
	// the data comes from.
	Source GeoSource

	// data holds all Geo data
	data []Geo

	// index contains references to the data
	// as index[country][zip] for faster access.
	index map[string]map[string]*Geo
}

// Init has to be called before serving it. It loads and initializes
// the data using the provided Source.
func (n *Neargo) Init() error {
	log.Println("Initialize")

	var err error
	n.data, err = n.Source.GetGeoData()
	if err != nil {
		return err
	}

	n.index = make(map[string]map[string]*Geo)
	for i := range n.data {
		countryIndex, ok := n.index[n.data[i].CountryCode]
		if !ok {
			n.index[n.data[i].CountryCode] = make(map[string]*Geo)
			countryIndex = n.index[n.data[i].CountryCode]
		}
		countryIndex[n.data[i].PostalCode] = &n.data[i]
	}

	log.Println("Initialized")

	return nil
}

// ServeHTTP responds with the found geo data as json
// based on the passed query parameters.
func (n Neargo) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	maxStr := req.URL.Query().Get("max")
	if maxStr == "" {
		maxStr = "100"
	}
	max, err := strconv.Atoi(maxStr)
	if err != nil {
		_, _ = fmt.Fprint(os.Stderr, err.Error())
		res.WriteHeader(400)
		return
	}

	country := req.URL.Query().Get("country")
	zipCode := req.URL.Query().Get("zip")
	geo, ok := n.index[country][zipCode]
	if !ok {
		res.WriteHeader(404)
		return
	}

	var result []Geo
	for _, g := range n.index[country] {
		dist := Distance(geo.Latitude, geo.Longitude, g.Latitude, g.Longitude)
		if dist <= float64(max) {
			result = append(result, *g)
		}
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		_, _ = fmt.Fprint(os.Stderr, err.Error())
		res.WriteHeader(500)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	_, err = res.Write(resultJSON)
	if err != nil {
		panic(err)
	}
}
