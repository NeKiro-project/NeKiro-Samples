package runtimeb

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/NeKiro-project/NeKiro/contracts"
	"github.com/NeKiro-project/NeKiro/sdks/agent-sdk/routerauth"
)

const (
	AgentIDEnvironment       = "RUNTIME_B_AGENT_ID"
	RouterEnvironment        = "RUNTIME_B_ROUTER_URL"
	RouterTokenEnvironment   = "RUNTIME_B_ROUTER_TOKEN"
	TargetAgentEnvironment   = "RUNTIME_B_TARGET_AGENT_ID"
	CapabilityEnvironment    = "RUNTIME_B_TARGET_CAPABILITY"
	ResponseLimitEnvironment = "RUNTIME_B_RESPONSE_LIMIT_BYTES"
	EventLimitEnvironment    = "RUNTIME_B_EVENT_LIMIT_BYTES"
)

var runtimeBIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// Config contains the explicit settings needed for Runtime B's managed
// nested-call fixture. It does not contain a direct target endpoint.
type Config struct {
	AgentID       string
	RouterURL     string
	RouterToken   string
	TargetAgentID string
	Capability    string
	ResponseLimit int64
	EventLimit    int64
	RouterAuth    routerauth.Config
}

// LoadConfig reads and validates every required Runtime B caller setting.
func LoadConfig(lookup func(string) (string, bool)) (Config, error) {
	agentID, err := requiredIdentifier(lookup, AgentIDEnvironment)
	if err != nil {
		return Config{}, err
	}
	routerURL, err := requiredValue(lookup, RouterEnvironment)
	if err != nil {
		return Config{}, err
	}
	if err := validateRouterURL(routerURL); err != nil {
		return Config{}, err
	}
	routerToken, err := requiredValue(lookup, RouterTokenEnvironment)
	if err != nil {
		return Config{}, err
	}
	targetAgentID, err := requiredIdentifier(lookup, TargetAgentEnvironment)
	if err != nil {
		return Config{}, err
	}
	capability, err := requiredIdentifier(lookup, CapabilityEnvironment)
	if err != nil {
		return Config{}, err
	}
	responseLimit, err := requiredLimit(lookup, ResponseLimitEnvironment)
	if err != nil {
		return Config{}, err
	}
	eventLimit, err := requiredLimit(lookup, EventLimitEnvironment)
	if err != nil {
		return Config{}, err
	}
	routerAuth, err := routerauth.LoadConfig(lookup)
	if err != nil {
		return Config{}, err
	}
	config := Config{
		AgentID:       agentID,
		RouterURL:     routerURL,
		RouterToken:   routerToken,
		TargetAgentID: targetAgentID,
		Capability:    capability,
		ResponseLimit: responseLimit,
		EventLimit:    eventLimit,
		RouterAuth:    routerAuth,
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (config Config) Validate() error {
	if err := validateIdentifierValue(AgentIDEnvironment, config.AgentID); err != nil {
		return err
	}
	if err := validateRequiredValue(RouterEnvironment, config.RouterURL); err != nil {
		return err
	}
	if err := validateRouterURL(config.RouterURL); err != nil {
		return err
	}
	if err := validateRequiredValue(RouterTokenEnvironment, config.RouterToken); err != nil {
		return err
	}
	if err := validateIdentifierValue(TargetAgentEnvironment, config.TargetAgentID); err != nil {
		return err
	}
	if err := validateIdentifierValue(CapabilityEnvironment, config.Capability); err != nil {
		return err
	}
	if config.ResponseLimit < contracts.RuntimeByteLimitMinimum || config.ResponseLimit > contracts.RuntimeByteLimitMaximum {
		return fmt.Errorf("%s must be an integer from %d through %d", ResponseLimitEnvironment, contracts.RuntimeByteLimitMinimum, contracts.RuntimeByteLimitMaximum)
	}
	if config.EventLimit < contracts.RuntimeByteLimitMinimum || config.EventLimit > contracts.RuntimeByteLimitMaximum {
		return fmt.Errorf("%s must be an integer from %d through %d", EventLimitEnvironment, contracts.RuntimeByteLimitMinimum, contracts.RuntimeByteLimitMaximum)
	}
	if err := config.RouterAuth.Validate(); err != nil {
		return err
	}
	return nil
}

func requiredValue(lookup func(string) (string, bool), name string) (string, error) {
	value, exists := lookup(name)
	if !exists {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, validateRequiredValue(name, value)
}

func validateRequiredValue(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s must be non-empty", name)
	}
	if strings.TrimSpace(value) != value {
		return fmt.Errorf("%s must not contain surrounding whitespace", name)
	}
	return nil
}

func requiredIdentifier(lookup func(string) (string, bool), name string) (string, error) {
	value, err := requiredValue(lookup, name)
	if err != nil {
		return "", err
	}
	return value, validateIdentifierValue(name, value)
}

func validateIdentifierValue(name, value string) error {
	if err := validateRequiredValue(name, value); err != nil {
		return err
	}
	if !runtimeBIdentifierPattern.MatchString(value) {
		return fmt.Errorf("%s must be a safe identifier", name)
	}
	return nil
}

func requiredLimit(lookup func(string) (string, bool), name string) (int64, error) {
	value, err := requiredValue(lookup, name)
	if err != nil {
		return 0, err
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return 0, fmt.Errorf("%s must be an unsigned base-10 integer", name)
		}
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < contracts.RuntimeByteLimitMinimum || parsed > contracts.RuntimeByteLimitMaximum {
		return 0, fmt.Errorf("%s must be an integer from %d through %d", name, contracts.RuntimeByteLimitMinimum, contracts.RuntimeByteLimitMaximum)
	}
	return parsed, nil
}

func validateRouterURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || strings.HasSuffix(parsed.Host, ":") || parsed.User != nil || parsed.Path != "" || parsed.RawPath != "" || parsed.RawQuery != "" || parsed.ForceQuery || strings.Contains(value, "#") || parsed.Fragment != "" || parsed.RawFragment != "" {
		return fmt.Errorf("%s must be an http or https origin URL without credentials, path, query, or fragment", RouterEnvironment)
	}
	if parsed.Port() != "" {
		port, err := strconv.Atoi(parsed.Port())
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("%s port must be an integer from 1 through 65535", RouterEnvironment)
		}
	}
	if parsed.Hostname() == "" {
		return fmt.Errorf("%s must declare a host", RouterEnvironment)
	}
	return nil
}
