package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/golang/glog"
	controllers "github.com/mjudeikis/ocp-controller/pkg/controllers"
	"github.com/mjudeikis/ocp-controller/pkg/signals"
	tagcontrollerclientset "github.com/mjudeikis/ocp-controller/pkg/tagcontroller/clientset/versioned"
	taginformers "github.com/mjudeikis/ocp-controller/pkg/tagcontroller/informers/externalversions"
	apps "github.com/openshift/client-go/apps/clientset/versioned"
	appsinformers "github.com/openshift/client-go/apps/informers/externalversions"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Set logging output to standard console out
	log.SetOutput(os.Stdout)

	stopCh := signals.SetupSignalHandler()

	// Create clientset for interacting with the kubernetes cluster
	originClient, kubeClient, tagClient, err := newClientSet()

	if err != nil {
		glog.Fatalf("Error running controller: %s", err.Error())
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

	kubeconfig := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	var restconfig *rest.Config
	var err error
	flag.Parse()

	if len(*kubeconfig) > 0 {
		restconfig, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		restconfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, nil, err
		}
	}

	tagClient, err := tagcontrollerclientset.NewForConfig(restconfig)
	if err != nil {
		glog.Fatalf("Error building tag clientset: %s", err.Error())
	}

	originClient, err := apps.NewForConfig(restconfig)
	if err != nil {
		return nil, nil, nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(restconfig)
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
