package main

import(
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

// Stuff specifically to do with oauth interactions

type OauthServiceConfig struct{
	AuthUri string
	ImplicitGrant bool
	AuthRedirectUri string
	TokenUri string
}

type OauthTokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType string `json:"token_type"`
}

func (d *Driver) handleOauthCode(code string) {
	resp,err := http.PostForm(d.oauth.TokenUri, 
		url.Values{"client_id":{d.state.ClientID}, "client_secret":{d.state.ClientSecret}, "code":{code}})
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

	var tokenResp = OauthTokenResp{}
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		log.Printf("Error unmarshalling oauth response: %s\n", string(body))
		return
	}
	//if tokenResp.AccessToken != nil {
	d.handleOauthAccessToken(tokenResp.AccessToken);
	// TODO extra information in response?
}

func (d *Driver) handleOauthAccessToken(token string) {
	log.Printf("Got access token %s\n", token)
	d.settingsLock.Lock()
	d.state.AccessToken = token
	d.SaveState()
	d.settings.Authorized = true
	d.settings.Status = DRIVER_OK
	d.settingsLock.Unlock()
	// TODO extra initialisation?
}

