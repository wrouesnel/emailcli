package envutil

import (
	"fmt"
	"os"
	"strings"
)

// EnvironmentVariablesError is raised when an environment variable is improperly formatted.
type EnvironmentVariablesError struct {
	Reason    string
	RawEnvVar string
}

// Error implements error.
func (eev EnvironmentVariablesError) Error() string {
	return fmt.Sprintf("%s: %s", eev.Reason, eev.RawEnvVar)
}

// FromEnvironment consumes the environment and outputs a valid input data field into the
// supplied map.
func FromEnvironment(env []string) (map[string]string, error) {
	results := map[string]string{}

	if env == nil {
		env = os.Environ()
	}

	const expectedArgs = 2

	for _, keyval := range env {
		splitKeyVal := strings.SplitN(keyval, "=", expectedArgs)
		if len(splitKeyVal) != expectedArgs {
			return results, error(EnvironmentVariablesError{
				Reason:    "Could not find an equals value to split on",
				RawEnvVar: keyval,
			})
		}
	}

	return results, nil
}
