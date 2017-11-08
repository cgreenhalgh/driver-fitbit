package main

import(
	"errors"
	"log"
	"io/ioutil"
	"net/http"
)

// General helper routines


// Do HTTP get with given authorization header.
// Return fully read response body.
func GetData(url string, authorization string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating request: %s", err.Error())
		return nil, err
	}
	client := &http.Client{}
	req.Header.Add("Authorization", authorization)
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error doing get %s: %s", url, err.Error())
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading get response (status %d): %s", res.StatusCode, err.Error())
		return nil, err
	}
	if res.StatusCode != 200 {
		log.Printf("Error doing get %s: status code %d, body %s", url, res.StatusCode, string(body))
		return nil, errors.New("Status code not 200")
	}
	return body,nil
}
