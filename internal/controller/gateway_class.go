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

	gateway "github.com/caddyserver/gateway/internal"
)

// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses/status,verbs=patch;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses/finalizers,verbs=update

type GatewayClassReconciler struct {
	client.Client

	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
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
		Type:   string(gatewayv1.GatewayClassConditionStatusAccepted),
		Status: metav1.ConditionTrue,
		Reason: string(gatewayv1.GatewayClassReasonAccepted),
		// Reason:  string(gatewayv1.GatewayClassReasonInvalidParameters),
		Message: "",
	})

	// TODO: validate CRD versions.
	meta.SetStatusCondition(&gwc.Status.Conditions, metav1.Condition{
		Type:   string(gatewayv1.GatewayClassConditionStatusSupportedVersion),
		Status: metav1.ConditionTrue,
		Reason: string(gatewayv1.GatewayClassReasonSupportedVersion),
		// Reason:  string(gatewayv1.GatewayClassReasonUnsupportedVersion),
		Message: "Gateway API CRD bundle version v1.0.0 is supported.",
	})

	supportedFeatures := []gatewayv1.SupportedFeature{
		{Name: "Gateway"},
		// {Name: "GatewayPort8080"},
		// {Name: "GatewayStaticAddresses"},
		{Name: "HTTPRoute"},
		// {Name: "HTTPRouteDestinationPortMatching"},
		// TODO: enable once we support URLRewrite Hostname
		// {Name: "HTTPRouteHostRewrite"},
		{Name: "HTTPRouteMethodMatching"},
		{Name: "HTTPRoutePathRedirect"},
		// TODO: enable once we support URLRewrite Path
		// {Name: "HTTPRoutePathRewrite"},
		{Name: "HTTPRoutePortRedirect"},
		{Name: "HTTPRouteQueryParamMatching"},
		// {Name: "HTTPRouteRequestMirror"},
		// {Name: "HTTPRouteRequestMultipleMirrors"},
		{Name: "HTTPRouteResponseHeaderModification"},
		{Name: "HTTPRouteSchemeRedirect"},
		// {Name: "Mesh"},
		{Name: "ReferenceGrant"},
		// {Name: "TLSRoute"},
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
