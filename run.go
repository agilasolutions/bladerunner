/*
	WARNING
	This piece of code will attempt to break the CXE liveagent go routine
	server and the meralco main chatbot server. Please use this code with CAUTION.
*/

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
)

type Config struct {
	AveResponseTime time.Time
	SuccessTotal    int
	FailTotal       int
	UserID          string `json:"userID"`
	CoreURL         string `json:"coreURL"`
	LoadCapacity    int    `json:"loadCapacity"`
	LoadReps        int    `json:"loadReps"`
	MsgLoad         string `json:"msgLoad"`
	DFUsername      string `json:"dfUsername"`
	DFPassword      string `json:"dfPassword"`
}

// Send payload to meralco main core
func sendPayload(id string, config *Config) {
	basicAuth := config.DFUsername + ":" + config.DFPassword
	basicAuth = base64.StdEncoding.EncodeToString([]byte(basicAuth))
	URL := config.CoreURL
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorizaiton": "Basic " + basicAuth,
	}
	payload := map[string]interface{}{
		"queryResult": map[string]interface{}{
			"intent": map[string]interface{}{
				"displayName": "Default Fallback Intent",
			},
		},
		"originalDetectIntentRequest": map[string]interface{}{
			"source": "facebook",
			"payload": map[string]interface{}{
				"data": map[string]interface{}{
					"sender": map[string]interface{}{
						"id": config.UserID,
					},
					"recipient": map[string]interface{}{
						"id": "2288306637923993",
					},
					"message": map[string]interface{}{
						"text": "yo",
					},
				},
				"source": "facebook",
			},
		},
	}

	resp, err := post(URL, payload, headers)
	if err != nil {
		//log.Println("[ID " + id + "] Result: " + err.Error())
		config.FailTotal++
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		config.SuccessTotal++
		//log.Println("[ID " + id + "] Result: Success " + strconv.Itoa(config.SuccessTotal) + "| Fail " + strconv.Itoa(config.FailTotal))

	} else {
		log.Println(resp.StatusCode)
	}
}

func post(URL string, payload map[string]interface{},
	headers map[string]string) (*http.Response, error) {
	client := &http.Client{}

	jsonPayload, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", URL, bytes.NewBuffer(jsonPayload))
	if headers != nil {
		for key, value := range headers {
			req.Header.Add(key, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// Synchronous load test
func botCoreLoadTest(config *Config, bar *pb.ProgressBar) {
	var waitGroup sync.WaitGroup
	waitGroup.Add(config.LoadCapacity)
	log.Print("Load Capacity: " + strconv.Itoa(config.LoadCapacity))

	for i := 0; i < config.LoadCapacity; i++ {
		go func() {
			defer waitGroup.Done()
			for j := 0; j < config.LoadReps; j++ {
				max := 10
				min := 5
				requestDelay := rand.Intn(max-min) + min
				time.Sleep(time.Duration(requestDelay) * time.Second)
				sendPayload("test", config)
				bar.Increment()
			}
		}()
	}

	waitGroup.Wait()
}

// Load test configuration
func loadConfig() *Config {
	log.Print("Loading configuration..")

	var config Config
	jsonFile, err := os.Open("config.json")
	if err != nil {
		fmt.Println(err)
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		fmt.Println(err)
	}
	json.Unmarshal([]byte(byteValue), &config)
	return &config
}

func main() {
	config := loadConfig()
	bar := pb.StartNew(config.LoadCapacity * config.LoadReps)
	log.Print("Starting load testing with host url: " + config.CoreURL)
	botCoreLoadTest(config, bar)
	bar.Finish()
	log.Println("Result: Success " + strconv.Itoa(config.SuccessTotal) + "| Fail " + strconv.Itoa(config.FailTotal))
}
