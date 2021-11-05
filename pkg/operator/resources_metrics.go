package operator

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultPortName = "http-metrics"
)

var (
	managedByOperatorLabel      = "app.kubernetes.io/managed-by"
	managedByOperatorLabelValue = "grafana-agent-operator"
	managedByOperatorLabels     = map[string]string{
		managedByOperatorLabel: managedByOperatorLabelValue,
	}
	agentNameLabelName        = "operator.agent.grafana.com/name"
	agentTypeLabel            = "operator.agent.grafana.com/type"
	probeTimeoutSeconds int32 = 3
)

// isManagedResource returns true if the given object has a managed-by
// grafana-agent-operator label.
func isManagedResource(obj client.Object) bool {
	labelValue := obj.GetLabels()[managedByOperatorLabel]
	return labelValue == managedByOperatorLabelValue
}
