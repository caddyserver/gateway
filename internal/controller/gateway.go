// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package controller

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/matthewpi/certwatcher"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	gateway "github.com/caddyserver/gateway/internal"
	"github.com/caddyserver/gateway/internal/caddy"
)

// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=backendtlspolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=patch;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=referencegrants,verbs=get;list;watch

// +kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch

type GatewayReconciler struct {
	client.Client

	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	rootCAs     *x509.CertPool
	certwatcher *certwatcher.TLSConfig

	tlsConfig *tls.Config
}

var _ reconcile.Reconciler = (*GatewayReconciler)(nil)

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctrlPredicate := builder.WithPredicates(
		predicate.NewPredicateFuncs(
			hasMatchingController(context.Background(), r.Client),
		),
	)

	r.rootCAs = x509.NewCertPool()
	v, err := os.ReadFile("/var/run/secrets/tls/ca.crt")
	if err != nil {
		return fmt.Errorf("error reading ca_path: %w", err)
	}
	if ok := r.rootCAs.AppendCertsFromPEM(v); !ok {
		return errors.New("failed to load ca certificates")
	}
	r.certwatcher = &certwatcher.TLSConfig{
		CertPath: "/var/run/secrets/tls/tls.crt",
		KeyPath:  "/var/run/secrets/tls/tls.key",
		Config: &tls.Config{
			RootCAs: r.rootCAs,
		},
		DontStaple: true,
	}
	r.tlsConfig, err = r.certwatcher.GetTLSConfig(context.Background())
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayv1.Gateway{}, ctrlPredicate).
		Watches(
			&gatewayv1.GatewayClass{},
			r.enqueueRequestForOwningGatewayClass(),
			ctrlPredicate,
		).
		Watches(
			&gatewayv1alpha2.GRPCRoute{},
			r.enqueueRequestForOwningGRPCRoute(),
		).
		Watches(
			&gatewayv1.HTTPRoute{},
			r.enqueueRequestForOwningHTTPRoute(),
			builder.WithPredicates(onlyStatusChanged()),
		).
		Watches(
			&gatewayv1alpha2.TCPRoute{},
			r.enqueueRequestForOwningTCPRoute(),
			builder.WithPredicates(onlyStatusChanged()),
		).
		Watches(
			&gatewayv1alpha2.TLSRoute{},
			r.enqueueRequestForOwningTLSRoute(),
			builder.WithPredicates(onlyStatusChanged()),
		).
		Watches(
			&gatewayv1alpha2.UDPRoute{},
			r.enqueueRequestForOwningUDPRoute(),
			builder.WithPredicates(onlyStatusChanged()),
		).
		Watches(&gatewayv1alpha2.BackendTLSPolicy{}, r.enqueueRequestForTLSPolicy()).
		Watches(
			&corev1.Secret{},
			r.enqueueRequestForTLSSecret(),
			builder.WithPredicates(predicate.NewPredicateFuncs(r.usedInGateway)),
		).
		Watches(
			&corev1.Namespace{},
			r.enqueueRequestForAllowedNamespace(),
		).
		Watches(
			&corev1.Service{},
			r.enqueueRequestForOwningResource(),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(func(object client.Object) bool {
					_, found := object.GetLabels()[owningGatewayLabel]
					return found
				}),
			),
		).
		Watches(
			&corev1.Endpoints{},
			r.enqueueRequestForOwningResource(),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(func(object client.Object) bool {
					_, found := object.GetLabels()[owningGatewayLabel]
					return found
				}),
			),
		).
		Owns(&corev1.Service{}).
		Owns(&corev1.Endpoints{}).
		Complete(r)
}

