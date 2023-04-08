package v1alpha2

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

const (
	onionbalanceDeploymentNameFmt     = "%s-tor-daemon"
	onionbalanceSecretNameFmt         = "%s-tor-secret"
	onionbalanceServiceNameFmt        = "%s-tor-svc"
	onionbalanceRoleNameFmt           = "%s-tor-role"
	onionbalanceServiceAccountNameFmt = "%s-tor-sa"
	onionbalanceConfigMapFmt          = "%s-tor-config"
)

func (s *OnionBalancedServiceSpec) GetVersion() int {
	v := 3
	if s.Version == 2 {
		v = 2
	}

	return v
}

func (s *OnionBalancedServiceSpec) GetBackends() int {
	return int(s.Backends)
}

func (s *OnionBalancedService) DeploymentName() string {
	return fmt.Sprintf(osDeploymentNameFmt, s.Name)
}

func (s *OnionBalancedService) ConfigMapName() string {
	return fmt.Sprintf(onionbalanceConfigMapFmt, s.Name)
}

func (s *OnionBalancedService) ServiceName() string {
	return fmt.Sprintf(osServiceNameFmt, s.Name)
}

func (s *OnionBalancedService) ServiceMetricsName() string {
	return fmt.Sprintf(osMetricsServiceNameFmt, s.Name)
}

func (s *OnionBalancedService) SecretName() string {
	if s.Spec.PrivateKeySecret != (SecretReference{}) {
		if len(s.Spec.PrivateKeySecret.Name) > 0 {
			return s.Spec.PrivateKeySecret.Name
		}
	}

	return fmt.Sprintf(osSecretNameFmt, s.Name)
}

func (s *OnionBalancedService) ServiceSelector() map[string]string {
	return map[string]string{
		"app":        s.ServiceName(),
		"controller": s.Name,
	}
}

func (s *OnionBalancedService) ServiceMetricsSelector() map[string]string {
	return map[string]string{
		"app":        s.ServiceMetricsName(),
		"controller": s.Name,
	}
}

func (s *OnionBalancedService) DeploymentLabels() map[string]string {
	return s.ServiceSelector()
}

func (s *OnionBalancedService) RoleName() string {
	return fmt.Sprintf(osRoleNameFmt, s.Name)
}

func (s *OnionBalancedService) ServiceAccountName() string {
	return fmt.Sprintf(osServiceAccountNameFmt, s.Name)
}

func (s *OnionBalancedService) IsSynced() bool {
	// All backends must exist
	if len(s.Status.Backends) != s.Spec.GetBackends() {
		return false
	}
	// All backends must have a hostname
	for _, backend := range s.Status.Backends {
		if len(backend.Hostname) == 0 {
			return false
		}
	}

	return true
}

func (s *OnionBalancedService) PodTemplate() corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: s.Spec.BalancerTemplate.ObjectMeta,
		Spec:       s.Spec.BalancerTemplate.Spec,
	}
}

func (s *OnionBalancedService) TorResources() corev1.ResourceRequirements {
	return s.Spec.BalancerTemplate.TorResources
}

func (s *OnionBalancedService) BalancerResources() corev1.ResourceRequirements {
	return s.Spec.BalancerTemplate.BalancerResources
}
