package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
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

type GeoDistance struct {
	Geo
	Distance float64
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

type ErrorMessage struct {
	Message string
}

func writeErrorMessage(res http.ResponseWriter, message string) {
	enc := json.NewEncoder(res)
	err := enc.Encode(ErrorMessage{
		Message: message,
	})
	if err != nil {
		_, _ = fmt.Fprint(os.Stderr, err.Error())
	}
}

// ServeHTTP responds with the found geo data as json
// based on the passed query parameters.
func (n Neargo) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")

	maxStr := req.URL.Query().Get("max")
	if maxStr == "" {
		maxStr = "100"
	}
	max, err := strconv.Atoi(maxStr)
	if err != nil {
		_, _ = fmt.Fprint(os.Stderr, err.Error())
		res.WriteHeader(400)
		writeErrorMessage(res, "invalid value for 'max'")
		return
	}

	country := req.URL.Query().Get("country")
	zipCode := req.URL.Query().Get("zip")
	geo, ok := n.index[country][zipCode]
	if !ok {
		res.WriteHeader(404)
		writeErrorMessage(res, "combination of 'country' and 'zip' not found")
		return
	}

	var result []GeoDistance
	for _, g := range n.index[country] {
		dist := Distance(geo.Latitude, geo.Longitude, g.Latitude, g.Longitude)
		if dist <= float64(max) {
			result = append(result, GeoDistance{
				Geo:      *g,
				Distance: dist,
			})
		}
	}

	slices.SortStableFunc(result, func(a, b GeoDistance) int {
		if a.Distance > b.Distance {
			return 1
		} else if a.Distance < b.Distance {
			return -1
		}
		return 0
	})

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
