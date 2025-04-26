package main

import (
	"fmt"
	"go-api/pkg/resource"
	"reflect"
)

func main() {
	// Remember to Set default PROPERTIES_FILE_PATH env. Default Location in init is configs/properties.yml
	// This file will reproduce a default properties file
	resource.Init("example/resource/application.yml")
	var valueString = "Value: "
	var valueType = ", Type: "

	// Get Raw Value
	var rawEnvValue = resource.Get("app.string")
	var rawIntValue = resource.Get("app.int")
	var rawSliceValue = resource.Get("app.string-slice")
	fmt.Println("Raw String,", valueString, rawEnvValue, valueType, reflect.TypeOf(rawEnvValue))
	fmt.Println("Raw Int,", valueString, rawIntValue, valueType, reflect.TypeOf(rawIntValue))
	fmt.Println("Raw Slice,", valueString, rawSliceValue, valueType, reflect.TypeOf(rawSliceValue))

	// Get Formatted Value
	var stringValue = resource.GetString("app.string")
	var stringIntValue = resource.GetString("app.int")
	fmt.Println("Correct String parsed,", valueString, stringValue, valueType, reflect.TypeOf(stringValue))
	fmt.Println("Int parsed to String,", valueString, stringIntValue, valueType, reflect.TypeOf(stringIntValue))

	var intStringValue = resource.GetInt("app.string")
	var intValue = resource.GetInt("app.int")
	fmt.Println("Correct Int parsed to int", valueString, intValue, valueType, reflect.TypeOf(intValue))
	fmt.Println("Incorrect String parsed to Int,", valueString, intStringValue, valueType, reflect.TypeOf(intStringValue))

	var duration = resource.GetDuration("app.duration")
	var stringDuration = resource.GetDuration("app.string")
	fmt.Println("Correct Duration Parsed,", valueString, duration, valueType, reflect.TypeOf(duration))
	fmt.Println("Incorrect String parsed to duration,", valueString, stringDuration, valueType, reflect.TypeOf(stringDuration))

	var time = resource.GetTime("app.time")
	var stringTime = resource.GetTime("app.string")
	fmt.Println("Correct Time Parsed,", valueString, time, valueType, reflect.TypeOf(time))
	fmt.Println("Incorrect String parsed to Time,", valueString, stringTime, valueType, reflect.TypeOf(stringTime))

	var stringSlice = resource.GetStringSlice("app.string-slice")
	var intSlice = resource.GetStringSlice("app.int-slice")
	fmt.Println("Example of String Slice,", valueString, stringSlice, valueType, reflect.TypeOf(stringSlice))
	fmt.Println("Example of Int Slice,", valueString, intSlice, valueType, reflect.TypeOf(intSlice))

}
