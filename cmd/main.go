/*
Copyright 2024 cuisongliu.

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

package main

import (
	"flag"
	v1 "k8s.io/api/core/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	infrav1beta1 "github.com/cuisongliu/automq-operator/api/v1beta1"
	"github.com/cuisongliu/automq-operator/internal/controller"
	"github.com/gin-gonic/gin"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	//+kubebuilder:scaffold:imports

	utilcontroller "github.com/labring/operator-sdk/controller"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(infrav1beta1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
	utilruntime.Must(promv1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var rateLimiterOptions utilcontroller.RateLimiterOptions
	var mountTZ bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&mountTZ, "mount-tz", false, "Mount the /etc/localtime file from the host to the container.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	rateLimiterOptions.BindFlags(flag.CommandLine)
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		Metrics:                 metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress:  probeAddr,
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        "97e298dd.cuisongliu.github.com",
		LeaderElectionNamespace: os.Getenv("NAMESPACE_NAME"),
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
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controller.AutoMQReconciler{
		Finalizer: "apps.cuisongliu.com/automq.finalizer",
		MountTZ:   mountTZ,
	}).SetupWithManager(mgr, rateLimiterOptions); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AutoMQ")
		os.Exit(1)
	}
	if ew, _ := os.LookupEnv("ENABLE_WEBHOOKS"); ew != "false" {
		if err = (&infrav1beta1.AutoMQ{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "AutoMQ")
			os.Exit(1)
		}
	}

	if os.Getenv("OPERATOR_APIS_SVC_NAME") == "" {
		setupLog.Error(err, "OPERATOR_APIS_SVC_NAME is empty")
		os.Exit(1)
	}

	if os.Getenv("NAMESPACE_NAME") == "" {
		_ = os.Setenv("NAMESPACE_NAME", "default")
	}

	//+kubebuilder:scaffold:builder

	ctx := ctrl.SetupSignalHandler()

	go func() {
		if mgr.GetCache().WaitForCacheSync(ctx) {
			setupLog.Info("cache sync success")
			router := gin.Default()
			router.GET("/api/v1/nodes/:name", func(c *gin.Context) {
				name := c.Param("name")
				node := &v1.Node{}
				node.Name = name
				if noe := mgr.GetClient().Get(ctx, client.ObjectKeyFromObject(node), node); noe != nil {
					c.JSON(500, gin.H{"message": noe.Error()})
					return
				}
				nodeIP := ""
				for _, addr := range node.Status.Addresses {
					if addr.Type == v1.NodeInternalIP {
						nodeIP = addr.Address
						break
					}
				}
				if nodeIP == "" {
					c.JSON(500, gin.H{"message": "node ip not found"})
					return
				}
				c.String(200, nodeIP)
			})
			router.Run(":9090")
		}
	}()

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
