package msg

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"
	"reflect"
	"strings"
)

var messages map[string]string

// init loads messages from YAML
func init() {
	var value, ok = os.LookupEnv("MESSAGES_FILE_PATH")
	if !ok {
		value = "configs/messages.yml"
	}
	Init(value)
}

func Init(filepath string) {
	viper.SetConfigFile(filepath)
	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Fail to read messages: %v", err)
	}

	if messages == nil {
		messages = make(map[string]string)
	}
	parseMessageMap("", viper.AllSettings(), messages)
}

// parseMessageMap read recursively the yml archive
func parseMessageMap(prefix string, data map[string]interface{}, result map[string]string) {
	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case string:
			result[fullKey] = v
		case map[string]interface{}:
			parseMessageMap(fullKey, v, result)
		default:
			log.Printf("Ignoring key '%s' with unsupported type.", fullKey)
		}
	}
}

// GetMessage returns a msg and format
func GetMessage(key string, args ...interface{}) string {
	msg, exists := messages[key]
	if !exists {
		return fmt.Sprintf("Message not found: %s", key)
	}

	for i, arg := range args {
		placeholder := fmt.Sprintf("{%d}", i)
		var argStr string

		if isPrimitive(arg) {
			argStr = fmt.Sprint(arg)
		} else {
			jsonBytes, err := json.Marshal(arg)
			if err != nil {
				argStr = fmt.Sprintf("%v", arg)
			} else {
				argStr = string(jsonBytes)
			}
		}

		msg = strings.ReplaceAll(msg, placeholder, argStr)
	}

	return msg
}

func isPrimitive(value interface{}) bool {
	switch reflect.TypeOf(value).Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return true
	default:
		return false
	}
}
