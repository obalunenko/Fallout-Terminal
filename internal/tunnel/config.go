package tunnel

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBinary  = "ngrok"
	DefaultDomain  = "fallout-terminal.ngrok.app"
	DefaultPort    = 3690
	passwordMinLen = 8
	passwordMaxLen = 128
)

const DefaultStartupTimeout = 20 * time.Second

// Credentials contains the ephemeral Basic Auth identity used only while
// preparing and starting a protected public tunnel.
type Credentials struct {
	Username string
	Password string
}

// Config contains validated, process-local tunnel settings. It is never
// serialized into a session or exposed directly to the frontend.
type Config struct {
	Enabled        bool
	Binary         string
	Domain         string
	Port           int
	LocalURL       string
	StartupTimeout time.Duration
	PolicyParent   string
	Credentials    Credentials
}

// ParseConfig combines application arguments and environment variables while
// retaining the legacy precedence rules. Argument credentials win over the
// environment; username/password pairs win over the combined environment form.
func ParseConfig(args []string, lookupEnv func(string) (string, bool)) (Config, error) {
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	config := Config{
		Binary:         environmentOrDefault(lookupEnv, "NGROK_BIN", DefaultBinary),
		Domain:         environmentOrDefault(lookupEnv, "NGROK_DOMAIN", DefaultDomain),
		Port:           DefaultPort,
		StartupTimeout: DefaultStartupTimeout,
		PolicyParent:   os.TempDir(),
	}
	if enabled, ok := lookupEnv("NGROK_ENABLED"); ok && enabled == "1" {
		config.Enabled = true
	}

	var parsed parsedArguments
	if err := parsed.read(args); err != nil {
		return Config{}, err
	}
	if parsed.enabled {
		config.Enabled = true
	}
	if parsed.binarySet {
		if parsed.binary == "" {
			return Config{}, fmt.Errorf("ngrok binary path must not be empty")
		}
		config.Binary = parsed.binary
	}
	if parsed.domainSet {
		if parsed.domain == "" {
			return Config{}, fmt.Errorf("ngrok domain must not be empty")
		}
		config.Domain = parsed.domain
	}

	timeoutText := ""
	if parsed.timeoutSet {
		timeoutText = parsed.timeout
	} else if value, ok := lookupEnv("NGROK_TIMEOUT"); ok {
		timeoutText = value
	} else if value, ok := lookupEnv("NGROK_TIMEOUT_MS"); ok && value != "" {
		milliseconds, err := strconv.ParseInt(value, 10, 64)
		if err != nil || milliseconds <= 0 {
			return Config{}, fmt.Errorf("NGROK_TIMEOUT_MS must be a positive integer")
		}
		config.StartupTimeout = time.Duration(milliseconds) * time.Millisecond
	}
	if timeoutText != "" {
		timeout, err := time.ParseDuration(timeoutText)
		if err != nil || timeout <= 0 {
			return Config{}, fmt.Errorf("ngrok startup timeout must be a positive duration")
		}
		config.StartupTimeout = timeout
	}

	credentials, provided, err := resolveCredentials(parsed, lookupEnv)
	if err != nil {
		if config.Enabled {
			return config, err
		}
		return config, nil
	}
	if provided {
		config.Credentials = credentials
	}
	if config.Enabled {
		if !provided {
			return config, fmt.Errorf("protected public access requires a username and password")
		}
		if err := validateCredentials(credentials); err != nil {
			return config, err
		}
	}

	return config, nil
}

type parsedArguments struct {
	enabled bool

	basicAuth    string
	basicAuthSet bool
	username     string
	usernameSet  bool
	password     string
	passwordSet  bool
	binary       string
	binarySet    bool
	domain       string
	domainSet    bool
	timeout      string
	timeoutSet   bool
}

func (parsed *parsedArguments) read(args []string) error {
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if argument == "--ngrok" {
			parsed.enabled = true
			continue
		}

		name, value, hasValue := strings.Cut(argument, "=")
		recognized := true
		switch name {
		case "--ngrok-basic-auth":
			parsed.basicAuthSet = true
		case "--ngrok-username":
			parsed.usernameSet = true
		case "--ngrok-password":
			parsed.passwordSet = true
		case "--ngrok-bin":
			parsed.binarySet = true
		case "--ngrok-domain":
			parsed.domainSet = true
		case "--ngrok-timeout":
			parsed.timeoutSet = true
		default:
			recognized = false
		}
		if !recognized {
			continue
		}
		if !hasValue {
			if index+1 >= len(args) || strings.HasPrefix(args[index+1], "--") {
				return fmt.Errorf("%s requires a value", name)
			}
			index++
			value = args[index]
		}

		switch name {
		case "--ngrok-basic-auth":
			parsed.basicAuth = value
		case "--ngrok-username":
			parsed.username = value
		case "--ngrok-password":
			parsed.password = value
		case "--ngrok-bin":
			parsed.binary = value
		case "--ngrok-domain":
			parsed.domain = value
		case "--ngrok-timeout":
			parsed.timeout = value
		}
	}
	return nil
}

func resolveCredentials(parsed parsedArguments, lookupEnv func(string) (string, bool)) (Credentials, bool, error) {
	if parsed.basicAuthSet {
		credentials, err := splitCredential(parsed.basicAuth)
		return credentials, true, err
	}

	username, _ := lookupEnv("NGROK_USERNAME")
	password, _ := lookupEnv("NGROK_PASSWORD")
	usernameProvided := username != ""
	passwordProvided := password != ""
	if parsed.usernameSet {
		username = parsed.username
		usernameProvided = true
	}
	if parsed.passwordSet {
		password = parsed.password
		passwordProvided = true
	}
	if usernameProvided || passwordProvided {
		credentials := Credentials{Username: username, Password: password}
		return credentials, true, validateCredentials(credentials)
	}

	combined, combinedProvided := lookupEnv("NGROK_BASIC_AUTH")
	if !combinedProvided || combined == "" {
		return Credentials{}, false, nil
	}
	credentials, err := splitCredential(combined)
	return credentials, true, err
}

func splitCredential(combined string) (Credentials, error) {
	separator := strings.IndexByte(combined, ':')
	if separator < 0 {
		return Credentials{}, fmt.Errorf("protected public access requires a username and password")
	}
	credentials := Credentials{
		Username: combined[:separator],
		Password: combined[separator+1:],
	}
	if err := validateCredentials(credentials); err != nil {
		return Credentials{}, err
	}
	return credentials, nil
}

func validateCredentials(credentials Credentials) error {
	if credentials.Username == "" {
		return fmt.Errorf("ngrok username must not be empty")
	}
	if strings.ContainsAny(credentials.Username, "\r\n") {
		return fmt.Errorf("ngrok username must not contain a newline")
	}
	if len(credentials.Password) < passwordMinLen || len(credentials.Password) > passwordMaxLen {
		return fmt.Errorf("ngrok password must contain between %d and %d characters", passwordMinLen, passwordMaxLen)
	}
	if strings.ContainsAny(credentials.Password, "\r\n") {
		return fmt.Errorf("ngrok password must not contain a newline")
	}
	return nil
}

func environmentOrDefault(lookupEnv func(string) (string, bool), key, fallback string) string {
	if value, ok := lookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
