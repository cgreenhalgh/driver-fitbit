package main

import(
	"encoding/json"
	"io/ioutil"
	"log"
	"sync"
	"time"
	
	databox "github.com/cgreenhalgh/lib-go-databox"
)

// driver-related state stuff - the main driver object, if you will

// just for defaults now
type OauthConfig struct {
  ClientID string `json:"client_id"`
  ClientSecret string `json:"client_secret"`
}

type DriverStatus int

const (
	DRIVER_STARTING DriverStatus = iota
	DRIVER_FATAL 
	DRIVER_UNAUTHORIZED 
	DRIVER_OK 
)

type SyncStatus int

const (
	SYNC_IDLE = iota
	SYNC_ACTIVE
	SYNC_FAILURE
	SYNC_SUCCESS
)

// Settings are maintained in-memory and back the UI view
type Settings struct {
  ServiceName string
  Status DriverStatus
  Error string // if there is an error
  ClientID string
  AuthUri string
  Authorized bool
  UserID string 
  UserName string 
  LastSync time.Time 
  LastSyncStatus SyncStatus
}

// State is persistent
type State struct {
  ClientID string `json:"client_id"`
  ClientSecret string `json:"client_secret"`
  RefreshToken string `json:"refresh_token"`
  AccessToken string `json:"access_token"`
  UserID string `json:"user_id"`
  UserName string `json:"user_name"`
  LastSync time.Time `json:"last_sync"` // e.g. "2013-08-24T00:04:12Z"
}

const DS_STATE = "state"

func (d *Driver) LoadState() {
	data,err := d.stateKv.Read()
	if err != nil {
		log.Print("Unable to read state\n")
		return
	}
	err = json.Unmarshal([]byte(data), &d.state)
	if err != nil {
		log.Printf("Unable to unmarshall etc/state.json: %s\n", string(data))
		return
	}
	log.Printf("Read state: %s\n", string(data))
}
func (d *Driver) SaveState() {
	data,err := json.Marshal(d.state)
	if err != nil {
		log.Printf("Unable to marshall state\n");
		return
	}
	err = d.stateKv.Write(string(data));
	if err != nil {
		log.Print("Error writing state\n")
		return
	}
	log.Print("Saved state\n")
}

func (d *Driver) LoadSettings() {
	d.settingsLock.Lock()
	defer d.settingsLock.Unlock()
	// read config
	data,err := ioutil.ReadFile("etc/oauth.json")
	if err != nil {
		log.Print("Unable to read etc/oauth.json\n")
		d.settings.Status = DRIVER_FATAL
		d.settings.Error = "The driver was not build correctly (unable to read etc/oauth.json)"
		return
	}
	oauthDefaults := OauthConfig{}
	err = json.Unmarshal(data, &oauthDefaults)
	if err != nil {
		log.Printf("Unable to unmarshall etc/oauth.json: %s\n", string(data))
		d.settings.Status = DRIVER_FATAL
		d.settings.Error = "The driver was not build correctly (unable to unmarshall etc/oauth.json)"
		return
	}
	log.Printf("oauth default config client ID %s\n", oauthDefaults.ClientID)
	
	d.LoadState()
	// defaults
	if len(d.state.ClientID)==0 {
		d.state.ClientID = oauthDefaults.ClientID
		d.state.ClientSecret = oauthDefaults.ClientSecret
	}
	log.Printf("oauth config from state %s\n", d.state.ClientID)
	d.settings.ClientID = d.state.ClientID
	if len(d.state.AccessToken)>0 {
		d.settings.Authorized = true
		d.settings.Status = DRIVER_OK
	} else {
		d.settings.Status = DRIVER_UNAUTHORIZED
	}
	d.settings.UserName = d.state.UserName
	d.settings.UserID = d.state.UserID
	d.settings.LastSync = d.state.LastSync
}

func (d *Driver) LogFatalError(message string, err error) {
	if  err != nil {
		log.Printf("%s: %s", message, err.Error())	
	} else {
		log.Print(message)
	}
	d.settingsLock.Lock()
	d.settings.Status = DRIVER_FATAL
	d.settings.Error = message
	d.settingsLock.Unlock()
}

// Main driver state
type Driver struct{
	stateKv databox.KeyValue_0_2_0
	settingsLock *sync.Mutex
	settings *Settings
	state *State
	// sync requests
	syncRequests chan chan bool
	dataStoreHref string
	datasource SyncDatasource
	oauth OauthServiceConfig
}

// Make a driver object for this specific service
func MakeDriver(dataStoreHref string, storeType string, serviceName string, oauth OauthServiceConfig, datasource SyncDatasource) *Driver {
	var driver = &Driver{dataStoreHref:dataStoreHref,oauth:oauth,datasource:datasource}
	driver.stateKv,_ = databox.MakeStoreKeyValue_0_2_0(dataStoreHref, DS_STATE, storeType)
	driver.settingsLock = &sync.Mutex{}
	driver.settings = &Settings{
		ServiceName:serviceName,
		AuthUri:oauth.AuthUri,
		Status: DRIVER_STARTING,
		Authorized:false,
		ClientID:""}
	driver.state = &State{}
	driver.syncRequests = make(chan chan bool)
	return driver
}
