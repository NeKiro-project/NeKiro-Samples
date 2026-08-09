package nacosregistration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const responseLimitBytes = 4096

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Registration struct {
	config     Config
	executor   HTTPDoer
	mu         sync.RWMutex
	registered bool
	ready      bool
}

func New(config Config, executor HTTPDoer) (*Registration, error) {
	if config.Mode != ModeNacos || executor == nil || config.Validate() != nil {
		return nil, errors.New("Nacos registration dependencies are invalid")
	}
	return &Registration{config: config, executor: executor}, nil
}

func (value *Registration) Register(ctx context.Context) error {
	metadata, _ := json.Marshal(map[string]string{"nekiro.instanceId": value.config.InstanceID})
	parameters := value.baseValues()
	parameters.Set("ephemeral", "true")
	parameters.Set("enabled", "true")
	parameters.Set("healthy", "true")
	parameters.Set("weight", "1.0")
	parameters.Set("metadata", string(metadata))
	if err := value.request(ctx, http.MethodPost, parameters); err != nil {
		value.setState(false, false)
		return fmt.Errorf("register runtime with Nacos: %w", err)
	}
	value.setState(true, true)
	return nil
}

func (value *Registration) Run(ctx context.Context) error {
	value.mu.RLock()
	registered := value.registered
	value.mu.RUnlock()
	if !registered {
		return errors.New("Nacos registration has not started")
	}
	ticker := time.NewTicker(value.config.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			value.setReady(false)
			return nil
		case <-ticker.C:
			if err := value.Heartbeat(ctx); err != nil {
				value.setReady(false)
				return fmt.Errorf("heartbeat Nacos registration: %w", err)
			}
		}
	}
}

func (value *Registration) Heartbeat(ctx context.Context) error {
	beat, _ := json.Marshal(struct {
		IP        string            `json:"ip"`
		Port      int               `json:"port"`
		Service   string            `json:"serviceName"`
		Cluster   string            `json:"cluster"`
		Metadata  map[string]string `json:"metadata"`
		Weight    float64           `json:"weight"`
		Scheduled bool              `json:"scheduled"`
	}{
		IP: value.config.AdvertisedIP, Port: value.config.AdvertisedPort,
		Service: value.config.GroupName + "@@" + value.config.ServiceName, Cluster: value.config.ClusterName,
		Metadata: map[string]string{"nekiro.instanceId": value.config.InstanceID},
		Weight:   1.0, Scheduled: false,
	})
	parameters := value.baseValues()
	parameters.Set("beat", string(beat))
	return value.request(ctx, http.MethodPut, parameters)
}

func (value *Registration) Deregister(ctx context.Context) error {
	value.mu.RLock()
	registered := value.registered
	value.mu.RUnlock()
	value.setReady(false)
	if !registered {
		return nil
	}
	err := value.request(ctx, http.MethodDelete, value.baseValues())
	value.setState(false, false)
	if err != nil {
		return fmt.Errorf("deregister runtime from Nacos: %w", err)
	}
	return nil
}

func (value *Registration) Ready() bool {
	value.mu.RLock()
	defer value.mu.RUnlock()
	return value.ready
}

func (value *Registration) baseValues() url.Values {
	parameters := url.Values{
		"serviceName": {value.config.ServiceName},
		"groupName":   {value.config.GroupName},
		"clusterName": {value.config.ClusterName},
		"namespaceId": {value.config.NamespaceID},
		"ip":          {value.config.AdvertisedIP},
		"port":        {strconv.Itoa(value.config.AdvertisedPort)},
		"ephemeral":   {"true"},
	}
	if value.config.AuthMode == AuthAccessToken {
		parameters.Set("accessToken", value.config.AccessToken)
	}
	return parameters
}

func (value *Registration) request(ctx context.Context, method string, parameters url.Values) error {
	if ctx == nil {
		return errors.New("Nacos request context is required")
	}
	requestContext, cancel := context.WithTimeout(ctx, value.config.RequestTimeout)
	defer cancel()
	endpoint := strings.TrimSuffix(value.config.APIOrigin, "/") + "/v1/ns/instance"
	if method == http.MethodPut {
		endpoint += "/beat"
	}
	var body io.Reader
	if method == http.MethodPut {
		body = strings.NewReader(parameters.Encode())
	} else {
		endpoint += "?" + parameters.Encode()
	}
	request, err := http.NewRequestWithContext(requestContext, method, endpoint, body)
	if err != nil {
		return errors.New("build Nacos request")
	}
	if method == http.MethodPut {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	response, err := value.executor.Do(request)
	if err != nil {
		if requestContext.Err() != nil {
			return errors.New("Nacos request canceled or timed out")
		}
		return errors.New("Nacos request unavailable")
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, responseLimitBytes+1))
	if err != nil {
		return errors.New("read Nacos response")
	}
	if len(responseBody) > responseLimitBytes {
		return errors.New("Nacos response exceeds limit")
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Nacos returned status %d", response.StatusCode)
	}
	if method == http.MethodPut {
		var outcome struct {
			Code int `json:"code"`
		}
		if json.Unmarshal(responseBody, &outcome) != nil || outcome.Code != 10200 {
			return errors.New("Nacos heartbeat outcome is invalid")
		}
	} else if strings.TrimSpace(string(responseBody)) != "ok" {
		return errors.New("Nacos registration outcome is invalid")
	}
	return nil
}

func (value *Registration) setReady(ready bool) {
	value.mu.Lock()
	value.ready = ready
	value.mu.Unlock()
}

func (value *Registration) setState(registered, ready bool) {
	value.mu.Lock()
	value.registered = registered
	value.ready = ready
	value.mu.Unlock()
}
