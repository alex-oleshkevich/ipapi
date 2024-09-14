package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/MadAppGang/httplog"
	"github.com/oschwald/geoip2-golang"
)

func getIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}

type geoSubdivisionType struct {
	Name string `json:"name"`
	Code string `json:"iso"`
}

type geoIPType struct {
	Continent        string               `json:"continent"`
	ContinentCode    string               `json:"continent_code"`
	Country          string               `json:"country"`
	CountryCode      string               `json:"country_code"`
	Subdivisions     []geoSubdivisionType `json:"subdivisions"`
	City             string               `json:"city"`
	LocationAccuracy uint16               `json:"location_accuracy"`
	Latitude         float64              `json:"latitude"`
	Longitude        float64              `json:"longitude"`
	TimeZone         string               `json:"time_zone"`
}

func getGeoIP(db *geoip2.Reader, ip string) (geoIPType, error) {
	ipAddr := net.ParseIP(ip)
	record, err := db.City(ipAddr)
	if err != nil {
		return geoIPType{}, err
	}

	subdivisions := make([]geoSubdivisionType, len(record.Subdivisions))
	for i, subdivision := range record.Subdivisions {
		subdivisions[i] = geoSubdivisionType{
			Name: subdivision.Names["en"],
			Code: subdivision.IsoCode,
		}
	}

	return geoIPType{
		Country:          record.Country.Names["en"],
		CountryCode:      record.Country.IsoCode,
		City:             record.City.Names["en"],
		Subdivisions:     subdivisions,
		Continent:        record.Continent.Names["en"],
		ContinentCode:    record.Continent.Code,
		LocationAccuracy: record.Location.AccuracyRadius,
		Latitude:         record.Location.Latitude,
		Longitude:        record.Location.Longitude,
		TimeZone:         record.Location.TimeZone,
	}, nil
}

type ipResponseType struct {
	IP string `json:"ip"`
}

type fullResponseType struct {
	ipResponseType
	geoIPType
}

type errorResponseType struct {
	Error string `json:"error"`
}

func main() {
	filePath := os.Getenv("GEOIP_DB_PATH")
	if filePath == "" {
		filePath = "data/GeoLite2-City.mmdb"
	}
	listenHost := os.Getenv("LISTEN_HOST")
	if listenHost == "" {
		listenHost = "0.0.0.0"
	}
	listenPort := os.Getenv("LISTEN_PORT")
	if listenPort == "" {
		listenPort = "8080"
	}

	db, err := geoip2.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ip := r.URL.Query().Get("ip")
		if ip == "" {
			ip = getIP(r)
		}

		geoIP, err := getGeoIP(db, ip)
		if err != nil {
			response := errorResponseType{Error: err.Error()}
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		response := fullResponseType{
			ipResponseType: ipResponseType{IP: ip},
			geoIPType:      geoIP,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	mux.HandleFunc("/ip", func(w http.ResponseWriter, r *http.Request) {
		ip := getIP(r)
		response := ipResponseType{IP: ip}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	srv := &http.Server{
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      httplog.Logger(mux),
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(listenHost, listenPort))
	if err != nil {
		log.Fatal(err)
	}

	err = http.Serve(listener, srv.Handler)
	if err != nil {
		log.Fatal(err)
	}
}
