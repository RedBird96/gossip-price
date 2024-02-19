package global

import (
	"os"
	"strconv"
	"strings"
)

// String returns a string from the environment variable with the given key.
// If the variable is not set, the default value is returned.
// Empty values are allowed and valid.
func EnvString(key, def string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return v
}

// Bool returns a string from the environment variable with the given key.
// If the variable is not set, the default value is returned.
// Empty values are allowed and valid.
func EnvBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return strings.ToLower(v) == "true"
}

// Int returns a string from the environment variable with the given key.
// If the variable is not set, the default value is returned.
// Empty values are allowed and valid.
func EnvInt(key string, def int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	iV, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return iV
}
