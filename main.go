/*
Copyright 2021 Red Hat Community Of Practice.

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
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/apiserver-runtime/pkg/builder"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	redhatcopv1alpha1 "github.com/redhat-cop/vault-apiserver/api/v1alpha1"
	vaultv1alpha1 "github.com/redhat-cop/vault-apiserver/api/v1alpha1"
	"github.com/redhat-cop/vault-apiserver/vaultstorage"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(redhatcopv1alpha1.AddToScheme(scheme))
	utilruntime.Must(vaultv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

// +kubebuilder:rbac:groups="",resources=configmaps;namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations;mutatingwebhookconfigurations,verbs=get;list;watch
// +kubebuilder:rbac:groups=flowcontrol.apiserver.k8s.io,resources=prioritylevelconfigurations;flowschemas,verbs=get;list;watch
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create

func main() {
	// var metricsAddr string
	// var enableLeaderElection bool
	// var probeAddr string
	// flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	// flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	// flag.BoolVar(&enableLeaderElection, "leader-elect", false,
	// 	"Enable leader election for controller manager. "+
	// 		"Enabling this will ensure there is only one active controller manager.")

	ctrl.SetLogger(zap.New())

	// mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
	// 	Scheme:                 scheme,
	// 	MetricsBindAddress:     metricsAddr,
	// 	Port:                   9443,
	// 	HealthProbeBindAddress: probeAddr,
	// 	LeaderElection:         enableLeaderElection,
	// 	LeaderElectionID:       "e04e27b0.redhat.io",
	// })
	// if err != nil {
	// 	setupLog.Error(err, "unable to start manager")
	// 	os.Exit(1)
	// }

	//+kubebuilder:scaffold:builder

	baselog := ctrl.Log.WithName("apiserver")

	cmd, err := builder.APIServer.
		WithResourceAndHandler(&redhatcopv1alpha1.SecretEngine{}, vaultstorage.NewVaultMountStorageProvider(baselog)).
		WithResourceAndHandler(&redhatcopv1alpha1.PolicyBinding{}, vaultstorage.NewVaultRoleResourceProvider(baselog)).
		WithResourceAndHandler(&redhatcopv1alpha1.Policy{}, vaultstorage.NewVaultPolicyResourceProvider(baselog)).
		WithLocalDebugExtension().
		WithoutEtcd().
		//WithOpenAPIDefinitions("vault.redhatcop.redhat.io", "v1alpha1", redhatcopv1alpha1.GetOpenAPIDefinitions).
		Build()

	if err != nil {
		setupLog.Error(err, "unable to set up apiserver")
		os.Exit(1)
	}

	cmd.Flags().AddFlag(&pflag.Flag{
		Name:     vaultstorage.KubeAuthPathFlagName,
		Usage:    "this is the path where the kubernetes authentication method was mounted",
		DefValue: "auth/kubernetes",
	})
	viper.BindPFlag(vaultstorage.KubeAuthPathFlagName, cmd.PersistentFlags().Lookup(vaultstorage.KubeAuthPathFlagName))
	err = cmd.Execute()
	if err != nil {
		setupLog.Error(err, "unable to start apiserver")
		os.Exit(1)
	}
	// if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
	// 	setupLog.Error(err, "unable to set up health check")
	// 	os.Exit(1)
	// }
	// if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
	// 	setupLog.Error(err, "unable to set up ready check")
	// 	os.Exit(1)
	// }

	// setupLog.Info("starting manager")
	// if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
	// 	setupLog.Error(err, "problem running manager")
	// 	os.Exit(1)
	// }
}