// Reconcile reconciles Gateway resources.
// ref; https://gateway-api.sigs.k8s.io/guides/implementers/#gateway
func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Get the Gateway we are reconciling.
	original := &gatewayv1.Gateway{}
	if err := r.Get(ctx, req.NamespacedName, original); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(3).Info("Gateway not found, ignoring reconcile request")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Gateway")
		return ctrl.Result{}, err
	}
	log = log.WithValues("Gateway.Namespace", original.Namespace, "Gateway.Name", original.Name)

	// Ignore the gateway if it is being deleted.
	if original.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	gw := original.DeepCopy()

	// Get the Gateway Class referenced by the Gateway.
	gwc := &gatewayv1.GatewayClass{}
	if err := r.Get(ctx, client.ObjectKey{Name: string(gw.Spec.GatewayClassName)}, gwc); err != nil {
		log.V(2).Error(err, "Unable to get GatewayClass", "GatewayClass.Name", gw.Spec.GatewayClassName)
		var message string
		if apierrors.IsNotFound(err) {
			message = "GatewayClass does not exist"
		} else {
			message = "Unable to get GatewayClass"
		}
		meta.SetStatusCondition(&gw.Status.Conditions, metav1.Condition{
			Type:    string(gatewayv1.GatewayConditionAccepted),
			Status:  metav1.ConditionFalse,
			Reason:  string(gatewayv1.GatewayReasonInvalid),
			Message: message,
		})
		return r.handleReconcileErrorWithStatus(ctx, err, original, gw)
	}

	// Check if the GatewayClass is using our controller.
	// ref; https://gateway-api.sigs.k8s.io/api-types/gatewayclass/#gatewayclass-controller-selection
	if !gateway.MatchesControllerName(gwc.Spec.ControllerName) {
		log.V(2).Info("Ignoring Gateway as it requests another controller")
		return ctrl.Result{}, nil
	}

	// Check if the GatewayClass is Accepted, if not don't cannot continue.
	if c := meta.FindStatusCondition(gwc.Status.Conditions, string(gatewayv1.GatewayClassConditionStatusAccepted)); c == nil || c.Status != metav1.ConditionTrue {
		log.V(2).Info("Ignoring Gateway as it's GatewayClass isn't Accepted", "GatewayClass.Name", gwc.Name)
		return ctrl.Result{}, nil
	}
	log.Info("Reconciling")

	httpRouteList := &gatewayv1.HTTPRouteList{}
	if err := r.Client.List(ctx, httpRouteList); err != nil {
		log.Error(err, "Unable to list HTTPRoutes")
		return r.handleReconcileErrorWithStatus(ctx, err, original, gw)
	}

	grpcRouteList := &gatewayv1alpha2.GRPCRouteList{}
	if err := r.Client.List(ctx, grpcRouteList); err != nil {
		log.Error(err, "Unable to list GRPCRoutes")
		return r.handleReconcileErrorWithStatus(ctx, err, original, gw)
	}

	tcpRouteList := &gatewayv1alpha2.TCPRouteList{}
	if err := r.Client.List(ctx, tcpRouteList); err != nil {
		log.Error(err, "Unable to list TCPRoutes")
		return r.handleReconcileErrorWithStatus(ctx, err, original, gw)
	}

	tlsRouteList := &gatewayv1alpha2.TLSRouteList{}
	if err := r.Client.List(ctx, tlsRouteList); err != nil {
		log.Error(err, "Unable to list TLSRoutes")
		return r.handleReconcileErrorWithStatus(ctx, err, original, gw)
	}

	udpRouteList := &gatewayv1alpha2.UDPRouteList{}
	if err := r.Client.List(ctx, udpRouteList); err != nil {
		log.Error(err, "Unable to list UDPRoutes")
		return r.handleReconcileErrorWithStatus(ctx, err, original, gw)
	}

	grantList := &gatewayv1beta1.ReferenceGrantList{}
	if err := r.Client.List(ctx, grantList); err != nil {
		log.Error(err, "Unable to list ReferenceGrants")
		return r.handleReconcileErrorWithStatus(ctx, err, original, gw)
	}

	backendTLSPolicyList := &gatewayv1alpha2.BackendTLSPolicyList{}
	if err := r.Client.List(ctx, backendTLSPolicyList); err != nil {
		log.Error(err, "Unable to list BackendTLSPolicies")
		return r.handleReconcileErrorWithStatus(ctx, err, original, gw)
	}

	// TODO: only list services from accepted routes.
	serviceList := &corev1.ServiceList{}
	if err := r.Client.List(ctx, serviceList); err != nil {
		log.Error(err, "Unable to list Services")
		return r.handleReconcileErrorWithStatus(ctx, err, original, gw)
	}

	// TODO: https://github.com/cilium/cilium/blob/main/operator/pkg/gateway-api/gateway_reconcile.go#L355
	meta.SetStatusCondition(&gw.Status.Conditions, metav1.Condition{
		Type:    string(gatewayv1.GatewayConditionAccepted),
		Status:  metav1.ConditionTrue,
		Reason:  string(gatewayv1.GatewayReasonAccepted),
		Message: "Gateway scheduled",
	})
	//meta.SetStatusCondition(&gw.Status.Conditions, metav1.Condition{
	//	Type:   string(gatewayv1.GatewayConditionAccepted),
	//	Status: metav1.ConditionFalse,
	//	Reason:  string(gatewayv1.GatewayReasonListenersNotValid),
	//	Message: "",
	//})

	i := &caddy.Input{
		Gateway:      original,
		GatewayClass: gwc,

		HTTPRoutes: r.filterHTTPRoutesByGateway(ctx, gw, httpRouteList.Items),
		GRPCRoutes: r.filterGRPCRoutesByGateway(ctx, gw, grpcRouteList.Items),
		TCPRoutes:  r.filterTCPRoutesByGateway(ctx, gw, tcpRouteList.Items),
		TLSRoutes:  r.filterTLSRoutesByGateway(ctx, gw, tlsRouteList.Items),
		UDPRoutes:  r.filterUDPRoutesByGateway(ctx, gw, udpRouteList.Items),

		Grants:             grantList.Items,
		BackendTLSPolicies: backendTLSPolicyList.Items,

		Services: serviceList.Items,

		Client: r.Client,
	}
	b, err := i.Config()
	if err != nil {
		log.Error(err, "Error generating Gateway config")
		return ctrl.Result{}, err
	}

	caddyEps, err := r.getEndpoints(ctx, gw)
	if err != nil {
		return ctrl.Result{}, err
	}
	if len(caddyEps.Subsets) < 1 {
		return ctrl.Result{}, errors.New("")
	}

	// Configure Caddy in parallel, so when someone runs Caddy as a DaemonSet on
	// a 5,000 node cluster, we bring the gateway controller to its knees.
	var wg sync.WaitGroup
	for _, a := range caddyEps.Subsets[0].Addresses {
		// TODO: is this necessary?
		a := a
		if a.TargetRef == nil {
			// TODO: log error
			continue
		}
		wg.Add(1)
		go func(a corev1.EndpointAddress) {
			defer wg.Done()

			target := client.ObjectKey{
				Namespace: a.TargetRef.Namespace,
				Name:      a.TargetRef.Name,
			}

			tlsConfig := r.tlsConfig.Clone()
			tlsConfig.ServerName = target.Name + "." + target.Namespace
			tr := http.DefaultTransport.(*http.Transport).Clone()
			tr.TLSClientConfig = tlsConfig
			httpClient := &http.Client{Transport: tr}

			log.V(1).Info("Programming Caddy instance", "ip", a.IP, "target", target)
			// TODO: configurable scheme  and port
			url := "https://" + net.JoinHostPort(a.IP, "2021") + "/load"
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(b))
			if err != nil {
				log.Error(err, "Error programming Caddy instance", "ip", a.IP, "target", target)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			res, err := httpClient.Do(req)
			if err != nil {
				log.Error(err, "Error programming Caddy instance", "ip", a.IP, "target", target)
				return
			}
			defer res.Body.Close()
			if res.StatusCode != http.StatusOK {
				b, _ := io.ReadAll(res.Body)
				log.Error(errors.New(string(b)), "Error programming Caddy instance", "status_code", res.StatusCode, "ip", a.IP, "target", target)
				return
			}
			_, _ = io.Copy(io.Discard, res.Body)
			log.V(1).Info("Successfully programmed Caddy instance", "ip", a.IP, "target", target)
		}(a)
	}
	wg.Wait()

	if reason, err := r.setAddressStatus(ctx, gw); err != nil {
		log.Error(err, "Address is not ready")
		meta.SetStatusCondition(&gw.Status.Conditions, metav1.Condition{
			Type:    string(gatewayv1.GatewayConditionProgrammed),
			Status:  metav1.ConditionFalse,
			Reason:  string(reason),
			Message: "Address is not ready",
		})
		return r.handleReconcileErrorWithStatus(ctx, err, original, gw)
	}
	meta.SetStatusCondition(&gw.Status.Conditions, metav1.Condition{
		Type:    string(gatewayv1.GatewayConditionProgrammed),
		Status:  metav1.ConditionTrue,
		Reason:  string(gatewayv1.GatewayReasonProgrammed),
		Message: "Gateway has been programmed",
	})
	if err := r.updateStatus(ctx, original, gw); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update Gateway status: %w", err)
	}

	log.Info("Successfully reconciled Gateway")
	return ctrl.Result{}, nil
}

