package main

import (
	"encoding/json"
	"errors"
	//"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	//"strings"
	"strconv"
	"sync"
	"time"
	
	"github.com/gorilla/mux"
	//databox "github.com/me-box/lib-go-databox"
	databox "github.com/cgreenhalgh/lib-go-databox"
)

var dataStoreHref = os.Getenv("DATABOX_STORE_ENDPOINT")

// Note: must match manifest!
const STORE_TYPE = "store-json"

const DS_ACTIVITIES = "activities"
const DS_STATE = "state"

const AUTH_REDIRECT_URL = "/#!/databox-driver-strava/ui"

// TODO any likely errors?!
var activitiesTs,_ = databox.MakeStoreTimeSeries_0_2_0(dataStoreHref, DS_ACTIVITIES, STORE_TYPE)
var stateKv,_ = databox.MakeStoreKeyValue_0_2_0(dataStoreHref, DS_STATE, STORE_TYPE)

// startup state
var started = false
var startupLock = &sync.Mutex{}
var startupCond = sync.NewCond(startupLock)

func isStarted() bool {
	startupLock.Lock()
	defer startupLock.Unlock()
	return started
}

func waitUntilStarted() {
	startupLock.Lock()
	defer startupLock.Unlock()
	for !started {
		startupCond.Wait()
	}
}

func signalStarted() {
	startupLock.Lock()
	defer startupLock.Unlock()
	started = true
	startupCond.Broadcast()
}

func getStatusEndpoint(w http.ResponseWriter, req *http.Request) {
	startupLock.Lock()
	defer startupLock.Unlock()
	if started {
		w.Write([]byte("active\n"))
	} else {
		w.Write([]byte("starting\n"))
	}
}

type OauthUris struct {
  AuthUri string
  TokenUri string
}

var oauthUris = OauthUris{ AuthUri: "https://www.strava.com/oauth/authorize?response_type=code&scope=view_private&approval_prompt=force&state=oauth_callback&", TokenUri: "https://www.strava.com/oauth/token"}

// just for defaults now
type OauthConfig struct {
  ClientID string `json:"client_id"`
  ClientSecret string `json:"client_secret"`
}

var oauthDefaults OauthConfig

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

type Settings struct {
  Status DriverStatus
  Error string // if there is an error
  ClientID string
  AuthUri string
  Authorized bool
  AthleteID string `json:"athlete_id"`
  Firstname string `json:"firstname"`
  Lastname string `json:"lastname"`
  LastSync time.Time `json:"last_sync"` // e.g. "2013-08-24T00:04:12Z"
  LastSyncStatus SyncStatus
  LatestActivity StravaActivity
}

type State struct {
  ClientID string `json:"client_id"`
  ClientSecret string `json:"client_secret"`
  AccessToken string `json:"access_token"`
  AthleteID string `json:"athlete_id"`
  Firstname string `json:"firstname"`
  Lastname string `json:"lastname"`
  LastSync time.Time `json:"last_sync"` // e.g. "2013-08-24T00:04:12Z"
}

func (state *State) Load() {
	data,err := stateKv.Read()
	if err != nil {
		log.Print("Unable to read state\n")
		return
	}
	err = json.Unmarshal([]byte(data), &state)
	if err != nil {
		log.Printf("Unable to unmarshall etc/state.json: %s\n", string(data))
		return
	}
	log.Printf("Read state: %s\n", string(data))
}
func (state *State) Save() {
	data,err := json.Marshal(state)
	if err != nil {
		log.Printf("Unable to marshall state\n");
		return
	}
	err = stateKv.Write(string(data));
	if err != nil {
		log.Print("Error writing state\n")
		return
	}
	log.Print("Saved state\n")
}

var settingsLock = &sync.Mutex{}
var settings = Settings{Status: DRIVER_STARTING, AuthUri: oauthUris.AuthUri, Authorized:false, ClientID:""}
var state = State{}

type Athlete struct {
	ID int64 `json:"id"`
	Firstname string `json:"firstname"`
	Lastname string `json:"lastname"`
}
type OauthTokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType string `json:"token_type"`
	Athlete Athlete `json:"athlete"`
}

