package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/mjudeikis/ocp-controller/pkg/controller"
	samplecontrollerclientset "github.com/mjudeikis/ocp-controller/pkg/samplecontroller/clientset/versioned"
	informers "github.com/mjudeikis/ocp-controller/pkg/samplecontroller/informers/externalversions"
	imagev1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Set logging output to standard console out
	log.SetOutput(os.Stdout)

	sigs := make(chan os.Signal, 1) // Create channel to receive OS signals
	stop := make(chan struct{})     // Create channel to receive stop signal

	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT) // Register the sigs channel to receieve SIGTERM

	wg := &sync.WaitGroup{} // Goroutines can add themselves to this to be waited on so that they finish

	// Create clientset for interacting with the kubernetes cluster
	kclient, imageClient, crdClient, err := newClientSet()

	if err != nil {
		panic(err.Error())
	}

	go controller.NewDeploymentController(kclient, imageClient).Run(stop, wg)

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kclient, time.Second*30)
	exampleInformerFactory := informers.NewSharedInformerFactory(crdClient, time.Second*30)

	controller := controller.NewController(kclient, crdClient, kubeInformerFactory, exampleInformerFactory)

	go kubeInformerFactory.Start(stop)
	go exampleInformerFactory.Start(stop)

	if err = controller.Run(2, stop); err != nil {
		glog.Fatalf("Error running controller: %s", err.Error())
	}

	<-sigs // Wait for signals (this hangs until a signal arrives)
	log.Printf("Shutting down...")

	close(stop) // Tell goroutines to stop themselves
	wg.Wait()   // Wait for all to be stopped
}

func newClientSet() (*kubernetes.Clientset, *imagev1.ImageV1Client, *samplecontrollerclientset.Clientset, error) {

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

	imageClient, err := imagev1.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, err
	}

	samplecontrollerClient, err := samplecontrollerclientset.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Error building example clientset: %s", err.Error())
	}

	kClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, err
	}

	return kClient, imageClient, samplecontrollerClient, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