func (r *GatewayReconciler) getService(ctx context.Context, gw *gatewayv1.Gateway) (*corev1.Service, error) {
	svcList := &corev1.ServiceList{}
	if err := r.Client.List(ctx, svcList, client.MatchingLabels{
		owningGatewayLabel: gw.Name,
	}); err != nil {
		return nil, err
	}
	if len(svcList.Items) == 0 {
		return nil, fmt.Errorf("no service found")
	}
	return &svcList.Items[0], nil
}

func (r *GatewayReconciler) getEndpoints(ctx context.Context, gw *gatewayv1.Gateway) (*corev1.Endpoints, error) {
	epsList := &corev1.EndpointsList{}
	if err := r.Client.List(ctx, epsList, client.MatchingLabels{
		owningGatewayLabel: gw.Name,
	}); err != nil {
		return nil, err
	}
	if len(epsList.Items) == 0 {
		return nil, fmt.Errorf("no endpoints found")
	}
	return &epsList.Items[0], nil
}

func GatewayAddressTypePtr(addr gatewayv1.AddressType) *gatewayv1.AddressType {
	return &addr
}

func (r *GatewayReconciler) setAddressStatus(ctx context.Context, gw *gatewayv1.Gateway) (gatewayv1.GatewayConditionReason, error) {
	svcList := &corev1.ServiceList{}
	if err := r.Client.List(ctx, svcList, client.MatchingLabels{
		owningGatewayLabel: gw.Name,
	}); err != nil {
		return gatewayv1.GatewayReasonNoResources, err
	}
	if len(svcList.Items) == 0 {
		return gatewayv1.GatewayReasonNoResources, fmt.Errorf("no service found")
	}
	svc := svcList.Items[0]
	if len(svc.Status.LoadBalancer.Ingress) == 0 {
		return gatewayv1.GatewayReasonAddressNotAssigned, fmt.Errorf("load balancer status is not ready")
	}

	var addresses []gatewayv1.GatewayStatusAddress
	for _, s := range svc.Status.LoadBalancer.Ingress {
		if len(s.IP) != 0 {
			addresses = append(addresses, gatewayv1.GatewayStatusAddress{
				Type:  GatewayAddressTypePtr(gatewayv1.IPAddressType),
				Value: s.IP,
			})
		}
		if len(s.Hostname) != 0 {
			addresses = append(addresses, gatewayv1.GatewayStatusAddress{
				Type:  GatewayAddressTypePtr(gatewayv1.HostnameAddressType),
				Value: s.Hostname,
			})
		}
	}
	gw.Status.Addresses = addresses
	return "", nil
}

