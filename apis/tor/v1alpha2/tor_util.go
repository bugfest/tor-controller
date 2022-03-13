package v1alpha2

import "fmt"

const (
	torDeploymentNameFmt       = "%s-tor-daemon"
	torSecretNameFmt           = "%s-tor-secret"
	torServiceNameFmt          = "%s-tor-svc"
	torMetricsServiceNameFmt   = "%s-tor-metrics-svc"
	torRoleNameFmt             = "%s-tor-role"
	torServiceAccountNameFmt   = "%s-tor-sa"
	onionServiceBackendNameFmt = "%s-tor-obb-%d"
)

func (s *Tor) DeploymentName() string {
	return fmt.Sprintf(torDeploymentNameFmt, s.Name)
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

func (s *Tor) DeploymentLabels() map[string]string {
	return s.ServiceSelector()
}

func (s *Tor) RoleName() string {
	return fmt.Sprintf(torRoleNameFmt, s.Name)
}

func (s *Tor) ServiceAccountName() string {
	return fmt.Sprintf(torServiceAccountNameFmt, s.Name)
}