func handleOauthCode(code string) {
	resp,err := http.PostForm(oauthUris.TokenUri, 
		url.Values{"client_id":{state.ClientID}, "client_secret":{state.ClientSecret}, "code":{code}})
	if err != nil {
		log.Printf("Error getting token for code %s: %s\n", code, err.Error())
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print("Error reading body of oauth token response\n")
		return
	} 
	log.Printf("oauth token resp %s\n", string(body))
	//state.AccessToken = code[0];
	//state.Save()
	var tokenResp = OauthTokenResp{}
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		log.Printf("Error unmarshalling oauth response: %s\n", string(body))
		return
	}
	//if tokenResp.AccessToken != nil {
	log.Printf("Got access token %s\n", tokenResp.AccessToken)
	settingsLock.Lock()
	state.AccessToken = tokenResp.AccessToken
	//if tokenResp.Athlete != nil {
	state.AthleteID = strconv.FormatInt(tokenResp.Athlete.ID, 10)
	state.Firstname = tokenResp.Athlete.Firstname
	state.Lastname = tokenResp.Athlete.Lastname
	//}
	state.Save()
	settings.Authorized = true
	settings.Status = DRIVER_OK
	settings.Firstname = state.Firstname
	settings.Lastname = state.Lastname
	settings.AthleteID = state.AthleteID
	//}
	settingsLock.Unlock()
}

func displayUI(w http.ResponseWriter, req *http.Request) {
	waitUntilStarted()

	// auth callback?
	params := req.URL.Query()
	codes := params["code"]
	if codes != nil && len(codes)>0 {
		log.Printf("Got oauth response code = %s\n", codes[0])
		code := codes[0]
		handleOauthCode(code)
	}

	var templates *template.Template
	templates, err := template.ParseFiles("tmpl/settings.tmpl")
	if err != nil {
		log.Printf("Error parsing template: %s", err.Error())
		w.Write([]byte("error\n"))
		return
	}
	s1 := templates.Lookup("settings.tmpl")
	settingsLock.Lock()
	err = s1.Execute(w,settings)
	settingsLock.Unlock()
	if err != nil {
		log.Printf("Error filling template: %s", err.Error())
		w.Write([]byte("error\n"))
		return
	}
}

func handleAuthCallback(w http.ResponseWriter, req *http.Request) {
	waitUntilStarted()

	// auth callback?
	params := req.URL.Query()
	codes := params["code"]
	if codes != nil && len(codes)>0 {
		log.Printf("Got oauth response code = %s\n", codes[0])
		code := codes[0]
		handleOauthCode(code)
	}

	errormsg := params["error"]
	if errormsg != nil && len(errormsg)>0 {
		log.Printf("Got oauth error response: %s", errormsg[0])
	}
	log.Printf("auth redirect  -> %s\n", AUTH_REDIRECT_URL)
	// proxy defeats redirect?
	//http.Redirect(w, req, string(url[0:ix]), 302)
	w.Write([]byte("<html><head><meta http-equiv=\"refresh\" content=\"0; URL="+AUTH_REDIRECT_URL+"\" /></head></html>"))
}

func handleConfigure(w http.ResponseWriter, req *http.Request) {
	waitUntilStarted()

	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)
	if err!=nil {
		log.Printf("Error reading configure body: %s", err.Error())
		w.WriteHeader(400)
		return
	}
	var newConfig = OauthConfig{}
	err = json.Unmarshal(body, &newConfig)
	if err != nil {
		log.Printf("Error parsing configure body (%s): %s", body, err.Error())
		w.WriteHeader(400)
		return
	}
	log.Printf("configure oauth client_id="+newConfig.ClientID)
	settingsLock.Lock()
	defer settingsLock.Unlock()
	state.ClientID = newConfig.ClientID
	state.ClientSecret = newConfig.ClientSecret
	settings.ClientID = state.ClientID
	settings.Status = DRIVER_UNAUTHORIZED
	settings.Authorized = false
	// discard old access token (presumable from old app)
	state.AccessToken = ""
	
	state.Save()
	
	w.Write([]byte("true"))
}

type SyncWorker struct {
	Requests chan chan bool
}
var syncWorker = SyncWorker{Requests:make(chan chan bool)}// needed?

// based on actual strava API value
type StravaActivity struct {
	ID int64 `json:"id"`
	Name string `json:"name"`
	Distance float64 `json:"distance"`
	MovingTime float64 `json:"moving_time"`
	ElapsedTime float64 `json:"elapsed_time"`
	Type string `json:"type"` // "ride"
	StartDate time.Time `json:"start_date"` // e.g. "2013-08-24T00:04:12Z"
	Timezone string `json:"timezone"`
}

