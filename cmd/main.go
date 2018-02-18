package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	controllers "github.com/mjudeikis/ocp-controller/pkg/controllers"
	"github.com/mjudeikis/ocp-controller/pkg/signals"
	tagcontrollerclientset "github.com/mjudeikis/ocp-controller/pkg/tagcontroller/clientset/versioned"
	taginformers "github.com/mjudeikis/ocp-controller/pkg/tagcontroller/informers/externalversions"
	apps "github.com/openshift/client-go/apps/clientset/versioned"
	appsinformers "github.com/openshift/client-go/apps/informers/externalversions"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Set logging output to standard console out
	log.SetOutput(os.Stdout)

	stopCh := signals.SetupSignalHandler()

	// Create clientset for interacting with the kubernetes cluster
	originClient, kubeClient, tagClient, err := newClientSet()

	if err != nil {
		panic(err.Error())
	}

	appsInformerFactory := appsinformers.NewSharedInformerFactory(originClient, time.Second*30)
	tagInformerFactory := taginformers.NewSharedInformerFactory(tagClient, time.Second*30)

	controller := controllers.NewController(originClient, kubeClient, tagClient, appsInformerFactory, tagInformerFactory)

	go appsInformerFactory.Start(stopCh)
	go tagInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		glog.Fatalf("Error running controller: %s", err.Error())
	}
}

func newClientSet() (*apps.Clientset, *kubernetes.Clientset, *tagcontrollerclientset.Clientset, error) {

	// use the current context in kubeconfig
	//config, err := templateclientsetclientcmd.BuildConfigFromFlags("", kubeConfigLocation)

	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, nil, nil, err
	}

	tagClient, err := tagcontrollerclientset.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Error building tag clientset: %s", err.Error())
	}

	originClient, err := apps.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, err
	}

	return originClient, kubeClient, tagClient, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
