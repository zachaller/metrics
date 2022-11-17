/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package prometheus

import (
	"context"
	prometheusv1 "github.com/argoproj/metrics/pkg/apis/prometheus/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// MetricQueryReconciler reconciles a MetricQuery object
type MetricQueryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=prometheus,resources=metricqueries,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=prometheus,resources=metricqueries/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=prometheus,resources=metricqueries/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MetricQuery object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *MetricQueryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var metricQuery prometheusv1.MetricQuery
	err := r.Client.Get(ctx, req.NamespacedName, &metricQuery)
	if err != nil {
		return ctrl.Result{}, err
	}

	var metricQueryRun prometheusv1.MetricQueryRun
	r.Client.Get(ctx, req.NamespacedName, &metricQueryRun) // ignore error because we want to create it if it doesn't exist
	if metricQueryRun.Name != "" {
		return ctrl.Result{}, err
	}

	annotations := metricQuery.GetAnnotations()
	delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")

	metricQueryRun = prometheusv1.MetricQueryRun{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:        metricQuery.Name,
			Namespace:   metricQuery.Namespace,
			Labels:      metricQuery.Labels,
			Annotations: annotations,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: metricQuery.APIVersion,
				Kind:       metricQuery.Kind,
				Name:       metricQuery.Name,
				UID:        metricQuery.UID,
			}},
		},
		Spec: prometheusv1.MetricQueryRunSpec{},
	}
	err = r.Client.Create(ctx, &metricQueryRun)
	if err != nil {
		return ctrl.Result{}, err
	}

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MetricQueryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&prometheusv1.MetricQuery{}).
		Complete(r)
}
