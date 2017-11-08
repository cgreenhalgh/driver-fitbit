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
const DS_PROFILE = "profile"
const DS_DEVICES = "devices"

// Driver-specific redirect after Oauth back into App-style view
const AUTH_REDIRECT_URL = "/#!/driver-fitbit/ui"

// OauthServiceConfig
var oauth = OauthServiceConfig{
	AuthUri: "https://www.fitbit.com/oauth2/authorize?response_type=token&scope=profile%20activity%20settings&prompt=consent&state=oauth_callback&expires_in=31536000&redirect_uri="+
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
	profile,err := SyncProfile(d, accessToken)
	// need profile! for timezone
	if profile == nil {
		return false, err
	}
	if firsterr == nil { firsterr = err }
	devices,err := SyncDevices(d, accessToken, profile)
	allok = allok && (devices != nil)
	if firsterr == nil { firsterr = err }	
	ok,err := SyncActivities(d, accessToken, profile, devices)
	allok = allok && ok
	if firsterr == nil { firsterr = err }	
	// TODO more
	return ok,firsterr
}

type FitbitProfileResponse struct{
	User FitbitProfile `json:"user"`
}

// Sync profile to KV
func SyncProfile(d *Driver, accessToken string) (*FitbitProfile, error) {
	log.Printf("Fitbit Sync Profile")
	data,err := GetData("https://api.fitbit.com/1/user/-/profile.json", "Bearer "+accessToken)
	if err != nil {
		log.Printf("Error getting profile: %s", err.Error())
		return nil, err
	}
	var resp = FitbitProfileResponse{}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		log.Printf("Error unmarshalling profile response: %s - %s", err.Error(), string(data))
		return nil,err
	}
	log.Printf("Got profile response for %s (%s), timezone %s (+%d)", resp.User.DisplayName, resp.User.FullName, resp.User.Timezone, resp.User.OffsetFromUTCMillis)
	// update state
	d.UpdateUser(resp.User.FullName, "-")
	// write to store
	ds := d.FindDatasource(DS_PROFILE)
	value,err := json.Marshal(resp.User)
	if err != nil {
		log.Printf("Error marshalling profile value: %s", err.Error())
		return nil, err
	}
	if ds.KvApi == nil {
		log.Printf("ERROR profile KvApi uninitialised!")
		return &resp.User, errors.New("Internal error (profile kvapi uninitialised)")
	}
	ds.KvApi.Write(string(value))
	log.Printf("Synced Profile")
	return &resp.User, nil
}

// Sync devices to KV
func SyncDevices(d *Driver, accessToken string, profile *FitbitProfile) ([]FitbitDevice, error) {
	log.Printf("Fitbit Sync Devices")
	data,err := GetData("https://api.fitbit.com/1/user/-/devices.json", "Bearer "+accessToken)
	if err != nil {
		log.Printf("Error getting devices: %s", err.Error())
		return nil, err
	}
	var resp = []FitbitDevice{}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		log.Printf("Error unmarshalling devices response: %s - %s", err.Error(), string(data))
		return nil,err
	}
	log.Printf("Got devices response with %d devices", len(resp))
	// write to store
	ds := d.FindDatasource(DS_DEVICES)
	// could write raw data straight back
	value,err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error marshalling devices value: %s", err.Error())
		return nil, err
	}
	if ds.KvApi == nil {
		log.Printf("ERROR devices KvApi uninitialised!")
		return resp, errors.New("Internal error (devices kvapi uninitialised)")
	}
	ds.KvApi.Write(string(value))
	log.Printf("Synced devices: %s", string(value))
	return resp, nil
}

// Fitbit Activities / day summary
type FitbitActivitiesResponse struct{
	Summary FitbitDaySummary `json:"summary"`
	// also has activities and goals
}

// Sync DaySummary to TS
func SyncActivities(d *Driver, accessToken string, profile *FitbitProfile, devices []FitbitDevice) (bool, error) {
	// have to do 1 day at a time, and what about current day changing?!
	// TODO limit by device sync information
	// How far back?
	// a test day
	return SyncOneDay("2017-11-05", d, accessToken, profile)
}

func SyncOneDay(date string, d *Driver, accessToken string, profile *FitbitProfile) (bool, error) {	
	log.Printf("Fitbit Sync activity for %s", date)
	data,err := GetData("https://api.fitbit.com/1/user/-/activities/date/"+date+".json", "Bearer "+accessToken)
	if err != nil {
		log.Printf("Error getting activity for %s: %s", date, err.Error())
		return false, err
	}
	var resp = FitbitActivitiesResponse{}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		log.Printf("Error unmarshalling activities response: %s - %s", err.Error(), string(data))
		return false,err
	}
	log.Printf("Got activities response for %s, steps %d", date, resp.Summary.Steps)
	resp.Summary.Date = date
	// TODO: from Profile
	resp.Summary.Timezone = profile.Timezone
	resp.Summary.OffsetFromUTCMillis = profile.OffsetFromUTCMillis
	// write to store
	ds := d.FindDatasource(DS_ACTIVITY_DAY_SUMMARIES)
	// TODO timestamp
	dse := FitbitDaySummaryDSE{
		Data: resp.Summary,
		Timestamp: 0,
	}
	value,err := json.Marshal(dse)
	if err != nil {
		log.Printf("Error marshalling day summary value: %s", err.Error())
		return false, err
	}
	if ds.TsApi == nil {
		log.Printf("ERROR day summary TsApi uninitialised!")
		return false, errors.New("Internal error (day summary tsapi uninitialised)")
	}
	// TODO actually store some information, checking if it is new first ?!
//	ds.TsApi.Write(string(value))
	log.Printf("Synced activity for %s: %s", date, string(value))
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
	DatasourceInfo{
		Metadata: databox.StoreMetadata{
			Description:    "Fitbit devices",
			ContentType:    "application/json",
			Vendor:         "Fitbit",
			DataSourceType: "Fitbit-Devices",
			DataSourceID:   DS_DEVICES,
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
