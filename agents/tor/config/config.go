package config

import (
	"bytes"
	"text/template"

	v1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
	"github.com/cockroachdb/errors"
)

const configFormat = `
SocksPort {{ .SocksPort }}
ControlPort {{ .ControlPort }}
MetricsPort {{ .MetricsPort }}
MetricsPortPolicy {{ .MetricsPortPolicy }}
HiddenServiceDir {{ .ServiceDir }}
{{ if .HiddenServiceOnionbalanceInstance }}
HiddenServiceOnionbalanceInstance 1
{{ end }}
HiddenServiceVersion {{ .Version }}
{{ range .Ports }}
HiddenServicePort {{ .PublicPort }} {{ .ServiceClusterIP }}:{{ .ServicePort }}
{{ end }}

{{ if .ExtraConfig }}
# ExtraConfig [START]
{{ .ExtraConfig }}
# ExtraConfig [END]
{{ end }}
`

const oBconfigFormat = `
{{ if .HiddenServiceOnionbalanceInstance }}
MasterOnionAddress {{.MasterOnionAddress}}
{{ end }}
`

var (
	configTemplate   = template.Must(template.New("config").Parse(configFormat))
	oBconfigTemplate = template.Must(template.New("config").Parse(oBconfigFormat))
)

type TorConfig struct {
	SocksPort                         string
	ControlPort                       string
	MetricsPort                       string
	MetricsPortPolicy                 string
	ServiceName                       string
	ServiceNamespace                  string
	ServiceDir                        string
	Version                           int
	Ports                             []portTuple
	MasterOnionAddress                string
	HiddenServiceOnionbalanceInstance bool
	ExtraConfig                       string
}

type portTuple struct {
	ServicePort      int32
	PublicPort       int32
	ServiceClusterIP string
}

func OnionServiceInputData(onion *v1alpha2.OnionService) TorConfig {
	ports := []portTuple{}

	for _, rule := range onion.Spec.Rules {
		port := portTuple{
			ServicePort:      rule.Backend.Service.Port.Number,
			PublicPort:       rule.Port.Number,
			ServiceClusterIP: rule.Backend.Service.Name,
		}
		ports = append(ports, port)
	}

	return TorConfig{
		SocksPort:                         "0",
		ControlPort:                       "0",
		MetricsPort:                       "0.0.0.0:9035",
		MetricsPortPolicy:                 "accept 0.0.0.0/0",
		ServiceName:                       onion.ServiceName(),
		ServiceNamespace:                  onion.Namespace,
		ServiceDir:                        "/run/tor/service",
		Ports:                             ports,
		Version:                           onion.Spec.GetVersion(),
		MasterOnionAddress:                onion.Spec.MasterOnionAddress,
		HiddenServiceOnionbalanceInstance: onion.Spec.MasterOnionAddress != "",
		ExtraConfig:                       onion.Spec.ExtraConfig,
	}
}

func TorConfigForService(onion *v1alpha2.OnionService) (string, error) {
	s := OnionServiceInputData(onion)

	var tmp bytes.Buffer

	err := configTemplate.Execute(&tmp, s)
	if err != nil {
		return "", errors.Wrap(err, "Error while Marshaling. %v")
	}

	return tmp.String(), nil
}

// Generates ob_config file if this instance handles traffic on behalf of a master hidden service.
func ObConfigForService(onion *v1alpha2.OnionService) (string, error) {
	s := OnionServiceInputData(onion)

	var tmp bytes.Buffer

	err := oBconfigTemplate.Execute(&tmp, s)
	if err != nil {
		return "", errors.Wrap(err, "Error while Marshaling. %v")
	}

	return tmp.String(), nil
}