type DataStoreEntry struct {
	//data 
	Timestamp float64 `json:"timestamp"`
}

type StravaActivityDSE struct {
	Timestamp float64 `json:"timestamp"`
	//*DataStoreEntry
	Data StravaActivity `json:"data"`
}

func syncInternal(accessToken string) (bool, error) {
	// see http://strava.github.io/api/v3/activities/#get-activities
	url := "https://www.strava.com/api/v3/athlete/activities"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating strava request: %s", err.Error())
		return false, err
	}
	client := &http.Client{}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error doing strava get activities: %s", err.Error())
		return false, err
	}
	if res.StatusCode != 200 {
		log.Printf("Error doing strava get activities: status code %d", res.StatusCode)
		return false, errors.New("Status code not 200")
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading strava get activities response: %s", err.Error())
		return false, err
	}
	//log.Printf("Got activities: %s", body)
	var activities []StravaActivity
	err = json.Unmarshal(body, &activities)
	if err != nil {
		log.Printf("Error parsing strave get activities response: %s", err.Error())
		return false, err
	}
	log.Printf("Got %d activities", len(activities))
	
	storeHref, _ := databox.GetStoreURLFromDsHref(dataStoreHref)
	dsHref := storeHref + "/" + DS_ACTIVITIES
	
activityLoop:
	for _,activity := range activities {
		log.Printf("- %d: %s %s at %s (%d)", activity.ID, activity.Type, activity.Name, activity.StartDate, activity.StartDate.Unix())
		// timestamps are Java-style UNIX ms (Number = double). Range query is inclusive
		startTime := float64(activity.StartDate.Unix()*1000)
		res,err := activitiesTs.ReadRange(activity.StartDate, activity.StartDate)
		if err != nil {
			log.Printf("Error checking store entry at %f: %s", startTime, err.Error())
			return false,err
		}
		log.Printf("check %s JSON range %f gave %s", dsHref, startTime, res);
		// timestamp
		got := []StravaActivityDSE{}
		err = json.Unmarshal([]byte(res), &got)
		if err != nil {
			log.Printf("Error unmarshalling existing values at %f: %s (%s)", startTime, err.Error(), res)
			continue
		}
		for _,gotActivity := range got {
			if gotActivity.Data.ID == activity.ID {
				log.Printf("Already got activity %d", activity.ID)
				continue activityLoop
			}
		}
		newValue,err := json.Marshal(activity)
		if err != nil {
			log.Printf("Error marshalling new data item: %s", err.Error())
			continue
		}
		log.Printf("write %s", string(newValue))
		err = activitiesTs.WriteRawValueAt(string(newValue), activity.StartDate)
		if err != nil {
			log.Printf("Error writing new data item to store: %s (%s at %s)", err.Error, string(newValue), activity.StartDate)
			continue;
		}
		//log.Printf("written ok")
		// latest?! NOTE: already holding settingsLock
		settings.LatestActivity = activity
		
		//log.Printf("updated latest")
	}
	log.Printf("sync complete")
	
	return true,nil
}

func (sw *SyncWorker) SyncWorkerServer() {
	for {
		//log.Print("Sync waiting")
		req := <- sw.Requests
		settingsLock.Lock()
		accessToken := state.AccessToken
		settingsLock.Unlock()
		if accessToken == "" {
			log.Print("Sync(internal) with no access token")
			if req != nil {
				req <- false
			}
		} else {
			log.Print("Sync (internal)...")
			settingsLock.Lock()
			settings.LastSyncStatus = SYNC_ACTIVE
			res, _ := syncInternal(accessToken)
			if res {
				settings.LastSyncStatus = SYNC_SUCCESS
				now := time.Now()
				settings.LastSync = now
				state.LastSync = now
				state.Save()
			} else {
				settings.LastSyncStatus = SYNC_FAILURE
			}
			settingsLock.Unlock()
			// signal done
			if req != nil {
				req <- res
			}			
		}
	}
}

func handleSync(w http.ResponseWriter, req *http.Request) {
	if !isStarted() {
		log.Print("ignore handleSync when not started")
		w.Write([]byte("false\n"))
		return
	}
	log.Print("request sync...")
	//rep := make(chan bool)
	syncWorker.Requests <- nil //rep
	w.Write([]byte("true\n"))
	//<- rep
	//log.Print("handleSync complete");
}

