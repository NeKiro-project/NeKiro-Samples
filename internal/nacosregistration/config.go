package nacosregistration

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	ModeDisabled    = "disabled"
	ModeNacos       = "nacos"
	AuthNone        = "none"
	AuthAccessToken = "access_token"
	minimumMillis   = 100
	maximumMillis   = 60000
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

type Config struct {
	Mode              string
	APIOrigin         string
	NamespaceID       string
	GroupName         string
	ServiceName       string
	ClusterName       string
	AdvertisedIP      string
	AdvertisedPort    int
	HeartbeatInterval time.Duration
	RequestTimeout    time.Duration
	AuthMode          string
	AccessToken       string
	InstanceID        string
}

func Load(lookup func(string) (string, bool), prefix, instanceID string) (Config, error) {
	if lookup == nil || !identifierPattern.MatchString(instanceID) || prefix != "RUNTIME_A" && prefix != "RUNTIME_B" {
		return Config{}, errorsFor(prefix, "registration dependencies are invalid")
	}
	name := func(suffix string) string { return prefix + "_" + suffix }
	mode, err := required(lookup, name("REGISTRATION_MODE"))
	if err != nil {
		return Config{}, err
	}
	config := Config{Mode: mode, InstanceID: instanceID}
	nacosSuffixes := []string{"NACOS_API_ORIGIN", "NACOS_NAMESPACE_ID", "NACOS_GROUP_NAME", "NACOS_SERVICE_NAME", "NACOS_CLUSTER_NAME", "NACOS_ADVERTISED_IP", "NACOS_ADVERTISED_PORT", "NACOS_HEARTBEAT_INTERVAL_MS", "NACOS_REQUEST_TIMEOUT_MS", "NACOS_AUTH_MODE", "NACOS_ACCESS_TOKEN"}
	if mode == ModeDisabled {
		for _, suffix := range nacosSuffixes {
			if _, exists := lookup(name(suffix)); exists {
				return Config{}, fmt.Errorf("%s must be absent when registration is disabled", name(suffix))
			}
		}
		return config, nil
	}
	if mode != ModeNacos {
		return Config{}, fmt.Errorf("%s is unsupported", name("REGISTRATION_MODE"))
	}
	if config.APIOrigin, err = required(lookup, name("NACOS_API_ORIGIN")); err != nil {
		return Config{}, err
	}
	if err := validateOrigin(config.APIOrigin, name("NACOS_API_ORIGIN")); err != nil {
		return Config{}, err
	}
	for environment, destination := range map[string]*string{
		name("NACOS_NAMESPACE_ID"): &config.NamespaceID,
		name("NACOS_GROUP_NAME"):   &config.GroupName,
		name("NACOS_SERVICE_NAME"): &config.ServiceName,
		name("NACOS_CLUSTER_NAME"): &config.ClusterName,
	} {
		*destination, err = requiredIdentifier(lookup, environment)
		if err != nil {
			return Config{}, err
		}
	}
	config.AdvertisedIP, err = required(lookup, name("NACOS_ADVERTISED_IP"))
	parsedIP := net.ParseIP(config.AdvertisedIP)
	if err != nil || parsedIP == nil || parsedIP.String() != config.AdvertisedIP {
		return Config{}, fmt.Errorf("%s must be a canonical IP address", name("NACOS_ADVERTISED_IP"))
	}
	port, err := requiredUnsigned(lookup, name("NACOS_ADVERTISED_PORT"), 1, 65535)
	if err != nil {
		return Config{}, err
	}
	config.AdvertisedPort = int(port)
	heartbeat, err := requiredUnsigned(lookup, name("NACOS_HEARTBEAT_INTERVAL_MS"), minimumMillis, maximumMillis)
	if err != nil {
		return Config{}, err
	}
	config.HeartbeatInterval = time.Duration(heartbeat) * time.Millisecond
	timeout, err := requiredUnsigned(lookup, name("NACOS_REQUEST_TIMEOUT_MS"), minimumMillis, maximumMillis)
	if err != nil {
		return Config{}, err
	}
	config.RequestTimeout = time.Duration(timeout) * time.Millisecond
	config.AuthMode, err = required(lookup, name("NACOS_AUTH_MODE"))
	if err != nil {
		return Config{}, err
	}
	switch config.AuthMode {
	case AuthNone:
		if _, exists := lookup(name("NACOS_ACCESS_TOKEN")); exists {
			return Config{}, fmt.Errorf("%s must be absent when Nacos authentication is none", name("NACOS_ACCESS_TOKEN"))
		}
	case AuthAccessToken:
		config.AccessToken, err = required(lookup, name("NACOS_ACCESS_TOKEN"))
		if err != nil {
			return Config{}, err
		}
	default:
		return Config{}, fmt.Errorf("%s is unsupported", name("NACOS_AUTH_MODE"))
	}
	return config, config.Validate()
}

