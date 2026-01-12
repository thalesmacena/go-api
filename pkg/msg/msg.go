package msg

import (
	"encoding/json"
	"fmt"
	"strconv"
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
			argStr = primitiveToString(arg)
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



// isPrimitive checks if the provided value is of a primitive type (bool, int, uint, float, or string).
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

// primitiveToString converts a primitive value to string using strconv for better performance
func primitiveToString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", value)
	}
}