type data struct {
	Data string `json:"data"`
}

func loadSettings() {
	settingsLock.Lock()
	defer settingsLock.Unlock()
	// read config
	data,err := ioutil.ReadFile("etc/oauth.json")
	if err != nil {
		log.Print("Unable to read etc/oauth.json\n")
		settings.Status = DRIVER_FATAL
		settings.Error = "The driver was not build correctly (unable to read etc/oauth.json)"
		return
	}
	err = json.Unmarshal(data, &oauthDefaults)
	if err != nil {
		log.Printf("Unable to unmarshall etc/oauth.json: %s\n", string(data))
		settings.Status = DRIVER_FATAL
		settings.Error = "The driver was not build correctly (unable to unmarshall etc/oauth.json)"
		return
	}
	log.Printf("oauth default config client ID %s\n", oauthDefaults.ClientID)
	
	state.Load()
	// defaults
	if len(state.ClientID)==0 {
		state.ClientID = oauthDefaults.ClientID
		state.ClientSecret = oauthDefaults.ClientSecret
	}
	log.Printf("oauth config from state %s\n", state.ClientID)
	settings.ClientID = state.ClientID
	if len(state.AccessToken)>0 {
		settings.Authorized = true
		settings.Status = DRIVER_OK
	} else {
		settings.Status = DRIVER_UNAUTHORIZED
	}
	settings.Firstname = state.Firstname
	settings.Lastname = state.Lastname
	settings.AthleteID = state.AthleteID
	settings.LastSync = state.LastSync
}

func server(c chan bool) {
	//
	// Handle Https requests
	//
	router := mux.NewRouter()

	router.HandleFunc("/status", getStatusEndpoint).Methods("GET")
	router.HandleFunc("/ui/auth_callback", handleAuthCallback).Methods("GET")
	router.HandleFunc("/ui", displayUI).Methods("GET")
	router.HandleFunc("/ui/api/sync", handleSync).Methods("POST")
	router.HandleFunc("/ui/api/configure", handleConfigure).Methods("POST")

	static := http.StripPrefix("/ui/static", http.FileServer(http.Dir("./www/")))
	router.PathPrefix("/ui/static").Handler(static)

	http.ListenAndServeTLS(":8080", databox.GetHttpsCredentials(), databox.GetHttpsCredentials(), router)
	log.Print("HTTP server exited?!")
	c <- true
}

func logFatalError(message string, err error) {
	if  err != nil {
		log.Printf("%s: %s", message, err.Error())	
	} else {
		log.Print(message)
	}
	settingsLock.Lock()
	settings.Status = DRIVER_FATAL
	settings.Error = message
	settingsLock.Unlock()
}

func getLatestActivity() {
	// latest value?
	res,err := activitiesTs.ReadLatest()
	if err != nil {
		log.Printf("Error checking latest store entry: %s", err.Error())
	} else if res=="" {
		log.Print("Warning: get latest -> no value")
	} else {
		//log.Printf("check %s JSON latest gave %s", dsHref, res);
		got := StravaActivityDSE{}
		err = json.Unmarshal([]byte(res), &got)
		if err != nil {
			log.Printf("Error unmarshalling latest value(s): %s (%s)", err.Error(), res)
		} else {
			log.Printf("Initialise latest to %s at %s (%s)", got.Data.Name, got.Data.StartDate, res)
			settingsLock.Lock()
			settings.LatestActivity = got.Data
			settingsLock.Unlock()
		}
	}
}

func main() {

	serverdone := make(chan bool)
	go server(serverdone)
	go syncWorker.SyncWorkerServer()
	
	//Wait for my store to become active
	databox.WaitForStoreStatus(dataStoreHref)

	loadSettings()	

	// register source
	metadata := databox.StoreMetadata{
		Description:    "Strava activities timeseries",
		ContentType:    "application/json",
		Vendor:         "Strava",
		DataSourceType: "Strava-Activity",
		DataSourceID:   DS_ACTIVITIES,
		StoreType:      "store-json",
		IsActuator:     false,
		Unit:           "",
		Location:       "",
	}
	_,err := databox.RegisterDatasource(dataStoreHref, metadata)
	if err != nil {
		logFatalError("Error registering activities datasource", err)
	} else {
		log.Printf("registered datasource %s", DS_ACTIVITIES)
	}

	getLatestActivity()

	signalStarted()
	log.Print("Driver has started")
	
	_ = <-serverdone
}
