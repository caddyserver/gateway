// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package controller

import (
	"cmp"
	"context"
	"slices"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/pkg/features"

	gateway "github.com/caddyserver/gateway/internal"
)

// Add RBAC permissions for GatewayClasses.
//
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses/status,verbs=patch;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses/finalizers,verbs=update

type GatewayClassReconciler struct {
	client.Client

	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	Info *GatewayAPIInfo
}

var _ reconcile.Reconciler = (*GatewayClassReconciler)(nil)

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayv1.GatewayClass{}, builder.WithPredicates(predicate.NewPredicateFuncs(objectMatchesControllerName()))).
		Complete(r)
}

func objectMatchesControllerName() func(object client.Object) bool {
	return func(object client.Object) bool {
		gwc, ok := object.(*gatewayv1.GatewayClass)
		if !ok {
			return false
		}
		return gateway.MatchesControllerName(gwc.Spec.ControllerName)
	}
}

// Reconcile reconciles GatewayClass resources.
// ref; https://gateway-api.sigs.k8s.io/guides/implementers/#gatewayclass
func (r *GatewayClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Get the GatewayClass we are reconciling.
	gwc := &gatewayv1.GatewayClass{}
	if err := r.Get(ctx, req.NamespacedName, gwc); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("GatewayClass not found, ignoring reconcile request")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get GatewayClass")
		return ctrl.Result{}, err
	}
	log = log.WithValues("GatewayClass.Name", gwc.Name)

	// Check if the GatewayClass is being deleted.
	if gwc.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	// Check if the GatewayClass is using our controller.
	// ref; https://gateway-api.sigs.k8s.io/api-types/gatewayclass/#gatewayclass-controller-selection
	if !gateway.MatchesControllerName(gwc.Spec.ControllerName) {
		log.V(2).Info("Ignoring GatewayClass as it requests another controller")
		return ctrl.Result{}, nil
	}
	log.Info("Reconciling")

	// TODO: the finalizer set below says "gateways exist",
	// should we only set it if a gateway was created on the gateway class?
	// If the resource ever gets deleted then we can always remove it, but
	// if no gateways exist there isn't really anything for us to cleanup.
	//
	//if !controllerutil.ContainsFinalizer(gwc, gatewayv1.GatewayClassFinalizerGatewaysExist) {
	//	if ok := controllerutil.AddFinalizer(gwc, gatewayv1.GatewayClassFinalizerGatewaysExist); !ok {
	//		log.Error(nil, "Failed to add finalizer")
	//		return ctrl.Result{Requeue: true}, nil
	//	}
	//	if err := r.Update(ctx, gwc); err != nil {
	//		log.Error(err, "Failed to update finalizer")
	//		return ctrl.Result{}, err
	//	}
	//	// TODO: requeue?
	//}

	meta.SetStatusCondition(&gwc.Status.Conditions, metav1.Condition{
		Type:    string(gatewayv1.GatewayClassConditionStatusAccepted),
		Status:  metav1.ConditionTrue,
		Reason:  string(gatewayv1.GatewayClassReasonAccepted), // gatewayv1.GatewayClassReasonInvalidParameters
		Message: "",
	})

	meta.SetStatusCondition(&gwc.Status.Conditions, metav1.Condition{
		Type:    string(gatewayv1.GatewayClassConditionStatusSupportedVersion),
		Status:  metav1.ConditionTrue,
		Reason:  string(gatewayv1.GatewayClassReasonSupportedVersion), // gatewayv1.GatewayClassReasonUnsupportedVersion
		Message: "Gateway API CRD bundle version " + r.Info.BundleVersion + " is supported.",
	})

	supportedFeatures := []gatewayv1.SupportedFeature{
		//
		// Gateway Features
		//

		{Name: gatewayv1.FeatureName(features.SupportGateway)},
		// {Name: gatewayv1.FeatureName(features.SupportGatewayPort8080)},
		// {Name: gatewayv1.FeatureName(features.SupportGatewayStaticAddresses)},
		// {Name: gatewayv1.FeatureName(features.SupportGatewayHTTPListenerIsolation)},
		// {Name: gatewayv1.FeatureName(features.SupportGatewayInfrastructurePropagation)},
		// {Name: gatewayv1.FeatureName(features.SupportGatewayAddressEmpty)},

		//
		// HTTPRoute Features
		//

		{Name: gatewayv1.FeatureName(features.SupportHTTPRoute)},
		// {Name: gatewayv1.FeatureName(features.SupportHTTPRouteDestinationPortMatching)},
		// {Name: gatewayv1.FeatureName(features.SupportHTTPRouteBackendRequestHeaderModification)}, // TODO: do we already support this?
		{Name: gatewayv1.FeatureName(features.SupportHTTPRouteQueryParamMatching)},
		{Name: gatewayv1.FeatureName(features.SupportHTTPRouteMethodMatching)},
		{Name: gatewayv1.FeatureName(features.SupportHTTPRouteResponseHeaderModification)},
		{Name: gatewayv1.FeatureName(features.SupportHTTPRoutePortRedirect)},
		{Name: gatewayv1.FeatureName(features.SupportHTTPRouteSchemeRedirect)},
		{Name: gatewayv1.FeatureName(features.SupportHTTPRoutePathRedirect)},
		// {Name: gatewayv1.FeatureName(features.SupportHTTPRouteHostRewrite)}, // TODO: enable once we support URLRewrite Hostname
		// {Name: gatewayv1.FeatureName(features.SupportHTTPRoutePathRewrite)}, // TODO: enable once we support URLRewrite Path
		// {Name: gatewayv1.FeatureName(features.SupportHTTPRouteRequestMirror)},
		// {Name: gatewayv1.FeatureName(features.SupportHTTPRouteRequestMultipleMirrors)},
		// {Name: gatewayv1.FeatureName(features.SupportHTTPRouteRequestPercentageMirror)},
		// {Name: gatewayv1.FeatureName(features.SupportHTTPRouteRequestTimeout)},
		// {Name: gatewayv1.FeatureName(features.SupportHTTPRouteBackendTimeout)},
		// {Name: gatewayv1.FeatureName(features.SupportHTTPRouteParentRefPort)},
		{Name: gatewayv1.FeatureName(features.SupportHTTPRouteBackendProtocolH2C)},
		{Name: gatewayv1.FeatureName(features.SupportHTTPRouteBackendProtocolWebSocket)},

		//
		// Mesh Features
		//

		// {Name: gatewayv1.FeatureName(features.SupportMesh)},
		// {Name: gatewayv1.FeatureName(features.SupportMeshClusterIPMatching)},
		// {Name: gatewayv1.FeatureName(features.SupportMeshConsumerRoute)},

		//
		// Other Features
		//

		// {Name: gatewayv1.FeatureName(features.SupportGRPCRoute)},
		{Name: gatewayv1.FeatureName(features.SupportReferenceGrant)},
		{Name: gatewayv1.FeatureName(features.SupportTLSRoute)}, // TODO: only add if TLSRoute CRDs are installed?
		{Name: gatewayv1.FeatureName(features.SupportUDPRoute)}, // TODO: only add if UDPRoute CRDs are installed?
	}

	// The Gateway API spec requires that the supported features array be sorted
	// in "ascending alphabetical order".
	slices.SortFunc(supportedFeatures, func(x, y gatewayv1.SupportedFeature) int {
		return cmp.Compare(x.Name, y.Name)
	})
	gwc.Status.SupportedFeatures = supportedFeatures

	// Save changes to the GatewayClass's status.
	if err := r.Status().Update(ctx, gwc); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
