package v1alpha2

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

const (
	torDeploymentNameFmt       = "%s-tor-daemon"
	torSecretNameFmt           = "%s-tor-secret"
	torServiceNameFmt          = "%s-tor-svc"
	torMetricsServiceNameFmt   = "%s-tor-metrics-svc"
	torRoleNameFmt             = "%s-tor-role"
	torServiceAccountNameFmt   = "%s-tor-sa"
	onionServiceBackendNameFmt = "%s-tor-obb-%d"
)

func (s *OnionServiceSpec) GetVersion() int {
	v := 3
	if s.Version == 2 {
		v = 2
	}
	return v
}

func (s *OnionBalancedService) OnionServiceBackendName(n int32) string {
	return fmt.Sprintf(onionServiceBackendNameFmt, s.Name, n)
}

func (s *OnionService) DeploymentName() string {
	return fmt.Sprintf(torDeploymentNameFmt, s.Name)
}

func (s *OnionService) ServiceName() string {
	return fmt.Sprintf(torServiceNameFmt, s.Name)
}

func (s *OnionService) ServiceMetricsName() string {
	return fmt.Sprintf(torMetricsServiceNameFmt, s.Name)
}

func (s *OnionService) ServiceMetricsSelector() map[string]string {
	return map[string]string{
		"app":        s.ServiceMetricsName(),
		"controller": s.Name,
	}
}

func (s *OnionService) SecretName() string {
	if len(s.Spec.PrivateKeySecret.Name) > 0 {
		return s.Spec.PrivateKeySecret.Name
	}
	return fmt.Sprintf(torSecretNameFmt, s.Name)
}

func (s *OnionService) ServiceSelector() map[string]string {
	serviceSelector := map[string]string{
		"app":        s.ServiceName(),
		"controller": s.Name,
	}
	return serviceSelector
}

func (s *OnionService) DeploymentLabels() map[string]string {
	return s.ServiceSelector()
}

func (s *OnionService) RoleName() string {
	return fmt.Sprintf(torRoleNameFmt, s.Name)
}

func (s *OnionService) ServiceAccountName() string {
	return fmt.Sprintf(torServiceAccountNameFmt, s.Name)
}

func (s *OnionService) PodTemplate() corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: s.Spec.Template.ObjectMeta,
		Spec:       s.Spec.Template.Spec,
	}
}

func (s *OnionService) Resources() corev1.ResourceRequirements {
	return s.Spec.Template.Resources
}
