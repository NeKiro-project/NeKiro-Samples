package runtimeb

import "github.com/NeKiro-project/NeKiro-Samples/internal/nacosregistration"

const (
	RegistrationModeEnvironment       = "RUNTIME_B_REGISTRATION_MODE"
	AgentCardVersionEnvironment       = "RUNTIME_B_AGENT_CARD_VERSION"
	ReleaseIDEnvironment              = "RUNTIME_B_RELEASE_ID"
	CardDigestEnvironment             = "RUNTIME_B_CARD_DIGEST"
	CanonicalEndpointEnvironment      = "RUNTIME_B_CANONICAL_ENDPOINT"
	AudienceEnvironment               = "RUNTIME_B_AUDIENCE"
	NacosAPIOriginEnvironment         = "RUNTIME_B_NACOS_API_ORIGIN"
	NacosNamespaceEnvironment         = "RUNTIME_B_NACOS_NAMESPACE_ID"
	NacosGroupEnvironment             = "RUNTIME_B_NACOS_GROUP_NAME"
	NacosServiceEnvironment           = "RUNTIME_B_NACOS_SERVICE_NAME"
	NacosClusterEnvironment           = "RUNTIME_B_NACOS_CLUSTER_NAME"
	NacosPortNameEnvironment          = "RUNTIME_B_NACOS_PORT_NAME"
	NacosAdvertisedIPEnvironment      = "RUNTIME_B_NACOS_ADVERTISED_IP"
	NacosAdvertisedPortEnvironment    = "RUNTIME_B_NACOS_ADVERTISED_PORT"
	NacosWeightEnvironment            = "RUNTIME_B_NACOS_WEIGHT"
	NacosHeartbeatIntervalEnvironment = "RUNTIME_B_NACOS_HEARTBEAT_INTERVAL_MS"
	NacosHeartbeatTimeoutEnvironment  = "RUNTIME_B_NACOS_HEARTBEAT_TIMEOUT_MS"
	NacosIPDeleteTimeoutEnvironment   = "RUNTIME_B_NACOS_IP_DELETE_TIMEOUT_MS"
	NacosRequestTimeoutEnvironment    = "RUNTIME_B_NACOS_REQUEST_TIMEOUT_MS"
	NacosAuthModeEnvironment          = "RUNTIME_B_NACOS_AUTH_MODE"
	NacosAccessTokenEnvironment       = "RUNTIME_B_NACOS_ACCESS_TOKEN"
	NacosTLSCAFileEnvironment         = "RUNTIME_B_NACOS_TLS_CA_FILE"
	NacosTLSServerNameEnvironment     = "RUNTIME_B_NACOS_TLS_SERVER_NAME"
	NacosTLSClientCertEnvironment     = "RUNTIME_B_NACOS_TLS_CLIENT_CERT_FILE"
	NacosTLSClientKeyEnvironment      = "RUNTIME_B_NACOS_TLS_CLIENT_KEY_FILE"
	RegistrationModeDisabled          = nacosregistration.ModeDisabled
	RegistrationModeNacos             = nacosregistration.ModeNacos
	NacosAuthNone                     = nacosregistration.AuthNone
	NacosAuthAccessToken              = nacosregistration.AuthAccessToken
)

type RegistrationConfig = nacosregistration.Config

func LoadRegistrationConfig(lookup func(string) (string, bool), agentID, instanceID string) (RegistrationConfig, error) {
	return nacosregistration.Load(lookup, "RUNTIME_B", agentID, instanceID)
}