// enqueueRequestForOwningGatewayClass returns an event handler for all Gateway objects
// belonging to the given GatewayClass.
func (r *GatewayReconciler) enqueueRequestForOwningGatewayClass() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a client.Object) []reconcile.Request {
		log := log.FromContext(ctx)

		gwList := &gatewayv1.GatewayList{}
		if err := r.Client.List(ctx, gwList); err != nil {
			log.Error(err, "Unable to list Gateways")
			return nil
		}

		reqs := make([]reconcile.Request, 0, len(gwList.Items))
		for _, gw := range gwList.Items {
			if gw.Spec.GatewayClassName != gatewayv1.ObjectName(a.GetName()) {
				continue
			}
			reqs = append(reqs, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: gw.Namespace,
					Name:      gw.Name,
				},
			})

			log.WithValues(
				"namespace", gw.GetNamespace(),
				"resource", gw.GetName(),
			).Info("Queueing gateway")
		}
		return reqs
	})
}

// enqueueRequestForOwningResource returns an event handler for all Gateway objects having
// owningGatewayLabel
func (r *GatewayReconciler) enqueueRequestForOwningResource() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a client.Object) []reconcile.Request {
		log := log.FromContext(ctx).WithValues(
			"controller", "gateway",
			"resource", a.GetName(),
		)

		key, found := a.GetLabels()[owningGatewayLabel]
		if !found {
			return nil
		}

		log.WithValues(
			"namespace", a.GetNamespace(),
			"gateway", key,
		).Info("Enqueued gateway for owning service")

		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: a.GetNamespace(),
					Name:      key,
				},
			},
		}
	})
}

// enqueueRequestForOwningHTTPRoute returns an event handler for any changes with HTTP Routes
// belonging to the given Gateway
func (r *GatewayReconciler) enqueueRequestForOwningHTTPRoute() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
		route, ok := o.(*gatewayv1.HTTPRoute)
		if !ok {
			return nil
		}
		return getReconcileRequestsForRoute(ctx, r.Client, o, route.Spec.CommonRouteSpec)
	})
}

