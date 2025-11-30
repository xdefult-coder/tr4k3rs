package main

// client.go
// - fetches ipinfo.io/json, extracts lat/lon and posts to server /report
// - uses Authorization: Bearer <token>
// - customizable via env vars: SERVER_URL, DEVICE_PHONE, DEVICE_TOKEN, INTERVAL

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type GeoIP struct {
	Loc string `json:"loc"`
	IP  string `json:"ip"`
}

func fetchGeoIP() (float64, float64, string, error) {
	resp, err := http.Get("https://ipinfo.io/json")
	if err != nil {
		return 0, 0, "", err
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	var g GeoIP
	if err := json.Unmarshal(b, &g); err != nil {
		return 0, 0, "", err
	}
	var lat, lon float64
	fmt.Sscanf(g.Loc, "%f,%f", &lat, &lon)
	return lat, lon, g.IP, nil
}

func main() {
	server := os.Getenv("SERVER_URL")
	if server == "" {
		server = "http://127.0.0.1:5000"
	}
	phone := os.Getenv("DEVICE_PHONE")
	if phone == "" {
		phone = "kali-device"
	}
	token := os.Getenv("DEVICE_TOKEN")
	if token == "" {
		token = "mytoken123"
	}
	interval := 10
	if v := os.Getenv("INTERVAL"); v != "" {
		if iv, err := strconv.Atoi(v); err == nil { interval = iv }
	}

	for {
		lat, lon, ip, err := fetchGeoIP()
		if err != nil {
			log.Println("geoip error:", err)
			time.Sleep(time.Duration(interval) * time.Second)
			continue
		}
		payload := map[string]interface{}{
			"phone": phone,
			"lat":   lat,
			"lon":   lon,
			"ip":    ip,
		}
		b, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", server+"/report", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println("post error:", err)
		} else {
			body, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			log.Println("posted:", string(body))
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
