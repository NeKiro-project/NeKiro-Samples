package runtimeb

import "github.com/NeKiro-project/NeKiro-Samples/internal/nacosregistration"

const (
	RegistrationModeEnvironment       = "RUNTIME_B_REGISTRATION_MODE"
	NacosAPIOriginEnvironment         = "RUNTIME_B_NACOS_API_ORIGIN"
	NacosNamespaceEnvironment         = "RUNTIME_B_NACOS_NAMESPACE_ID"
	NacosGroupEnvironment             = "RUNTIME_B_NACOS_GROUP_NAME"
	NacosServiceEnvironment           = "RUNTIME_B_NACOS_SERVICE_NAME"
	NacosClusterEnvironment           = "RUNTIME_B_NACOS_CLUSTER_NAME"
	NacosAdvertisedIPEnvironment      = "RUNTIME_B_NACOS_ADVERTISED_IP"
	NacosAdvertisedPortEnvironment    = "RUNTIME_B_NACOS_ADVERTISED_PORT"
	NacosHeartbeatIntervalEnvironment = "RUNTIME_B_NACOS_HEARTBEAT_INTERVAL_MS"
	NacosRequestTimeoutEnvironment    = "RUNTIME_B_NACOS_REQUEST_TIMEOUT_MS"
	NacosAuthModeEnvironment          = "RUNTIME_B_NACOS_AUTH_MODE"
	NacosAccessTokenEnvironment       = "RUNTIME_B_NACOS_ACCESS_TOKEN"
	RegistrationModeDisabled          = nacosregistration.ModeDisabled
	RegistrationModeNacos             = nacosregistration.ModeNacos
	NacosAuthNone                     = nacosregistration.AuthNone
	NacosAuthAccessToken              = nacosregistration.AuthAccessToken
)

type RegistrationConfig = nacosregistration.Config

func LoadRegistrationConfig(lookup func(string) (string, bool), instanceID string) (RegistrationConfig, error) {
	return nacosregistration.Load(lookup, "RUNTIME_B", instanceID)
}
