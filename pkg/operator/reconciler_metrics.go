package operator

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/go-jsonnet"
	"github.com/grafana/agent/pkg/operator/assets"
	"github.com/grafana/agent/pkg/operator/clientutil"
	"github.com/grafana/agent/pkg/operator/config"
	core_v1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// createMetricsConfigurationSecret creates the Grafana Agent metrics configuration and stores
// it into a secret.
func (r *reconciler) createMetricsConfigurationSecret(
	ctx context.Context,
	l log.Logger,
	d config.Deployment,
	s assets.SecretStore,
) error {
	return r.createTelemetryConfigurationSecret(ctx, l, d, s, config.MetricsType)
}

func (r *reconciler) createTelemetryConfigurationSecret(
	ctx context.Context,
	l log.Logger,
	d config.Deployment,
	s assets.SecretStore,
	ty config.Type,
) error {

	var shouldCreate bool
	key := types.NamespacedName{Namespace: d.Agent.Namespace}

	switch ty {
	case config.MetricsType:
		key.Name = fmt.Sprintf("%s-config", d.Agent.Name)
		shouldCreate = len(d.Metrics) > 0
	case config.LogsType:
		key.Name = fmt.Sprintf("%s-logs-config", d.Agent.Name)
		shouldCreate = len(d.Logs) > 0
	default:
		return fmt.Errorf("unknown telemetry type %s", ty)
	}

	// Delete the old Secret if one exists and we have nothing to create.
	if !shouldCreate {
		var secret core_v1.Secret
		err := r.Client.Get(ctx, key, &secret)
		if k8s_errors.IsNotFound(err) || !isManagedResource(&secret) {
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to find stale secret %s: %w", key, err)
		}

		err = r.Client.Delete(ctx, &secret)
		if err != nil {
			return fmt.Errorf("failed to delete stale secret %s: %w", key, err)
		}
		return nil
	}

	rawConfig, err := d.BuildConfig(s, ty)

	var jsonnetError jsonnet.RuntimeError
	if errors.As(err, &jsonnetError) {
		// Dump Jsonnet errors to the console to retain newlines and make them
		// easier to digest.
		fmt.Fprintf(os.Stderr, "%s", jsonnetError.Error())
	}
	if err != nil {
		return fmt.Errorf("unable to build config: %w", err)
	}

	blockOwnerDeletion := true

	secret := core_v1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Namespace: key.Namespace,
			Name:      key.Name,
			Labels:    r.config.Labels.Merge(managedByOperatorLabels),
			OwnerReferences: []v1.OwnerReference{{
				APIVersion:         d.Agent.APIVersion,
				BlockOwnerDeletion: &blockOwnerDeletion,
				Kind:               d.Agent.Kind,
				Name:               d.Agent.Name,
				UID:                d.Agent.UID,
			}},
		},
		Data: map[string][]byte{"agent.yml": []byte(rawConfig)},
	}

	level.Info(l).Log("msg", "reconciling secret", "secret", secret.Name)
	err = clientutil.CreateOrUpdateSecret(ctx, r.Client, &secret)
	if err != nil {
		return fmt.Errorf("failed to reconcile secret: %w", err)
	}
	return nil
}
