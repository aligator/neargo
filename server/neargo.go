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
	index map[string]map[string][]*Geo
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

	n.index = make(map[string]map[string][]*Geo)
	for i := range n.data {
		countryIndex, ok := n.index[n.data[i].CountryCode]
		if !ok {
			n.index[n.data[i].CountryCode] = make(map[string][]*Geo)
			countryIndex = n.index[n.data[i].CountryCode]
		}

		countryIndex[n.data[i].PostalCode] = append(countryIndex[n.data[i].PostalCode], &n.data[i])
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
	exactGeos, ok := n.index[country][zipCode]
	if !ok || len(exactGeos) <= 0 {
		res.WriteHeader(404)
		writeErrorMessage(res, "combination of 'country' and 'zip' not found")
		return
	}

	// If max is < 0, the api will only match the equal zip codes.
	var result []GeoDistance
	if max < 0 {
		for _, geos := range n.index[country] {
			for _, geo := range geos {
				if geo.PostalCode == zipCode {
					result = append(result, GeoDistance{
						Geo:      *geo,
						Distance: 0,
					})
				}
			}
		}
	} else {
		deduplicateSet := make(map[*Geo]bool)

		// If a zip code exists multiple times, do the search for each of them.
		for _, exactGeo := range exactGeos {
			for _, geos := range n.index[country] {
				for _, geo := range geos {
					dist := Distance(exactGeo.Latitude, exactGeo.Longitude, geo.Latitude, geo.Longitude)
					if dist <= float64(max) {
						// Skip already added geos
						if _, ok := deduplicateSet[geo]; ok {
							continue
						}

						result = append(result, GeoDistance{
							Geo:      *geo,
							Distance: dist,
						})
						deduplicateSet[geo] = true
					}
				}
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
