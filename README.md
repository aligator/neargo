# neargo
A simple postal code proximity search webservice.  
It uses geonames.org as source.

Just start it and call
http://localhost:3141/?country=DE&zip=80331&max=100

Param `country` is the country code,  
Param `zip` is the postal code query,
Param `max` is the max distance in km.

The result is json in this form:
```
[
  {
    "CountryCode": string // iso country code, 2 characters
    "PostalCode" : string
    "PlaceName"  : string
    "AdminName1" : string // 1. order subdivision (state) varchar(100)
    "AdminCode1" : string // 1. order subdivision (state) varchar(20)
    "AdminName2" : string // 2. order subdivision (county/province) varchar(100)
    "AdminCode2" : string // 2. order subdivision (county/province) varchar(20)
    "AdminName3" : string // 3. order subdivision (community) varchar(100)
    "AdminCode3" : string // 3. order subdivision (community) varchar(20)
    "Latitude"   : number // estimated latitude (wgs84)
    "Longitude"  : number // estimated longitude (wgs84)
  }
]
```

By default it uses `https://download.geonames.org/export/zip/DE.zip` as datasource and therefore only supports germany.  
However you can use any other zip file from https://download.geonames.org/export/zip by using the following comand line parameter:  

`neargo -geonames-url https://download.geonames.org/export/zip/allCountries.zip` which would use all supported countries.  
`neargo -host 127.0.0.1:7744` can be used to change the host and port.
