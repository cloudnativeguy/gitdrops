package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {
	url := "https://api.digitalocean.com/v2/droplets"

	// Create a Bearer string by appending string access token
	var bearer = "Bearer " + os.Getenv("DIGITALOCEAN_TOKEN")

	// Create a new request using http
	req, err := http.NewRequest("GET", url, nil)

	// add authorization header to the req
	req.Header.Add("Authorization", bearer)

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error while reading the response bytes:", err)
	}
	log.Println(string([]byte(body)))
}
