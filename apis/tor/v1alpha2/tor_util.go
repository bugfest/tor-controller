package v1alpha2

import "fmt"

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

func (s *Tor) DeploymentLabels() map[string]string {
	return s.ServiceSelector()
}

func (s *Tor) RoleName() string {
	return fmt.Sprintf(torRoleNameFmt, s.Name)
}

func (s *Tor) ServiceAccountName() string {
	return fmt.Sprintf(torServiceAccountNameFmt, s.Name)
}

func (p *TorGenericPortSpec) DefaultPort(port int32) TorGenericPortSpec {
	var new TorGenericPortSpec = *p.DeepCopy()
	if p.Port == int32(0) {
		new.Port = port
	}
	return new
}

func (p *TorGenericPortSpec) DefaultEnable(enable bool) TorGenericPortSpec {
	var new TorGenericPortSpec = *p.DeepCopy()
	if enable {
		p.Enable = true
	}
	return new
}

func (s *Tor) GetAllPorts() []TorGenericPortDef {
	var ports = []TorGenericPortDef{}

	// Control
	ports = append(ports, TorGenericPortDef{Name: "control",
		Protocol: "TCP",
		Port:     s.Spec.Control.TorGenericPortSpec.DefaultPort(9051)},
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
		Port:     s.Spec.Client.HTTPtunnel.TorGenericPortSpec},
	)
	ports = append(ports, TorGenericPortDef{Name: "natd",
		Protocol: "TCP",
		Port:     s.Spec.Client.NATD.TorGenericPortSpec},
	)

	// socks always default
	s.Spec.Client.Socks.TorGenericPortSpec.Enable = true
	if s.Spec.Client.Socks.TorGenericPortSpec.Port == 0 {
		s.Spec.Client.Socks.TorGenericPortSpec.Port = 9050
	}
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
