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

// Get retrieves a value from the properties map by key.
// It returns the value as an interface{} type.
// If the key is not found, it returns nil.
// Example: Get("server.port") might return 8080.
func Get(key string) any {
	return viper.Get(key)
}

// GetString retrieves a value from the properties map by key and returns it as a string.
// If the key is not found, it returns an empty string.
// Example: GetString("server.port") might return "8080".
func GetString(key string) string {
	return viper.GetString(key)
}

// GetBool retrieves a value from the properties map by key and returns it as a boolean.
// If the key is not found, it returns false.
// Example: GetBool("server.enabled") might return true.
func GetBool(key string) bool {
	return viper.GetBool(key)
}

// GetDuration retrieves a value from the properties map by key and returns it as a time.Duration.
// If the key is not found, it returns 0.
// Example: GetDuration("server.timeout") might return 10 * time.Second.
func GetDuration(key string) time.Duration {
	return viper.GetDuration(key)
}

// GetTime retrieves a value from the properties map by key and returns it as a time.Time.
// If the key is not found, it returns a zero time.Time.
// Example: GetTime("server.start_time") might return the current time.
func GetTime(key string) time.Time {
	return viper.GetTime(key)
}

// GetInt retrieves a value from the properties map by key and returns it as an integer.
// If the key is not found, it returns 0.
// Example: GetInt("server.port") might return 8080.
func GetInt(key string) int {
	return viper.GetInt(key)
}

// GetInt32 retrieves a value from the properties map by key and returns it as an int32.
// If the key is not found, it returns 0.
// Example: GetInt32("server.port") might return 8080.
func GetInt32(key string) int32 {
	return viper.GetInt32(key)
}

// GetInt64 retrieves a value from the properties map by key and returns it as an int64.
// If the key is not found, it returns 0.
// Example: GetInt64("server.port") might return 8080.
func GetInt64(key string) int64 {
	return viper.GetInt64(key)
}

// GetIntSlice retrieves a value from the properties map by key and returns it as an integer slice.
// If the key is not found, it returns an empty slice.
// Example: GetIntSlice("server.ports") might return []int{8080, 8081}.
func GetIntSlice(key string) []int {
	return viper.GetIntSlice(key)
}

// GetFloat64 retrieves a value from the properties map by key and returns it as a float64.
// If the key is not found, it returns 0.
// Example: GetFloat64("server.port") might return 8080.
func GetFloat64(key string) float64 {
	return viper.GetFloat64(key)
}

// GetSizeInBytes retrieves a value from the properties map by key and returns it as a uint.
// If the key is not found, it returns 0.
// Example: GetSizeInBytes("server.port") might return 8080.
func GetSizeInBytes(key string) uint {
	return viper.GetSizeInBytes(key)
}

// GetStringSlice retrieves a value from the properties map by key and returns it as a string slice.
// If the key is not found, it returns an empty slice.
// Example: GetStringSlice("server.ports") might return []string{"8080", "8081"}.
func GetStringSlice(key string) []string {
	return viper.GetStringSlice(key)
}
