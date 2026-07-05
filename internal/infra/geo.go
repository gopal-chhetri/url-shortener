package infra

import (
	"fmt"
	"net"

	"github.com/oschwald/geoip2-golang"
)

type GeoService struct {
	db *geoip2.Reader
}

func NewGeoService(dbPath string) (*GeoService, error) {
	db, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open geoip database: %w", err)
	}
	return &GeoService{db: db}, nil
}

func (g *GeoService) Close() error {
	if g.db != nil {
		return g.db.Close()
	}
	return nil
}

type GeoInfo struct {
	Country string
	City    string
}

func (g *GeoService) LookupIP(ipStr string) (*GeoInfo, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	record, err := g.db.City(ip)
	if err != nil {
		return &GeoInfo{Country: "Unknown", City: "Unknown"}, nil
	}

	country := "Unknown"
	if record.Country.Names["en"] != "" {
		country = record.Country.Names["en"]
	}

	city := "Unknown"
	if record.City.Names["en"] != "" {
		city = record.City.Names["en"]
	}

	return &GeoInfo{
		Country: country,
		City:    city,
	}, nil
}
