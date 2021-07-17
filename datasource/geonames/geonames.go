package geonames

import (
	"archive/zip"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/aligator/checkpoint"
	"github.com/aligator/neargo/server"
)

// Geonames implements neargo.GeoSource by downloading and parsing a zip
// file from geonames.org.
type Geonames struct {
	URL  *string
	Path *string
}

// Flag adds cli flags to configure Geonames using the cli.
func (g *Geonames) Flag() {
	g.URL = flag.String("geonames-url", "https://download.geonames.org/export/zip/DE.zip", "URL to geonames.org zip - pick one from https://download.geonames.org/export/zip/")
	g.Path = flag.String("geonames-file", "", "path where the zip file should be stored - it will only be downloaded if it doesn't exist yet")
}

// readCSV parses the csv files provided by geonames.org.
// The csv separator is a \t.
func (g Geonames) readCSV(data io.Reader) ([]server.Geo, error) {
	r := csv.NewReader(data)
	r.Comma = '\t'

	var result []server.Geo
	for {
		line, err := r.Read()
		// Stop on EOF.
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, checkpoint.From(err)
		}

		// Check if all columns exist.
		if len(line) < 11 {
			return nil, checkpoint.From(errors.New("not enough columns"))
		}

		// Convert lon and lat to float.
		lat, err := strconv.ParseFloat(line[9], 64)
		if err != nil {
			return nil, checkpoint.From(err)
		}
		lon, err := strconv.ParseFloat(line[10], 64)
		if err != nil {
			return nil, checkpoint.From(err)
		}

		// Map the data.
		result = append(result, server.Geo{
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

// GetGeoData downloads a geonames zip file (if it doesn't exist yet),
// unpacks it and then parses the csv files in it. (ignoring "readme.txt" file)
func (g Geonames) GetGeoData() (result []server.Geo, err error) {
	filePath := *g.Path
	if filePath == "" {
		// Use a tmp file if no Path is specified.
		f, err := ioutil.TempFile(os.TempDir(), "geonames.*.zip")
		if err != nil {
			return nil, checkpoint.From(err)
		}

		// Clean it up afterwards.
		defer func() {
			// Clean up tmp file.
			if err = f.Close(); err != nil {
				err = checkpoint.From(err)
				return
			}
			log.Println("Delete tmp file", f.Name())
			err = os.Remove(f.Name())
			if err != nil {
				err = checkpoint.From(err)
				return
			}
		}()

		filePath = f.Name()
	}

	log.Println("Using file", filePath)

	// Download zip file if it doesn't exist yet or if it is a tmp file.
	if _, err := os.Stat(filePath); os.IsNotExist(err) || *g.Path == "" {
		log.Println("Download from", *g.URL)

		if os.IsNotExist(err) {
			// Create it if it doesn't exist.
			_, err := os.Create(*g.Path)
			if err != nil {
				return nil, checkpoint.From(err)
			}
		}

		// Download zip.
		res, err := http.Get(*g.URL)
		if err != nil {
			return nil, checkpoint.From(err)
		}

		if res.StatusCode != 200 {
			return nil, checkpoint.From(fmt.Errorf("could not download zip: %v", res.Status))
		}

		// Save it.
		f, err := os.OpenFile(filePath, os.O_WRONLY, 0)
		if err != nil {
			return nil, checkpoint.From(err)
		}

		_, err = f.ReadFrom(res.Body)
		if err != nil {
			return nil, checkpoint.From(err)
		}

		err = f.Close()
		if err != nil {
			return nil, checkpoint.From(err)
		}
	} else if err != nil {
		return nil, checkpoint.From(err)
	}

	log.Println("Unpack and parse zip file")

	// Unpack zip.
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, checkpoint.From(err)
	}
	defer reader.Close()

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
			err = csvReader.Close()
			if err != nil {
				return nil, checkpoint.From(err)
			}

			result = append(result, geo...)
		}
	}

	log.Println("Parsing finished")
	return result, nil
}
