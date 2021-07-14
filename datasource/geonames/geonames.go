package geonames

import (
	"archive/zip"
	"encoding/csv"
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/aligator/checkpoint"
	"github.com/aligator/neargo/neargo"
)

type Geonames struct {
	url  *string
	path *string
}

func (g *Geonames) Flag() {
	g.url = flag.String("geonames-url", "https://download.geonames.org/export/zip/DE.zip", "url to geonames.org zip - pick one from https://download.geonames.org/export/zip/")
	g.path = flag.String("geonames-file", "", "path where the zip file should be stored - it will only be downloaded if it doesn't exist yet")
}

func (g Geonames) readCSV(data io.Reader) ([]neargo.Geo, error) {
	r := csv.NewReader(data)
	r.Comma = '\t'

	var result []neargo.Geo
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

		result = append(result, neargo.Geo{
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

func (g Geonames) GetGeoData() (result []neargo.Geo, err error) {
	filePath := *g.path
	if filePath == "" {
		// Use a tmp file if no path is specified.
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
	if _, err := os.Stat(filePath); os.IsNotExist(err) || *g.path == "" {
		log.Println("Download from", *g.url)

		if os.IsNotExist(err) {
			// Create it if it doesn't exist.
			os.Create(*g.path)
		}

		// Download zip.
		res, err := http.Get(*g.url)
		if err != nil {
			return nil, checkpoint.From(err)
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

		f.Close()
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
			csvReader.Close()

			result = append(result, geo...)
		}
	}

	log.Println("Parsing finished")
	return result, nil
}
