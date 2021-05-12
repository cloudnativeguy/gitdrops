package main

import (
	"context"
	"github.com/digitalocean/godo"
	"log"
	"os"
)

func main() {
	ctx := context.TODO()
	client := godo.NewFromToken(os.Getenv("DIGITALOCEAN_TOKEN"))
	droplets, err := DropletList(ctx, client)
	if err != nil {
		log.Println("Error while listing droplets", err)
	}
	log.Println(droplets)

}

func DropletList(ctx context.Context, client *godo.Client) ([]godo.Droplet, error) {
	// create a list to hold our droplets
	list := []godo.Droplet{}

	// create options. initially, these will be blank
	opt := &godo.ListOptions{}
	for {
		droplets, resp, err := client.Droplets.List(ctx, opt)
		if err != nil {
			return nil, err
		}

		// append the current page's droplets to our list
		list = append(list, droplets...)

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		// set the page we want for the next request
		opt.Page = page + 1
	}

	return list, nil
}
