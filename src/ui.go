package main

import(
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	//"net/url"
	//"time"
	
	"github.com/gorilla/mux"
	databox "github.com/cgreenhalgh/lib-go-databox"
)

// The UI embedded HTTP server

func getStatusEndpoint(w http.ResponseWriter, req *http.Request) {
	startupLock.Lock()
	defer startupLock.Unlock()
	if IsStarted() {
		w.Write([]byte("active\n"))
	} else {
		w.Write([]byte("starting\n"))
	}
}

func (d *Driver) handleOauthToken(token string) {
	log.Printf("Got access token %s\n", token)
	d.settingsLock.Lock()
	d.state.AccessToken = token
	d.SaveState()
	d.settings.Authorized = true
	d.settings.Status = DRIVER_OK
	d.settingsLock.Unlock()
	// TODO name etc.
}

func (d *Driver) displayUI(w http.ResponseWriter, req *http.Request) {
	WaitUntilStarted()

	var templates *template.Template
	templates, err := template.ParseFiles("tmpl/settings.tmpl")
	if err != nil {
		log.Printf("Error parsing template: %s", err.Error())
		w.Write([]byte("error\n"))
		return
	}
	s1 := templates.Lookup("settings.tmpl")
	d.settingsLock.Lock()
	err = s1.Execute(w,d.settings)
	d.settingsLock.Unlock()
	if err != nil {
		log.Printf("Error filling template: %s", err.Error())
		w.Write([]byte("error\n"))
		return
	}
}

// some apis, e.g. fitbit, return parameters after fragment, not in query
func (d *Driver) handleHashAuthCallback(w http.ResponseWriter, req *http.Request) {
	log.Printf("Got oauth hash callback");
	w.Write([]byte("<html><head></head><body><script>var hash=location.hash; location.assign('auth_callback?'+hash.substring(1));</script></body></html>"))
}

func (d *Driver) handleAuthCallback(w http.ResponseWriter, req *http.Request) {
	WaitUntilStarted()

	//log.Printf("Got oauth callback: %s", req.URL)
	// debug
	//time.Sleep(10000 * time.Millisecond)

	// auth callback?
	params := req.URL.Query()
	codes := params["code"]
	if codes != nil && len(codes)>0 {
		log.Printf("Got oauth response code = %s\n", codes[0])
		code := codes[0]
		d.handleOauthCode(code)
	} else if tokens := params["access_token"]; tokens != nil && len(tokens)>0 {
		log.Printf("Got oauth response access token = %s", tokens[0])
		d.handleOauthAccessToken(tokens[0])
	} else if errormsg := params["error"]; errormsg != nil && len(errormsg)>0 {
		log.Printf("Got oauth error response: %s", errormsg[0])
	} else {
		log.Printf("Got oauth callback unhandled: %s", req.URL)
	}
	log.Printf("auth redirect  -> %s\n", AUTH_REDIRECT_URL)
	// proxy defeats redirect?
	//http.Redirect(w, req, string(url[0:ix]), 302)
	w.Write([]byte("<html><head><meta http-equiv=\"refresh\" content=\"0; URL="+d.oauth.AuthRedirectUri+"\" /></head></html>"))
}

func (d *Driver) handleConfigure(w http.ResponseWriter, req *http.Request) {
	WaitUntilStarted()

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
	d.settingsLock.Lock()
	defer d.settingsLock.Unlock()
	d.state.ClientID = newConfig.ClientID
	d.state.ClientSecret = newConfig.ClientSecret
	d.settings.ClientID = d.state.ClientID
	d.settings.Status = DRIVER_UNAUTHORIZED
	d.settings.Authorized = false
	// discard old access token (presumable from old app)
	d.state.AccessToken = ""
	
	d.SaveState()
	
	w.Write([]byte("true"))
}

func (d *Driver) handleSync(w http.ResponseWriter, req *http.Request) {
	if !IsStarted() {
		log.Print("ignore handleSync when not started")
		w.Write([]byte("false\n"))
		return
	}
	log.Print("request sync...")
	//rep := make(chan bool)
	d.syncRequests <- nil //rep
	w.Write([]byte("true\n"))
	//<- rep
	//log.Print("handleSync complete");
}

func (d *Driver) server(c chan bool) {
	//
	// Handle Https requests
	//
	router := mux.NewRouter()

	router.HandleFunc("/status", getStatusEndpoint).Methods("GET")
	router.HandleFunc("/ui/auth_callback", d.handleAuthCallback).Methods("GET")
	router.HandleFunc("/ui/hash_auth_callback", d.handleHashAuthCallback).Methods("GET")
	router.HandleFunc("/ui", d.displayUI).Methods("GET")
	router.HandleFunc("/ui/api/sync", d.handleSync).Methods("POST")
	router.HandleFunc("/ui/api/configure", d.handleConfigure).Methods("POST")

	static := http.StripPrefix("/ui/static", http.FileServer(http.Dir("./www/")))
	router.PathPrefix("/ui/static").Handler(static)

	http.ListenAndServeTLS(":8080", databox.GetHttpsCredentials(), databox.GetHttpsCredentials(), router)
	log.Print("HTTP server exited?!")
	c <- true
}

// outputs bool on channel when server terminates
func (d *Driver) Start() chan bool {
	serverdone := make(chan bool)
	go d.server(serverdone)
	go d.syncWorkerServer()
	
	//Wait for my store to become active
	databox.WaitForStoreStatus(d.dataStoreHref)

	d.LoadSettings()
	
	return serverdone
}
