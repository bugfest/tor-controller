package v1alpha2

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

const (
	torDeploymentNameFmt     = "%s-tor-daemon"
	torSecretNameFmt         = "%s-tor-secret"
	torServiceNameFmt        = "%s-tor-svc"
	torMetricsServiceNameFmt = "%s-tor-metrics-svc"
	torRoleNameFmt           = "%s-tor-role"
	torServiceAccountNameFmt = "%s-tor-sa"
	torConfigMapFmt          = "%s-tor-config"

	dnsPort        = 53
	natdPort       = 8082
	httpTunnelPort = 8080
	transPort      = 8081
	socksPort      = 9050
	controlPort    = 9051
	metricsPort    = 9035
	serverPort     = 9999
)

func (tor *Tor) DeploymentName() string {
	return fmt.Sprintf(torDeploymentNameFmt, tor.Name)
}

func (tor *Tor) ConfigMapName() string {
	return fmt.Sprintf(torConfigMapFmt, tor.Name)
}

func (tor *Tor) InstanceName() string {
	return fmt.Sprintf(torServiceNameFmt, tor.Name)
}

func (tor *Tor) ServiceMetricsName() string {
	return fmt.Sprintf(torMetricsServiceNameFmt, tor.Name)
}

func (tor *Tor) ServiceMetricsSelector() map[string]string {
	return map[string]string{
		"app":        tor.ServiceMetricsName(),
		"controller": tor.Name,
	}
}

func (tor *Tor) ServiceSelector() map[string]string {
	serviceSelector := map[string]string{
		"app":        tor.InstanceName(),
		"controller": tor.Name,
	}

	return serviceSelector
}

func (tor *Tor) ServiceName() string {
	return fmt.Sprintf(torServiceNameFmt, tor.Name)
}

func (tor *Tor) SecretName() string {
	return fmt.Sprintf(torSecretNameFmt, tor.Name)
}

func (tor *Tor) DeploymentLabels() map[string]string {
	return tor.ServiceSelector()
}

func (tor *Tor) RoleName() string {
	return fmt.Sprintf(torRoleNameFmt, tor.Name)
}

func (tor *Tor) ServiceAccountName() string {
	return fmt.Sprintf(torServiceAccountNameFmt, tor.Name)
}

// Set default vaules port all the Tor ports.
func (tor *Tor) SetTorDefaults() {
	tor.Spec.Client.DNS.setPortsDefaults(dnsPort)
	tor.Spec.Client.NATD.setPortsDefaults(natdPort)
	tor.Spec.Client.HTTPTunnel.setPortsDefaults(httpTunnelPort)
	tor.Spec.Client.Trans.setPortsDefaults(transPort)
	tor.Spec.Client.Socks.setPortsDefaults(socksPort)
	tor.Spec.Control.setPortsDefaults(controlPort)
	tor.Spec.Metrics.setPortsDefaults(metricsPort)
	tor.Spec.Server.setPortsDefaults(serverPort)

	if tor.Spec.Client.TransProxyType == "" {
		tor.Spec.Client.TransProxyType = "default"
	}

	anyPortEnabled := false
	// Loop thru available ports but metrics
	for _, enabled := range []bool{
		tor.Spec.Client.DNS.Enable,
		tor.Spec.Client.NATD.Enable,
		tor.Spec.Client.HTTPTunnel.Enable,
		tor.Spec.Client.Trans.Enable,
		tor.Spec.Client.Socks.Enable,
		tor.Spec.Server.Enable,
	} {
		if enabled {
			anyPortEnabled = true
		}
	}

	if !anyPortEnabled {
		// if no client or server port is enabled, socks is the default
		tor.Spec.Client.Socks.Enable = true
	}
}

// Set default values for port number, address and policy.
func (torPort *TorGenericPortWithFlagSpec) setPortsDefaults(portDefault int32) {
	defaultAddress := []string{"0.0.0.0", "::"}
	if len(torPort.Address) == 0 {
		torPort.Address = defaultAddress
	}

	if len(torPort.Policy) == 0 {
		torPort.Policy = []string{"accept 0.0.0.0/0", "accept ::/0"}
	}

	if torPort.Port == 0 {
		torPort.Port = portDefault
	}
}

// Retrieves an array of TorGenericPortDef with their protocols and port details.
func (tor *Tor) GetAllPorts() []TorGenericPortDef {
	return []TorGenericPortDef{
		// Control
		{
			Name:     "control",
			Protocol: "TCP",
			Port:     tor.Spec.Control.TorGenericPortSpec,
		},

		// Metrics
		{
			Name:     "metrics",
			Protocol: "TCP",
			Port:     tor.Spec.Metrics.TorGenericPortSpec,
		},

		// Server
		{
			Name:     "server",
			Protocol: "TCP",
			Port:     tor.Spec.Server.TorGenericPortSpec,
		},

		// Client
		{
			Name:     "dns",
			Protocol: "UDP",
			Port:     tor.Spec.Client.DNS.TorGenericPortSpec,
		},

		{
			Name:     "httptunnel",
			Protocol: "TCP",
			Port:     tor.Spec.Client.HTTPTunnel.TorGenericPortSpec,
		},

		{
			Name:     "natd",
			Protocol: "TCP",
			Port:     tor.Spec.Client.NATD.TorGenericPortSpec,
		},

		{
			Name:     "socks",
			Protocol: "TCP",
			Port:     tor.Spec.Client.Socks.TorGenericPortSpec,
		},

		{
			Name:     "trans",
			Protocol: "TCP",
			Port:     tor.Spec.Client.Trans.TorGenericPortSpec,
		},
	}
}

func (tor *Tor) PodTemplate() corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: tor.Spec.Template.ObjectMeta,
		Spec:       tor.Spec.Template.Spec,
	}
}

func (tor *Tor) Resources() corev1.ResourceRequirements {
	return tor.Spec.Template.Resources
}
