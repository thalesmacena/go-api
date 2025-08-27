package main

import (
	"go-api/pkg/log"
	"go.uber.org/zap"
)

type anyStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {

	var booleanValue bool = true
	var stringValue string = "value"
	var intValue int64 = 1
	var floatValue float64 = 1.0
	example := anyStruct{
		Name: "John Doe",
		Age:  30,
	}

	// Remember to set APPLICATION_NAME env
	log.Info("Example info with Zap Logger with normal Logger (hardTyped). You can use log.Infom, log.Debug, log.Error, etc.",
		zap.Bool("bool", booleanValue),
		zap.String("string", stringValue),
		zap.Int64p("int", &intValue),
		zap.Float64("float", floatValue),
		zap.Any("struct", example),
	)

	log.Infow("Example with less Performatic suggaredLogger (non Strong Type). You can use log.Infow logDebugw, logErrorw, etc.",
		"key1", booleanValue,
		"key2", stringValue,
		"key3", intValue,
		"key4", floatValue,
		"key5", example)

	log.Infof("Example with lassPerformatic suggaredLogger message formatter, You can use log.Infof, logDebugf, logErrorf."+
		" Example message: 'Failed to fetch URL: %s'", stringValue)
}
