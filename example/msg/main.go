package main

import (
	"encoding/json"
	"fmt"
	"go-api/pkg/msg"
)

type anyStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	// Remember to Set default MESSAGES_FILE_PATH env. Default Location in init is configs/messages.yml
	// This file will reproduce a default messages file
	msg.Init("example/msg/messages.yml")
	var messageWithOneField string = "app.field.one"

	// Message without fields
	fmt.Println(msg.GetMessage("app.message"))

	// Message with one field
	fmt.Println(msg.GetMessage(messageWithOneField, "value1"))

	fmt.Println(msg.GetMessage("app.field.two", "value1", 20.0))

	// Load another messages file
	msg.Init("example/msg/example.yml")

	// Old and new messages loaded
	fmt.Println(msg.GetMessage("app.message"))
	fmt.Println(msg.GetMessage("app.new"))

	// Not found message
	fmt.Println(msg.GetMessage("app.not"))

	// Struct field
	example := anyStruct{
		Name: "John Doe",
		Age:  30,
	}
	var logJSON, _ = json.Marshal(example)
	fmt.Println(msg.GetMessage(messageWithOneField, string(logJSON)))
	fmt.Println(msg.GetMessage(messageWithOneField, example))

}
