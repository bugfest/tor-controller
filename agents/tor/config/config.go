package config

import (
	"bytes"
	"text/template"

	v1alpha2 "example.com/null/tor-controller/apis/tor/v1alpha2"
)

const configFormat = `
SocksPort 0
HiddenServiceDir {{ .ServiceDir }}
HiddenServiceVersion {{ .Version }}
{{ range .Ports }}
HiddenServicePort {{ .PublicPort }} {{ $.ServiceClusterIP }}:{{ .ServicePort }}
{{ end }}
`

var configTemplate = template.Must(template.New("config").Parse(configFormat))

type onionService struct {
	ServiceName      string
	ServiceNamespace string
	ServiceClusterIP string
	ServiceDir       string
	Version          int
	Ports            []portPair
}

type portPair struct {
	ServicePort int32
	PublicPort  int32
}

func TorConfigForService(onion *v1alpha2.OnionService) (string, error) {
	ports := []portPair{}
	for _, rule := range onion.Spec.Rules {
		port := portPair{
			ServicePort: rule.Port.Number,
			PublicPort:  rule.Port.Number,
		}
		ports = append(ports, port)
	}

	s := onionService{
		ServiceName:      onion.ServiceName(),
		ServiceNamespace: onion.Namespace,
		ServiceClusterIP: onion.Status.TargetClusterIP,
		ServiceDir:       "/run/tor/service",
		Ports:            ports,
		Version:          onion.Spec.GetVersion(),
	}

	var tmp bytes.Buffer
	err := configTemplate.Execute(&tmp, s)
	if err != nil {
		return "", err
	}
	return tmp.String(), nil
}
