package runtimeb

import "github.com/NeKiro-project/NeKiro-Samples/internal/nacosregistration"

type HTTPDoer = nacosregistration.HTTPDoer
type NacosRegistration = nacosregistration.Registration

func NewNacosRegistration(config RegistrationConfig, executor HTTPDoer) (*NacosRegistration, error) {
	return nacosregistration.New(config, executor)
}