func (config Config) Validate() error {
	if !identifierPattern.MatchString(config.InstanceID) {
		return errorsFor("runtime", "instance ID is invalid")
	}
	if config.Mode == ModeDisabled {
		return nil
	}
	if config.Mode != ModeNacos || validateOrigin(config.APIOrigin, "Nacos API origin") != nil || !identifierPattern.MatchString(config.NamespaceID) || !identifierPattern.MatchString(config.GroupName) || !identifierPattern.MatchString(config.ServiceName) || !identifierPattern.MatchString(config.ClusterName) {
		return errorsFor("runtime", "Nacos registration tuple is invalid")
	}
	parsedIP := net.ParseIP(config.AdvertisedIP)
	if parsedIP == nil || parsedIP.String() != config.AdvertisedIP || config.AdvertisedPort < 1 || config.AdvertisedPort > 65535 || config.HeartbeatInterval < minimumMillis*time.Millisecond || config.HeartbeatInterval > maximumMillis*time.Millisecond || config.RequestTimeout < minimumMillis*time.Millisecond || config.RequestTimeout > maximumMillis*time.Millisecond {
		return errorsFor("runtime", "Nacos registration endpoint or timing is invalid")
	}
	if config.AuthMode != AuthNone && config.AuthMode != AuthAccessToken || config.AuthMode == AuthNone && config.AccessToken != "" || config.AuthMode == AuthAccessToken && strings.TrimSpace(config.AccessToken) == "" {
		return errorsFor("runtime", "Nacos authentication configuration is invalid")
	}
	return nil
}

func required(lookup func(string) (string, bool), name string) (string, error) {
	value, exists := lookup(name)
	if !exists || value == "" || strings.TrimSpace(value) != value {
		return "", fmt.Errorf("%s is required and must contain no surrounding whitespace", name)
	}
	return value, nil
}

func requiredIdentifier(lookup func(string) (string, bool), name string) (string, error) {
	value, err := required(lookup, name)
	if err != nil || !identifierPattern.MatchString(value) {
		return "", fmt.Errorf("%s must be a safe identifier", name)
	}
	return value, nil
}

func requiredUnsigned(lookup func(string) (string, bool), name string, minimum, maximum int64) (int64, error) {
	value, err := required(lookup, name)
	if err != nil {
		return 0, err
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return 0, fmt.Errorf("%s must be an unsigned base-10 integer", name)
		}
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < minimum || parsed > maximum {
		return 0, fmt.Errorf("%s must be an integer from %d through %d", name, minimum, maximum)
	}
	return parsed, nil
}

func validateOrigin(value, name string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Path != "/nacos" || parsed.RawPath != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.RawFragment != "" {
		return fmt.Errorf("%s must be an HTTP(S) URL with the exact /nacos path", name)
	}
	return nil
}

func errorsFor(owner, message string) error { return fmt.Errorf("%s %s", owner, message) }
