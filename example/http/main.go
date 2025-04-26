package main

import (
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

func main() {
	// Client Options
	clientOptions := http.ClientOptions{
		FollowRedirect: true,
		Dismiss404:     false,
		DefaultHeaders: map[string]string{"Authorization": "Bearer token"},
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

	// Creating Client For Post
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
}
