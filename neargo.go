package main

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"

	"github.com/aligator/checkpoint"
)

type Geo struct {
	CountryCode string
	PostalCode string
	PlaceName string
	AdminName1 string
	AdminCode1 string
	AdminName2 string
	AdminCode2 string
	AdminName3 string
	AdminCode3 string
	Latitude float64
	Longitude float64
}

type GeoSource interface {
	GetGeoData() ([]Geo, error)
}

type Geonames struct {
	URL string
}

func (g Geonames) readCSV(data io.Reader) ([]Geo, error) {
	r :=  csv.NewReader(data)
	r.Comma = '\t'

	var result []Geo
	for {
		line, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, checkpoint.From(err)
		}

		if len(line) < 11 {
			return nil, checkpoint.From(errors.New("not enough columns"))
		}

		lat, err := strconv.ParseFloat(line[9], 64)
		if err != nil {
			return nil, checkpoint.From(err)
		}
		lon, err := strconv.ParseFloat(line[10], 64)
		if err != nil {
			return nil, checkpoint.From(err)
		}

		result = append(result, Geo{
			CountryCode: line[0],
			PostalCode:  line[1],
			PlaceName:   line[2],
			AdminName1:  line[3],
			AdminCode1:  line[4],
			AdminName2:  line[5],
			AdminCode2:  line[6],
			AdminName3:  line[7],
			AdminCode3:  line[8],
			Latitude:    lat,
			Longitude:   lon,
		})
	}

	return result, nil
}

func (g Geonames) GetGeoData() ([]Geo, error) {
	// Download zip.
	res, err := http.Get(g.URL)
	if err != nil {
		return nil, checkpoint.From(err)
	}

	// Save it as temp file.
	f, err := ioutil.TempFile(os.TempDir(), "geonames.*.zip")
	if err != nil {
		return nil, checkpoint.From(err)
	}
	defer f.Close()
	_, err = f.ReadFrom(res.Body)
	if err != nil {
		return nil, checkpoint.From(err)
	}

	// Unpack zip.
	reader, err := zip.OpenReader(f.Name())
	if err != nil {
		return nil, checkpoint.From(err)
	}
	defer reader.Close()

	var result []Geo
	for _, file := range reader.File {
		if file.Name != "readme.txt" {
			csvReader, err := file.Open()
			if err != nil {
				return nil, checkpoint.From(err)
			}

			geo, err := g.readCSV(csvReader)
			if err != nil {
				return nil, checkpoint.From(err)
			}
			csvReader.Close()

			result = append(result, geo...)
		}
	}

	return result, nil
}

type Neargo struct {
	Source GeoSource
	data []Geo
	index map[string]*Geo
}

func (n *Neargo) Init() error {
	fmt.Println("Init")

	var err error
	n.data, err = n.Source.GetGeoData()
	if err != nil {
		return err
	}

	n.index = make(map[string]*Geo)
	for i := range n.data {
		n.index[n.data[i].PostalCode] = &n.data[i]
	}

	return nil
}

// HSin haversin(θ) function
func HSin(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
}

func Distance(aLat, aLon, bLat, bLon float64) float64 {
	// convert to radians
	// must cast radius as float to multiply later
	var aRLat, aRLon, bRLat, bRLon, r float64
	aRLat = aLat * math.Pi / 180
	aRLon = aLon * math.Pi / 180
	bRLat = bLat * math.Pi / 180
	bRLon = bLon * math.Pi / 180
	r = 6378100 // Earth radius in METERS


	// calculate
	h := HSin(bRLat-aRLat) + math.Cos(aRLat)*math.Cos(bRLat)*HSin(bRLon-aRLon)

	return 2 * r * math.Asin(math.Sqrt(h)) / 1000
}

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

	query := req.URL.Query().Get("q")
	geo, ok := n.index[query]
	if !ok {
		res.WriteHeader(404)
		return
	}

	var result []Geo
	for _, g := range n.data {
		dist := Distance(geo.Latitude, geo.Longitude, g.Latitude, g.Longitude)
		if dist <= float64(max) {
			result = append(result, g)
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

func main() {
	host := flag.String("host", "0.0.0.0:3141", "Host and Port to listen on.")
	geonamesURL := flag.String("geonames-url", "https://download.geonames.org/export/zip/DE.zip", "url to geonames.org zip")
	flag.Parse()

	neargo := Neargo{
		Source: Geonames{URL: *geonamesURL},
	}
	err := neargo.Init()
	if err != nil {
		panic(err)
	}

	fmt.Println("Serving on " + *host)
	err = http.ListenAndServe(*host, neargo)
	panic(err)
}
