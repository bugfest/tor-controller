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
)

func (s *Tor) DeploymentName() string {
	return fmt.Sprintf(torDeploymentNameFmt, s.Name)
}

func (s *Tor) ConfigMapName() string {
	return fmt.Sprintf(torConfigMapFmt, s.Name)
}

func (s *Tor) InstanceName() string {
	return fmt.Sprintf(torServiceNameFmt, s.Name)
}

func (s *Tor) ServiceMetricsName() string {
	return fmt.Sprintf(torMetricsServiceNameFmt, s.Name)
}

func (s *Tor) ServiceMetricsSelector() map[string]string {
	return map[string]string{
		"app":        s.ServiceMetricsName(),
		"controller": s.Name,
	}
}

func (s *Tor) ServiceSelector() map[string]string {
	serviceSelector := map[string]string{
		"app":        s.InstanceName(),
		"controller": s.Name,
	}
	return serviceSelector
}

func (s *Tor) ServiceName() string {
	return fmt.Sprintf(torServiceNameFmt, s.Name)
}

func (s *Tor) SecretName() string {
	return fmt.Sprintf(torSecretNameFmt, s.Name)
}

func (s *Tor) DeploymentLabels() map[string]string {
	return s.ServiceSelector()
}

func (s *Tor) RoleName() string {
	return fmt.Sprintf(torRoleNameFmt, s.Name)
}

func (s *Tor) ServiceAccountName() string {
	return fmt.Sprintf(torServiceAccountNameFmt, s.Name)
}

// func (p *TorGenericPortSpec) DefaultPort(port int32) TorGenericPortSpec {
// 	var new TorGenericPortSpec = *p.DeepCopy()
// 	if p.Port == int32(0) {
// 		new.Port = port
// 	}
// 	return new
// }

// func (p *TorGenericPortSpec) DefaultEnable(enable bool) TorGenericPortSpec {
// 	var new TorGenericPortSpec = *p.DeepCopy()
// 	if enable {
// 		p.Enable = true
// 	}
// 	return new
// }

// Set default vaules port all the Tor ports
func (tor *Tor) SetTorDefaults() {
	tor.Spec.Client.DNS.setPortsDefaults(53)
	tor.Spec.Client.NATD.setPortsDefaults(8082)
	tor.Spec.Client.HTTPTunnel.setPortsDefaults(8080)
	tor.Spec.Client.Trans.setPortsDefaults(8081)
	tor.Spec.Client.Socks.setPortsDefaults(9050)
	tor.Spec.Control.setPortsDefaults(9051)
	tor.Spec.Metrics.setPortsDefaults(9035)
	tor.Spec.Server.setPortsDefaults(9999)
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

// Set default values for port number, address and policy
func (torPort *TorGenericPortWithFlagSpec) setPortsDefaults(portDefault int32) {
	defaultAddress := "0.0.0.0"
	if torPort.Address == "" {
		torPort.Address = defaultAddress
	}
	if len(torPort.Policy) == 0 {
		torPort.Policy = []string{"accept 0.0.0.0"}
	}
	if torPort.Port == 0 {
		torPort.Port = portDefault
	}
}

// Retrieves an array of TorGenericPortDef with their protocols and port details
func (s *Tor) GetAllPorts() []TorGenericPortDef {
	var ports = []TorGenericPortDef{}

	// Control
	ports = append(ports, TorGenericPortDef{Name: "control",
		Protocol: "TCP",
		Port:     s.Spec.Control.TorGenericPortSpec},
	)

	// Metrics
	ports = append(ports, TorGenericPortDef{Name: "metrics",
		Protocol: "TCP",
		Port:     s.Spec.Metrics.TorGenericPortSpec},
	)

	// Server
	ports = append(ports, TorGenericPortDef{Name: "server",
		Protocol: "TCP",
		Port:     s.Spec.Server.TorGenericPortSpec},
	)

	// Client
	ports = append(ports, TorGenericPortDef{Name: "dns",
		Protocol: "UDP",
		Port:     s.Spec.Client.DNS.TorGenericPortSpec},
	)
	ports = append(ports, TorGenericPortDef{Name: "httptunnel",
		Protocol: "TCP",
		Port:     s.Spec.Client.HTTPTunnel.TorGenericPortSpec},
	)
	ports = append(ports, TorGenericPortDef{Name: "natd",
		Protocol: "TCP",
		Port:     s.Spec.Client.NATD.TorGenericPortSpec},
	)
	ports = append(ports, TorGenericPortDef{Name: "socks",
		Protocol: "TCP",
		Port:     s.Spec.Client.Socks.TorGenericPortSpec},
	)
	ports = append(ports, TorGenericPortDef{Name: "trans",
		Protocol: "TCP",
		Port:     s.Spec.Client.Trans.TorGenericPortSpec},
	)

	return ports
}

func (s *Tor) PodTemplate() corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: s.Spec.Template.ObjectMeta,
		Spec:       s.Spec.Template.Spec,
	}
}

func (s *Tor) Resources() corev1.ResourceRequirements {
	return s.Spec.Template.Resources
}
