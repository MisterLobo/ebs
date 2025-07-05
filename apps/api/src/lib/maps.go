package lib

import (
	"ebs/src/config"

	"googlemaps.github.io/maps"
)

var mapsClient *maps.Client

func GetMapsClient() (*maps.Client, error) {
	if mapsClient != nil {
		return mapsClient, nil
	}
	cli, err := maps.NewClient(maps.WithAPIKey(config.GAPI_API_KEY))
	if err != nil {
		return nil, err
	}
	mapsClient = cli
	return cli, nil
}
