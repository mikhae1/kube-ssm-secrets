/*
Copyright 2023.

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

package controller

import (
	"context"
	"encoding/json"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const SSM_ANN_PREFIX = "secret.ssm-parameter"

// SSMParamReconciler reconciles a SSMParam object
type SSMParamReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ssm.secrets.github.io,resources=ssmparams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ssm.secrets.github.io,resources=ssmparams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ssm.secrets.github.io,resources=ssmparams/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SSMParam object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *SSMParamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	log := log.Log.WithValues(SSM_ANN_PREFIX, req.NamespacedName)
	log.Info("Reconciling...")

	// Fetch the ServiceAccount or Deployment object
	var serviceAccount corev1.ServiceAccount
	err := r.Get(ctx, req.NamespacedName, &serviceAccount)
	if err != nil {
		log.Error(err, "unable to fetch ServiceAccount")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// "secret.ssm-parameter/"
	for key, value := range serviceAccount.Annotations {
		if strings.HasPrefix(key, SSM_ANN_PREFIX+"/") {

			secretName := strings.TrimPrefix(key, SSM_ANN_PREFIX+"/")
			secretNamespace := req.Namespace

			secretData := map[string]string{}
			// Process the annotation value
			for _, pair := range strings.Split(value, ",") {
				keyValue := strings.Split(pair, "=")
				if len(keyValue) == 2 {
					key := keyValue[0]
					ssmParameterPath := keyValue[1]

					ssmParameterValue, err := getSSMParameterValue(ssmParameterPath)
					if err != nil {
						log.Error(err, "unable to fetch SSM parameter value for: "+pair)
						return ctrl.Result{}, err
					}
					secretData[key] = ssmParameterValue
				} else if len(keyValue) == 1 {
					// Assuming the SSM parameter contains a JSON object
					ssmParameterPath := keyValue[0]
					ssmParameterValue, err := getSSMParameterValue(ssmParameterPath)
					if err != nil {
						log.Error(err, "unable to fetch SSM parameter value")
						return ctrl.Result{}, err
					}
					var jsonData map[string]string
					if err := json.Unmarshal([]byte(ssmParameterValue), &jsonData); err != nil {
						log.Error(err, "failed to unmarshal JSON SSM parameter value")
						return ctrl.Result{}, err
					}
					for k, v := range jsonData {
						secretData[k] = v
					}
				}
			}

			// Check if the Secret already exists
			var existingSecret corev1.Secret
			err = r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNamespace}, &existingSecret)
			if err != nil && errors.IsNotFound(err) {
				// Create a new Secret
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: secretNamespace,
					},
					StringData: secretData,
				}
				if err := r.Create(ctx, secret); err != nil {
					log.Error(err, "failed to create Secret")
					return ctrl.Result{}, err
				}
				log.Info("created Secret", "namespace", secretNamespace, "name", secretName)
			} else if err == nil {
				// Update the existing Secret
				existingSecret.StringData = secretData
				if err := r.Update(ctx, &existingSecret); err != nil {
					log.Error(err, "failed to update Secret")
					return ctrl.Result{}, err
				}
				log.Info("updated Secret", "namespace", secretNamespace, "name", secretName)
			} else {
				log.Error(err, "failed to get Secret")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SSMParamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// return ctrl.NewControllerManagedBy(mgr).
	// 	For(&corev1.ServiceAccount{}).
	// 	Complete(r)
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ServiceAccount{}, builder.WithPredicates(serviceAccountPredicate())).
		Complete(r)
}

// Reconcile only when the ServiceAccount is updated and has the expected annotation
func serviceAccountPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			for key := range e.ObjectNew.GetAnnotations() {
				if strings.HasPrefix(key, SSM_ANN_PREFIX) {
					return true
				}
			}
			return false
		},
		CreateFunc: func(e event.CreateEvent) bool {
			for key := range e.Object.GetAnnotations() {
				if strings.HasPrefix(key, SSM_ANN_PREFIX) {
					return true
				}
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func getSSMParameterValue(ssmParameterPath string) (string, error) {
	// Implement your logic to fetch the SSM parameter value using the AWS SDK for Go
	// ...
	return "A secret from: " + ssmParameterPath, nil
}
