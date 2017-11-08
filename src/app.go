package main

import (
	"encoding/json"
	"errors"
	//"fmt"
	//"html/template"
	//"io/ioutil"
	"log"
	//"net/http"
	"net/url"
	"os"
	//"strings"
	//"strconv"
	//"sync"
	//"time"
	
	//"github.com/gorilla/mux"
	//databox "github.com/me-box/lib-go-databox"
	databox "github.com/cgreenhalgh/lib-go-databox"
)

var dataStoreHref = os.Getenv("DATABOX_STORE_ENDPOINT")

// Note: must match manifest!
const STORE_TYPE = "store-json"

// Our data source IDs
const DS_ACTIVITY_DAY_SUMMARIES = "activity-day-summaries"
const DS_PROFILE= "profile"

// Driver-specific redirect after Oauth back into App-style view
const AUTH_REDIRECT_URL = "/#!/driver-fitbit/ui"

// OauthServiceConfig
var oauth = OauthServiceConfig{
	AuthUri: "https://www.fitbit.com/oauth2/authorize?response_type=token&scope=profile%20activity&prompt=consent&state=oauth_callback&expires_in=31536000&redirect_uri="+
	url.PathEscape("http://localhost:8989/driver-fitbit/ui/hash_auth_callback"),
	ImplicitGrant: true,
	AuthRedirectUri: AUTH_REDIRECT_URL,
	//TokenUri: "https://api.fitbit.com/oauth2/token",
}

// Our sync function
func FitbitSyncHandler(d *Driver, accessToken string) (bool, error) {
	//log.Printf("Fitbit Sync")
	allok := true
	var firsterr error 
	ok,err := SyncProfile(d, accessToken)
	allok = allok && ok
	if firsterr == nil { firsterr = err }
	// TODO
	return true,nil
}

// Fitbit profile information
type FitbitProfile struct{
	DisplayName string `json:"displayName"`
	FullName string `json:"fullName"`
	OffsetFromUTCMillis int `json:"offsetFromUTCMillis"`
	Timezone string `json:"timezone"`
}

type FitbitProfileResponse struct{
	User FitbitProfile `json:"user"`
}

// Sync profile to KV
func SyncProfile(d *Driver, accessToken string) (bool, error) {
	log.Printf("Fitbit Sync Profile")
	data,err := GetData("https://api.fitbit.com/1/user/-/profile.json", "Bearer "+accessToken)
	if err != nil {
		log.Printf("Error getting profile: %s", err.Error())
		return false, err
	}
	var resp = FitbitProfileResponse{}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		log.Printf("Error unmarshalling profile response: %s - %s", err.Error(), string(data))
		return false,err
	}
	log.Printf("Got profile response for %s (%s), timezone %s (+%d)", resp.User.DisplayName, resp.User.FullName, resp.User.Timezone, resp.User.OffsetFromUTCMillis)
	// update state
	d.UpdateUser(resp.User.FullName, "-")
	// write to store
	ds := d.FindDatasource(DS_PROFILE)
	value,err := json.Marshal(resp.User)
	if err != nil {
		log.Printf("Error marshalling profile value: %s", err.Error())
		return false, err
	}
	if ds.KvApi == nil {
		log.Printf("ERROR profile KvApi uninitialised!")
		return false, errors.New("Internal error (profile kvapi uninitialised)")
	}
	ds.KvApi.Write(string(value))
	log.Printf("Synced Profile")
	return true, nil
}

// All of our datasources
var datasources = []DatasourceInfo{
	DatasourceInfo{
		Metadata: databox.StoreMetadata{
			Description:    "Fitbit activity day summary timeseries",
			ContentType:    "application/json",
			Vendor:         "Fitbit",
			DataSourceType: "Fitbit-Activity-DaySummary",
			DataSourceID:   DS_ACTIVITY_DAY_SUMMARIES,
			StoreType:      STORE_TYPE,
			IsActuator:     false,
			Unit:           "",
			Location:       "",
		},
		Api: API_TIMESERIES,
		DataStoreHref: dataStoreHref,
	},
	DatasourceInfo{
		Metadata: databox.StoreMetadata{
			Description:    "Fitbit profile",
			ContentType:    "application/json",
			Vendor:         "Fitbit",
			DataSourceType: "Fitbit-Profile",
			DataSourceID:   DS_PROFILE,
			StoreType:      STORE_TYPE,
			IsActuator:     false,
			Unit:           "",
			Location:       "",
		},
		Api: API_KEYVALUE,
		DataStoreHref: dataStoreHref,
	},
}

func main() {
	driver := MakeDriver(dataStoreHref, STORE_TYPE, "Fitbit", oauth, datasources, FitbitSyncHandler)
	serverdone := driver.Start()

	//getLatestActivity()

	SignalStarted()
	log.Print("Driver has started")
	
	_ = <-serverdone
}
