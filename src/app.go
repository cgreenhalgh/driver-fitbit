package main

import (
	//"encoding/json"
	//"errors"
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

const DS_ACTIVITY_DAY_SUMMARIES = "activity-day-summaries"

const AUTH_REDIRECT_URL = "/#!/driver-fitbit/ui"

// TODO any likely errors?!
var activitiesTs,_ = databox.MakeStoreTimeSeries_0_2_0(dataStoreHref, DS_ACTIVITY_DAY_SUMMARIES, STORE_TYPE)

// OauthServiceConfig
var oauth = OauthServiceConfig{
	AuthUri: "https://www.fitbit.com/oauth2/authorize?response_type=token&scope=profile%20activity&prompt=consent&state=oauth_callback&expires_in=31536000&redirect_uri="+
	url.PathEscape("http://localhost:8989/driver-fitbit/ui/hash_auth_callback"),
	ImplicitGrant: true,
	AuthRedirectUri: AUTH_REDIRECT_URL,
	//TokenUri: "https://api.fitbit.com/oauth2/token",
}

type DaySummaries struct{}

// SyncDatasource
func (ds *DaySummaries) SyncInternal(accessToken string, d *Driver) (bool, error) {
	log.Printf("DaySummaries SyncInternal")
	// TODO
	return true,nil
}

func main() {
	datasource := &DaySummaries{}
	driver := MakeDriver(dataStoreHref, STORE_TYPE, "Fitbit", oauth, datasource)
	serverdone := driver.Start()

	// register source
	metadata := databox.StoreMetadata{
		Description:    "Fitbit activity day summary timeseries",
		ContentType:    "application/json",
		Vendor:         "Fitbit",
		DataSourceType: "Fitbit-Activity-DaySummary",
		DataSourceID:   DS_ACTIVITY_DAY_SUMMARIES,
		StoreType:      "store-json",
		IsActuator:     false,
		Unit:           "",
		Location:       "",
	}
	_,err := databox.RegisterDatasource(dataStoreHref, metadata)
	if err != nil {
		driver.LogFatalError("Error registering activities datasource", err)
	} else {
		log.Printf("registered datasource %s", DS_ACTIVITY_DAY_SUMMARIES)
	}

	//getLatestActivity()

	SignalStarted()
	log.Print("Driver has started")
	
	_ = <-serverdone
}
