// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"log/slog"
	"maps"
	"os"
	"slices"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	//+kubebuilder:scaffold:imports

	"github.com/caddyserver/gateway/internal/controller"
	"github.com/go-logr/logr"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(gatewayv1.Install(scheme))
	utilruntime.Must(gatewayv1alpha2.Install(scheme))
	utilruntime.Must(gatewayv1alpha3.Install(scheme))
	utilruntime.Must(gatewayv1beta1.Install(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	var enableLeaderElection bool
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	var probeAddr string
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	var secureMetrics bool
	flag.BoolVar(&secureMetrics, "metrics-secure", false, "If set the metrics endpoint is served securely")
	var enableHTTP2 bool
	flag.BoolVar(&enableHTTP2, "enable-http2", false, "If set, HTTP/2 will be enabled for the metrics and webhook servers")
	var logLevel slog.Level
	flag.TextVar(&logLevel, "log-level", slog.LevelInfo, "Set the log level (DEBUG, INFO, WARN, ERROR)")

	flag.Parse()

	// Use slog as the application's logger.
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	})
	slog.SetDefault(slog.New(h))
	ctrl.SetLogger(logr.FromSlogHandler(h))

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	tlsOpts := []func(*tls.Config){}
	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress:   metricsAddr,
			SecureServing: secureMetrics,
			TLSOpts:       tlsOpts,
		},
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "657d83d7.caddyserver.com",

		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
		return
	}

	ctx := ctrl.SetupSignalHandler()
	cs, err := clientset.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to create clientset")
		os.Exit(1)
		return
	}

	gi, err := checkCRDs(ctx, cs, setupLog)
	if err != nil {
		setupLog.Error(err, "")
		os.Exit(1)
		return
	}

	client := mgr.GetClient()
	scheme := mgr.GetScheme()
	recorder := mgr.GetEventRecorderFor("caddy-gateway")

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
		return
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
		return
	}

	if err = (&controller.GatewayReconciler{
		Client:   client,
		Scheme:   scheme,
		Recorder: recorder,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Gateway")
		os.Exit(1)
		return
	}

	if err = (&controller.GatewayClassReconciler{
		Client:   client,
		Scheme:   scheme,
		Recorder: recorder,
		Info:     gi,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GatewayClass")
		os.Exit(1)
		return
	}

	//if err = (&controller.GRPCRouteReconciler{
	//	Client:   client,
	//	Scheme:   scheme,
	//	Recorder: recorder,
	//}).SetupWithManager(mgr); err != nil {
	//	setupLog.Error(err, "unable to create controller", "controller", "GRPCRoute")
	//	os.Exit(1)
	//	return
	//}

	if err = (&controller.HTTPRouteReconciler{
		Client:   client,
		Scheme:   scheme,
		Recorder: recorder,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HTTPRoute")
		os.Exit(1)
		return
	}

	if slices.Contains(gi.Resources, tcpRouteGVK) {
		if err = (&controller.TCPRouteReconciler{
			Client:   client,
			Scheme:   scheme,
			Recorder: recorder,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "TCPRoute")
			os.Exit(1)
			return
		}
	}

	if slices.Contains(gi.Resources, tlsRouteGVK) {
		if err = (&controller.TLSRouteReconciler{
			Client:   client,
			Scheme:   scheme,
			Recorder: recorder,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "TLSRoute")
			os.Exit(1)
			return
		}
	}

	if slices.Contains(gi.Resources, udpRouteGVK) {
		if err = (&controller.UDPRouteReconciler{
			Client:   client,
			Scheme:   scheme,
			Recorder: recorder,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "UDPRoute")
			os.Exit(1)
			return
		}
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

var (
	backendTLSPolicyGVK = gatewayv1alpha3.SchemeGroupVersion.WithKind("BackendTLSPolicy")
	tcpRouteGVK         = gatewayv1alpha2.SchemeGroupVersion.WithKind("TCPRoute")
	tlsRouteGVK         = gatewayv1alpha2.SchemeGroupVersion.WithKind("TLSRoute")
	udpRouteGVK         = gatewayv1alpha2.SchemeGroupVersion.WithKind("UDPRoute")

	requiredGVKs = []schema.GroupVersionKind{
		gatewayv1.SchemeGroupVersion.WithKind("GatewayClass"),
		gatewayv1.SchemeGroupVersion.WithKind("Gateway"),
		gatewayv1.SchemeGroupVersion.WithKind("HTTPRoute"),
		// gatewayv1.SchemeGroupVersion.WithKind("GRPCRoute"),
		gatewayv1beta1.SchemeGroupVersion.WithKind("ReferenceGrant"),
	}

	optionalGVKs = []schema.GroupVersionKind{
		backendTLSPolicyGVK,
		tcpRouteGVK,
		tlsRouteGVK,
		udpRouteGVK,
	}
)

// Add RBAC permissions to get CRDs, so we can verify that the gateway-api CRDs
// are not just installed but also a supported version.
//
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch

type MissingCRDError struct {
	schema.GroupVersionKind
}

func (e *MissingCRDError) Error() string {
	return "missing crd: " + e.Group + "/" + e.Version + " " + e.Kind
}

func checkCRDs(ctx context.Context, cs *clientset.Clientset, log logr.Logger) (*controller.GatewayAPIInfo, error) {
	crdList, err := cs.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error(err, "Failed to list CustomResourceDefinitions")
		return nil, err
	}

	filteredCRDs := make([]apiextensionsv1.CustomResourceDefinition, 0, len(crdList.Items))
	for _, crd := range crdList.Items {
		if crd.Spec.Group != "gateway.networking.k8s.io" && crd.Spec.Group != "gateway.networking.x-k8s.io" {
			continue
		}
		filteredCRDs = append(filteredCRDs, crd)
	}
	filteredCRDs = slices.Clip(filteredCRDs)

	presentGVKs := make([]schema.GroupVersionKind, len(filteredCRDs))
	gatewayVersions := make(map[string][]string)
	gatewayChannels := make(map[string][]string)
	for i, crd := range filteredCRDs {
		ver, ok := crd.Annotations["gateway.networking.k8s.io/bundle-version"]
		if !ok {
			// TODO: what?
		}
		channel, ok := crd.Annotations["gateway.networking.k8s.io/channel"]
		if !ok {
			// TODO: what?
		}
		log.Info("Found CustomResourceDefinitions", "CRD.Group", crd.Spec.Group, "CRD.Kind", crd.Spec.Names.Kind, "BundleVersion", ver, "Channel", channel)

		if _, ok := gatewayVersions[ver]; !ok {
			gatewayVersions[ver] = []string{crd.Spec.Names.Kind}
		} else {
			gatewayVersions[ver] = append(gatewayVersions[ver], crd.Spec.Names.Kind)
		}

		if _, ok := gatewayChannels[channel]; !ok {
			gatewayChannels[channel] = []string{crd.Spec.Names.Kind}
		} else {
			gatewayChannels[channel] = append(gatewayChannels[channel], crd.Spec.Names.Kind)
		}

		presentGVKs[i] = schema.GroupVersionKind{
			Group: crd.Spec.Group,
			// TODO: we should probably use a type with a list of versions
			Version: crd.Spec.Versions[0].Name,
			Kind:    crd.Spec.Names.Kind,
		}
	}
	var version string
	for key := range maps.Keys(gatewayVersions) {
		if version != "" {
			log.Error(nil, "Multiple Gateway API versions are installed on the cluster. This is prohibited, please re-install the Gateway API CRDs", "Versions", slices.Collect(maps.Keys(gatewayVersions)))
		}
		version = key
	}
	var channel string
	for key := range maps.Keys(gatewayChannels) {
		if channel != "" {
			log.Error(nil, "Multiple Gateway API channels are installed on the cluster. This is prohibited, please re-install the Gateway API CRDs", "Channels", slices.Collect(maps.Keys(gatewayChannels)))
		}
		channel = key
	}

	var errs error
	for _, gvk := range requiredGVKs {
		if slices.Contains(presentGVKs, gvk) {
			log.Info("Required CRD found", "CRD.Group", gvk.Group, "CRD.Version", gvk.Version, "CRD.Kind", gvk.Kind)
			continue
		}
		log.Error(nil, "Required CRD is missing", "CRD.Group", gvk.Group, "CRD.Version", gvk.Version, "CRD.Kind", gvk.Kind)
		errs = errors.Join(errs, &MissingCRDError{GroupVersionKind: gvk})
	}
	for _, gvk := range optionalGVKs {
		if slices.Contains(presentGVKs, gvk) {
			log.Info("Optional CRD found", "CRD.Group", gvk.Group, "CRD.Version", gvk.Version, "CRD.Kind", gvk.Kind)
			continue
		}
		log.Info("Optional CRD is missing", "CRD.Group", gvk.Group, "CRD.Version", gvk.Version, "CRD.Kind", gvk.Kind)
	}
	if errs != nil {
		return nil, errs
	}

	log.Info("Found Gateway API CRDs", "BundleVersion", version, "Channel", channel)
	return &controller.GatewayAPIInfo{
		BundleVersion: version,
		Channel:       channel,
		Resources:     presentGVKs,
	}, errs
}
