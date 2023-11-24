# neargo
This is a simple postal code proximity search webservice.  
It uses geonames.org as source.

Just start it with `go run .` and call
http://localhost:3141/?country=DE&zip=80331&max=100

The full documentation of the api is in the [OpenAPI documentation](openapi.yml).

## CLI
You can get all options by `neargo --help`

## Docker image

```
docker run --rm -p3141:3141 -v ./data:/data ghcr.io/aligator/neargo:latest --geonames-file geonames.zip
```