// enqueueRequestForOwningGRPCRoute returns an event handler for any changes with GRPC Routes
// belonging to the given Gateway
func (r *GatewayReconciler) enqueueRequestForOwningGRPCRoute() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
		route, ok := o.(*gatewayv1alpha2.GRPCRoute)
		if !ok {
			return nil
		}
		return getReconcileRequestsForRoute(ctx, r.Client, o, route.Spec.CommonRouteSpec)
	})
}

// enqueueRequestForOwningTCPRoute returns an event handler for any changes with TCP Routes
// belonging to the given Gateway
func (r *GatewayReconciler) enqueueRequestForOwningTCPRoute() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
		route, ok := o.(*gatewayv1alpha2.TCPRoute)
		if !ok {
			return nil
		}
		return getReconcileRequestsForRoute(ctx, r.Client, o, route.Spec.CommonRouteSpec)
	})
}

// enqueueRequestForOwningTLSRoute returns an event handler for any changes with TLS Routes
// belonging to the given Gateway
func (r *GatewayReconciler) enqueueRequestForOwningTLSRoute() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
		route, ok := o.(*gatewayv1alpha2.TLSRoute)
		if !ok {
			return nil
		}
		return getReconcileRequestsForRoute(ctx, r.Client, o, route.Spec.CommonRouteSpec)
	})
}

// enqueueRequestForOwningUDPRoute returns an event handler for any changes with UDP Routes
// belonging to the given Gateway
func (r *GatewayReconciler) enqueueRequestForOwningUDPRoute() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
		route, ok := o.(*gatewayv1alpha2.UDPRoute)
		if !ok {
			return nil
		}
		return getReconcileRequestsForRoute(ctx, r.Client, o, route.Spec.CommonRouteSpec)
	})
}

func getReconcileRequestsForRoute(ctx context.Context, c client.Client, object metav1.Object, route gatewayv1.CommonRouteSpec) []reconcile.Request {
	log := log.FromContext(ctx, "resource", types.NamespacedName{
		Namespace: object.GetNamespace(),
		Name:      object.GetName(),
	})

	reqs := make([]reconcile.Request, 0, len(route.ParentRefs))
	for _, parent := range route.ParentRefs {
		if !gateway.IsGateway(parent) {
			continue
		}

		ns := gateway.NamespaceDerefOr(parent.Namespace, object.GetNamespace())
		gw := &gatewayv1.Gateway{}
		if err := c.Get(ctx, types.NamespacedName{
			Namespace: ns,
			Name:      string(parent.Name),
		}, gw); err != nil {
			if !apierrors.IsNotFound(err) {
				log.Error(err, "Failed to get Gateway")
			}
			continue
		}

		if !hasMatchingController(ctx, c)(gw) {
			log.V(3).Info("Gateway does not have a matching controller, skipping")
			continue
		}

		log.Info(
			"Enqueued Gateway for Route",
			"namespace", ns,
			"resource", parent.Namespace,
			"route", object.GetName(),
		)

		reqs = append(reqs, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: ns,
				Name:      string(parent.Name),
			},
		})
	}
	return reqs
}

// enqueueRequestForTLSPolicy .
// TODO: document
func (r *GatewayReconciler) enqueueRequestForTLSPolicy() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
		_, ok := o.(*gatewayv1alpha2.BackendTLSPolicy)
		if !ok {
			return nil
		}
		// TODO: implement the rest of the logic
		return nil
	})
}

// enqueueRequestForOwningTLSCertificate returns an event handler for any changes with TLS secrets
func (r *GatewayReconciler) enqueueRequestForTLSSecret() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
		gateways := getGatewaysForSecret(ctx, r.Client, o)
		reqs := make([]reconcile.Request, len(gateways))
		for i, gw := range gateways {
			reqs[i] = reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: gw.GetNamespace(),
					Name:      gw.GetName(),
				},
			}
		}
		return reqs
	})
}

// enqueueRequestForAllowedNamespace returns an event handler for any changes
// with allowed namespaces
func (r *GatewayReconciler) enqueueRequestForAllowedNamespace() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, ns client.Object) []reconcile.Request {
		gateways := getGatewaysForNamespace(ctx, r.Client, ns)
		reqs := make([]reconcile.Request, len(gateways))
		for i, gw := range gateways {
			reqs[i] = reconcile.Request{
				NamespacedName: gw,
			}
		}
		return reqs
	})
}

