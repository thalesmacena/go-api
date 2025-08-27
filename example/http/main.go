package main

import (
	"encoding/xml"
	"go-api/pkg/http"
	"go-api/pkg/log"
)

type PokeResponse struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Height int    `json:"height"`
	Weight int    `json:"weight"`
	Order  int    `json:"order"`
}

type APIErrorResponse struct {
	Error APIError `json:"error"`
}

type APIError struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Errors  []APIErrorDetail `json:"errors"`
	Status  string           `json:"status"`
}

type APIErrorDetail struct {
	Message string `json:"message"`
	Domain  string `json:"domain"`
	Reason  string `json:"reason"`
}

type CreatePostRequest struct {
	Title  string `json:"title"`
	Body   string `json:"body"`
	UserID int    `json:"userId"`
}

type CreatePostResponse struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	UserID int    `json:"userId"`
}

// XML structure for httpbin /xml example
// https://httpbin.io/xml returns a <slideshow> root with attributes
// We map only a few attributes for demonstration
// Full structure is not required for successful unmarshaling
// and unknown fields are ignored by encoding/xml

type HttpbinXML struct {
	XMLName xml.Name `xml:"slideshow"`
	Title   string   `xml:"title,attr"`
	Date    string   `xml:"date,attr"`
	Author  string   `xml:"author,attr"`
}

// XML POST request payload for /anything
// The response from /anything will be JSON echoing the raw body in the "data" field

type XMLAnythingRequest struct {
	XMLName xml.Name `xml:"note"`
	To      string   `xml:"to"`
	From    string   `xml:"from"`
	Heading string   `xml:"heading"`
	Body    string   `xml:"body"`
}

type HttpbinAnythingResponse struct {
	Data    string              `json:"data"`
	Headers map[string][]string `json:"headers"`
	URL     string              `json:"url"`
}

func main() {
	// Client Options with JSON as default content type
	clientOptions := http.ClientOptions{
		FollowRedirect:     true,
		Dismiss404:         false,
		DefaultHeaders:     map[string]string{"Authorization": "Bearer token"},
		DefaultContentType: "application/json",
	}

	// Creating a Client
	client := http.NewHttpClient("https://pokeapi.co/api/v2", clientOptions)

	// Success Request
	success, failure, status, err := client.Get("/pokemon/ditto", nil, nil, &PokeResponse{}, nil)

	if err != nil {
		log.Errorw("Request Error", "status", status, "error", err, "body", failure)
	} else {
		log.Infow("Request Success", "status", status, "body", success)
	}

	// Creating other Client
	client = http.NewHttpClient("https://www.googleapis.com/youtube/v3", clientOptions)

	queryParams := map[string]string{
		"q": "test",
	}

	// Error Request
	success, failure, status, err = client.Get("search", queryParams, nil, nil, &APIErrorResponse{})

	if err != nil {
		log.Errorw("Request Error", "status", status, "error", err, "body", failure)
	} else {
		log.Infow("Request Success", "status", status, "body", success)
	}

	// Creating Client For Post with JSON
	client = http.NewHttpClient(
		"https://jsonplaceholder.typicode.com",
		clientOptions,
	)

	reqBody := CreatePostRequest{
		Title:  "foo",
		Body:   "bar",
		UserID: 1,
	}

	// Success Request
	success, failure, status, err = client.Post("/posts", nil, nil, reqBody, &CreatePostResponse{}, nil)

	if err != nil {
		log.Errorw("Request Error", "status", status, "error", err, "body", failure)
	} else {
		log.Infow("Request Success", "status", status, "body", success)
	}

	// Using Request Builder
	requestSuccessBody, requestErrorBody, requestStatus, requestErr := client.Request().
		WithMethod(http.POST).
		WithPath("/posts").
		WithBody(reqBody).
		WithSuccessResp(&CreatePostResponse{}).
		Execute()

	if requestErr != nil {
		log.Errorw("Request Error", "status", requestStatus, "error", requestErr, "body", requestErrorBody)
	} else {
		log.Infow("Request Success", "status", requestStatus, "body", requestSuccessBody)
	}

	// Example: Public XML API using httpbin
	xmlClient := http.NewHttpClient("https://httpbin.io", http.ClientOptions{DefaultContentType: "application/xml"})
	var xmlResp HttpbinXML
	xmlSuccess, xmlFailure, xmlStatus, xmlErr := xmlClient.Get("/xml", nil, nil, &xmlResp, nil)
	if xmlErr != nil {
		log.Errorw("XML Request Error", "status", xmlStatus, "error", xmlErr, "body", xmlFailure)
	} else {
		log.Infow("XML Request Success", "status", xmlStatus, "body", xmlSuccess)
	}

	// Example: XML POST to /xml (httpbin returns static XML)
	xmlPostClient := http.NewHttpClient("https://httpbin.io", http.ClientOptions{DefaultContentType: "application/xml"})
	xmlReq := XMLAnythingRequest{To: "Alice", From: "Bob", Heading: "Reminder", Body: "Meeting at 10"}
	var xmlPostResp HttpbinXML
	xmlPostSuccess, xmlPostFailure, xmlPostStatus, xmlPostErr := xmlPostClient.Post("/xml", nil, nil, xmlReq, &xmlPostResp, nil)
	if xmlPostErr != nil {
		log.Errorw("XML POST Error", "status", xmlPostStatus, "error", xmlPostErr, "body", xmlPostFailure)
	} else {
		log.Infow("XML POST Success", "status", xmlPostStatus, "body", xmlPostSuccess)
	}

	// Example: Public text/plain endpoint (robots.txt) using httpbin
	textClient := http.NewHttpClient("https://httpbin.io", http.ClientOptions{DefaultContentType: "text/plain"})
	var textResponse string
	textSuccess, textFailure, textStatus, textErr := textClient.Get("/robots.txt", nil, nil, &textResponse, nil)
	if textErr != nil {
		log.Errorw("Text Request Error", "status", textStatus, "error", textErr, "body", textFailure)
	} else {
		log.Infow("Text Request Success", "status", textStatus, "body", textSuccess)
	}

	// Example: Public binary endpoint using httpbin (random bytes)
	binaryClient := http.NewHttpClient("https://httpbin.io", http.ClientOptions{DefaultContentType: "application/octet-stream"})
	var binaryResponse []byte
	binarySuccess, binaryFailure, binaryStatus, binaryErr := binaryClient.Get("/bytes/16", nil, nil, &binaryResponse, nil)
	if binaryErr != nil {
		log.Errorw("Binary Request Error", "status", binaryStatus, "error", binaryErr, "body", binaryFailure)
	} else {
		log.Infow("Binary Request Success", "status", binaryStatus, "body", binarySuccess)
	}
}
