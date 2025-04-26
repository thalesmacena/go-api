package resource

import (
	"github.com/spf13/viper"
	"log"
	"os"
	"regexp"
	"time"
)

var properties map[string]any
var envPattern = regexp.MustCompile(`\$\{([^:}]+)(?::([^}]+))?}`)

// init loads application properties from YAML
func init() {
	var value, ok = os.LookupEnv("PROPERTIES_FILE_PATH")
	if !ok {
		value = "configs/application.yml"
	}
	Init(value)
}

func Init(filepath string) {
	viper.SetConfigFile(filepath)
	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Fail to read properties: %v", err)
	}

	if properties == nil {
		properties = make(map[string]any)
	}
	parsePropertiesMap("", viper.AllSettings(), properties)

	err := viper.MergeConfigMap(properties)
	if err != nil {
		log.Fatalf("Error to load application.properties: %v", err)
	}
}

// parsePropertiesMap reads recursively the YAML file
func parsePropertiesMap(prefix string, data map[string]any, result map[string]any) {
	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case string:
			result[fullKey] = resolveEnvVariable(v)
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
			result[fullKey] = v
		case map[string]interface{}:
			parsePropertiesMap(fullKey, v, result)
		default:
			log.Printf("Ignoring key '%s' with unsupported type.", fullKey)
		}
	}
}

// resolveEnvVariable checks if the value is an environment variable pattern and resolves it
func resolveEnvVariable(value string) interface{} {
	matches := envPattern.FindStringSubmatch(value)
	if len(matches) > 0 {
		envName := matches[1]
		defaultValue := ""
		if len(matches) > 2 {
			defaultValue = matches[2]
		}

		if envValue, exists := os.LookupEnv(envName); exists {
			return envValue
		}
		if defaultValue != "" {
			return defaultValue
		}
		return nil
	}
	return nil
}

func Get(key string) any {
	return viper.Get(key)
}

func GetString(key string) string {
	return viper.GetString(key)
}

func GetBool(key string) bool {
	return viper.GetBool(key)
}

func GetDuration(key string) time.Duration {
	return viper.GetDuration(key)
}

func GetTime(key string) time.Time {
	return viper.GetTime(key)
}

func GetInt(key string) int {
	return viper.GetInt(key)
}

func GetInt32(key string) int32 {
	return viper.GetInt32(key)
}

func GetInt64(key string) int64 {
	return viper.GetInt64(key)
}

func GetIntSlice(key string) []int {
	return viper.GetIntSlice(key)
}

func GetFloat64(key string) float64 {
	return viper.GetFloat64(key)
}

func GetSizeInBytes(key string) uint {
	return viper.GetSizeInBytes(key)
}

func GetStringSlice(key string) []string {
	return viper.GetStringSlice(key)
}