// updateStatus .
// TODO: document
func (r *GatewayReconciler) updateStatus(ctx context.Context, original, new *gatewayv1.Gateway) error {
	oldStatus := original.Status.DeepCopy()
	newStatus := new.Status.DeepCopy()
	if cmp.Equal(oldStatus, newStatus, cmpopts.IgnoreFields(metav1.Condition{}, "LastTransitionTime")) {
		return nil
	}
	return r.Client.Status().Update(ctx, new)
}

// handleReconcileErrorWithStatus .
// TODO
func (r *GatewayReconciler) handleReconcileErrorWithStatus(ctx context.Context, reconcileErr error, original, modified *gatewayv1.Gateway) (ctrl.Result, error) {
	if err := r.updateStatus(ctx, original, modified); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update Gateway status while handling the reconcile error %w: %w", reconcileErr, err)
	}
	return ctrl.Result{}, reconcileErr
}

// filterHTTPRoutesByGateway .
// TODO
func (r *GatewayReconciler) filterHTTPRoutesByGateway(ctx context.Context, gw *gatewayv1.Gateway, routes []gatewayv1.HTTPRoute) []gatewayv1.HTTPRoute {
	_log := log.FromContext(
		ctx,
		"gateway", types.NamespacedName{
			Namespace: gw.Namespace,
			Name:      gw.Name,
		},
	)
	var filtered []gatewayv1.HTTPRoute
	for _, route := range routes {
		log2 := _log.WithValues("route", types.NamespacedName{
			Namespace: route.Namespace,
			Name:      route.Name,
		})

		ctx2 := log.IntoContext(ctx, log2)

		if !isAttachable(ctx2, gw, &route, route.Status.Parents) {
			log2.Info("route is not attachable")
			continue
		}

		if !isAllowed(ctx2, r.Client, gw, &route) {
			log2.Info("route is not allowed")
			continue
		}

		//if len(computeHosts(gw, route.Spec.Hostnames)) > 1 {
		//	log2.Info("couldn't compute hosts")
		//	continue
		//}

		filtered = append(filtered, route)
	}
	return filtered
}

// filterGRPCRoutesByGateway .
// TODO
func (r *GatewayReconciler) filterGRPCRoutesByGateway(ctx context.Context, gw *gatewayv1.Gateway, routes []gatewayv1alpha2.GRPCRoute) []gatewayv1alpha2.GRPCRoute {
	var filtered []gatewayv1alpha2.GRPCRoute
	for _, route := range routes {
		if isAttachable(ctx, gw, &route, route.Status.Parents) && isAllowed(ctx, r.Client, gw, &route) && len(computeHosts(gw, route.Spec.Hostnames)) > 0 {
			filtered = append(filtered, route)
		}
	}
	return filtered
}

// filterTCPRoutesByGateway .
// TODO
func (r *GatewayReconciler) filterTCPRoutesByGateway(ctx context.Context, gw *gatewayv1.Gateway, routes []gatewayv1alpha2.TCPRoute) []gatewayv1alpha2.TCPRoute {
	var filtered []gatewayv1alpha2.TCPRoute
	for _, route := range routes {
		if isAttachable(ctx, gw, &route, route.Status.Parents) && isAllowed(ctx, r.Client, gw, &route) {
			filtered = append(filtered, route)
		}
	}
	return filtered
}

// filterTLSRoutesByGateway .
// TODO
func (r *GatewayReconciler) filterTLSRoutesByGateway(ctx context.Context, gw *gatewayv1.Gateway, routes []gatewayv1alpha2.TLSRoute) []gatewayv1alpha2.TLSRoute {
	var filtered []gatewayv1alpha2.TLSRoute
	for _, route := range routes {
		if isAttachable(ctx, gw, &route, route.Status.Parents) && isAllowed(ctx, r.Client, gw, &route) && len(computeHosts(gw, route.Spec.Hostnames)) > 0 {
			filtered = append(filtered, route)
		}
	}
	return filtered
}

// filterUDPRoutesByGateway .
// TODO
func (r *GatewayReconciler) filterUDPRoutesByGateway(ctx context.Context, gw *gatewayv1.Gateway, routes []gatewayv1alpha2.UDPRoute) []gatewayv1alpha2.UDPRoute {
	var filtered []gatewayv1alpha2.UDPRoute
	for _, route := range routes {
		if isAttachable(ctx, gw, &route, route.Status.Parents) && isAllowed(ctx, r.Client, gw, &route) {
			filtered = append(filtered, route)
		}
	}
	return filtered
}

func (r *GatewayReconciler) usedInGateway(obj client.Object) bool {
	return len(getGatewaysForSecret(context.Background(), r.Client, obj)) > 0
}
