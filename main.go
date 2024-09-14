package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/oschwald/geoip2-golang"
)

func getIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}

type geoIPType struct {
	Country          string  `json:"country"`
	CountryCode      string  `json:"country_code"`
	City             string  `json:"city"`
	Continent        string  `json:"continent"`
	ContinentCode    string  `json:"continent_code"`
	LocationAccuracy uint16  `json:"location_accuracy"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	TimeZone         string  `json:"time_zone"`
}

func getGeoIP(db *geoip2.Reader, ip string) (geoIPType, error) {
	ipAddr := net.ParseIP(ip)
	record, err := db.City(ipAddr)
	if err != nil {
		return geoIPType{}, err
	}
	return geoIPType{
		Country:          record.Country.Names["en"],
		CountryCode:      record.Country.IsoCode,
		City:             record.City.Names["en"],
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
		filePath = "GeoLite2-City.mmdb"
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

	err = http.ListenAndServe(net.JoinHostPort(listenHost, listenPort), mux)
	if err != nil {
		log.Fatal(err)
	}
}
