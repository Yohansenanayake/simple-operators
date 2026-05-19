/*
Copyright 2026.

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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	computev1 "github.com/Yohansenanayake/simple-operators/api/v1"
)

// Ec2InstanceReconciler reconciles a Ec2Instance object
type Ec2InstanceReconciler struct {
	client.Client //help to communicate with k8s api server
	Scheme        *runtime.Scheme
}

// Kubebuilder RBAC makers to access EC2Instance resources across namespaces
// +kubebuilder:rbac:groups=compute.yohancloud.com,resources=ec2instances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=compute.yohancloud.com,resources=ec2instances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=compute.yohancloud.com,resources=ec2instances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Ec2Instance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *Ec2InstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := logf.FromContext(ctx)

	l.Info("===== Reconcile Loop Started =====", "namespace", req.Namespace, "name", req.Name)
	ec2Instance := &computev1.Ec2Instance{}

	if err := r.Get(ctx, req.NamespacedName, ec2Instance); err != nil {
		if errors.IsNotFound(err) {
			l.Info("Ec2Instance resource not found. Ignoring since object must be deleted.")
			// k8s will not retry - done with this request , wait for next event
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		l.Error(err, "Failed to get EC2Instance object, retry with backoff")
		return ctrl.Result{}, err
	}

	// check if we already have an instance ID in the status 
	// This prevents creating multiple EC2 instances if the Reconcile function is triggered  multiple times for the same CR (CR modified) before the status is updated with the instance ID.
	if ec2Instance.Status.InstanceID != "" {
		l.Info("Instance already exists for this CR, skipping creation", "instanceID", ec2Instance.Status.InstanceID)
		return ctrl.Result{}, nil
	}

	l.Info("==== Creating New Instance ====")

	// Add a finalizer so Kubernetes keeps the Ec2Instance during deletion until the controller cleans up the external EC2 instance.
	l.Info("=== ABOUT TO ADD FINALIZER ===")
	ec2Instance.Finalizers = append(ec2Instance.Finalizers, "ec2instance.yohancloud.com")
	if err := r.Update(ctx, ec2Instance); err != nil {
		l.Error(err, "Failed to add finalizer")
		return ctrl.Result{}, err // Result{} is ignored since err
	}
	l.Info("==== FINALIZER ADDED - This update will trigger a NEW Reconcile loop , but current reconcile continues ====")

	l.Info(" === CONTINUE WITH EC2 INSTANCE CREATION IN CURRENT RECONCILE ====")
	createdInstanceInfo, err := createEc2Instance(ec2Instance)
	if err != nil {
		l.Error(err, "Failed to create EC2 instance")
		return ctrl.Result{}, err
	}

	l.Info("=== ABOUT TO UPDATE STATUS - This will trigger reconcile loop again ===",
		"instanceID", createdInstanceInfo.InstanceID,
		"state", createdInstanceInfo.State)

	// Update the status of the CR with the instance information
	ec2Instance.Status.InstanceID = createdInstanceInfo.InstanceID
	ec2Instance.Status.State = createdInstanceInfo.State
	ec2Instance.Status.PublicIP = createdInstanceInfo.PublicIP
	ec2Instance.Status.PrivateIP = createdInstanceInfo.PrivateIP
	ec2Instance.Status.PublicDNS = createdInstanceInfo.PublicDNS
	ec2Instance.Status.PrivateDNS = createdInstanceInfo.PrivateDNS
	ec2Instance.Status.LaunchTime = createdInstanceInfo.LaunchTime

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Ec2InstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&computev1.Ec2Instance{}).
		Named("ec2instance").
		Complete(r)
}
