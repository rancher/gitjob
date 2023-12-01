//go:generate bash ./scripts/controller-gen-generate.sh

package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rancher/gitjob/pkg/webhook"

	gitjobv1 "github.com/rancher/gitjob/pkg/apis/gitjob.cattle.io/v1"
	"github.com/rancher/gitjob/pkg/controller"
	"github.com/rancher/gitjob/pkg/git/poll"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(gitjobv1.AddToScheme(scheme))
}

func main() {
	ctx := context.Background()
	namespace := os.Getenv("NAMESPACE")
	var metricsAddr string
	var enableLeaderElection bool
	var image string
	var listen string
	var debug bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8081", "The address the metric endpoint binds to.")
	flag.StringVar(&image, "gitjob-image", "rancher/gitjob:dev", "The gitjob image that will be used in the generated job.")
	flag.StringVar(&listen, "listen", ":8080", "The port the webhook listens.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&debug, "debug", false, "debug mode.")
	flag.Parse()

	opts := zap.Options{
		Development: debug,
	}
	opts.BindFlags(flag.CommandLine)
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        "fleet-gitjob-leader",
		LeaderElectionNamespace: namespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	go startWebhook(ctx, namespace, listen, mgr.GetClient(), mgr.GetCache())

	setupLog.Info("starting manager")
	if err = (&controller.GitJobReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Image:     image,
		GitPoller: poll.NewHandler(mgr.GetClient()),
		Log:       ctrl.Log.WithName("gitjob-reconciler"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GitJob")
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

}

func startWebhook(ctx context.Context, namespace string, addr string, client client.Client, cacheClient cache.Cache) {
	setupLog.Info("Setting up webhook listener")
	handler, err := webhook.HandleHooks(ctx, namespace, client, cacheClient)
	if err != nil {
		setupLog.Error(err, "webhook handler can't be created")
		os.Exit(1)
	}
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
		// According to https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		setupLog.Error(err, "failed to listen ", "address", addr)
		os.Exit(1)
	}
}
