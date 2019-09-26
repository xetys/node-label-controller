package main

/*
This is a very straight forward implementation of a tiny controller. By using Watch on nodes, we get a very simple
watch cycle, which lacks caching, rate-limiting and other advanced features. As node updates shouldn't occur as frequent
like other resources, like pods, I have chosen this simple approach
*/

import (
	"flag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	watch2 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Controller struct {
	clientset *kubernetes.Clientset
	watch     watch2.Interface
}

// NewController creates a new instance of this controller
func NewController(clientset *kubernetes.Clientset) *Controller {
	return &Controller{clientset: clientset}
}

// Run starts the watch cycle
func (c *Controller) Run() error {
	var err error

	klog.Info("Starting node label controller, watching node events...")
	c.watch, err = c.clientset.CoreV1().Nodes().Watch(metav1.ListOptions{})
	if err != nil {
		return err
	}

	go func() {
		for {
			// wait for events
			event := <-c.watch.ResultChan()
			if event.Object != nil {
				node := event.Object.(*v1.Node)
				// watch only ADDED events, as MODIFIED occurs quite often without relevance to this operator
				if event.Type == watch2.Added {
					err := c.handleAddedNode(node)
					if err != nil {
						klog.Fatal(err)
					}
				}
			}
		}
	}()

	return nil
}

// SetupCloseHandler installs a signal handler for a clean exit
func (c *Controller) SetupCloseHandler(stop chan struct{}) chan struct{} {
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		// cleanly closing the watch before exit
		c.watch.Stop()
		stop <- struct{}{}
	}()
	return stop
}

// handleAddedNode takes added nodes and labels it if it doesn't have the desired label but is a Container Linux node
func (c *Controller) handleAddedNode(node *v1.Node) error {
	klog.Infof("handling node %s", node.Name)

	// check if the node already has the "kubermatic.io/uses-container-linux" label on it
	for labelName := range node.Labels {

		if labelName == "kubermatic.io/uses-container-linux" {
			// this node is already marked, it shouldn't change it's OS
			klog.Infof("node %s is already labeled with kubermatic.io/uses-container-linux", node.Name)
			return nil
		}
	}

	operatingSystem := node.Status.NodeInfo.OSImage
	if strings.Contains(operatingSystem, "Container Linux") {
		klog.Infof("node %s is running %s, labeling...", node.Name, operatingSystem)

		// add new label to the node
		node.Labels["kubermatic.io/uses-container-linux"] = "true"
		_, err := c.clientset.CoreV1().Nodes().Update(node)
		if err != nil {
			return err
		}
	} else {
		klog.Infof("node %s is running %s", node.Name, operatingSystem)
	}

	return nil
}

// homeDir retrieves the users home dir across different OS'
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// K8SConfig tries to create an in-cluster config first, and falls back to flag-oriented and default kubeconfig creation
func K8SConfig() (*rest.Config, error) {

	var kubeconfig *string

	config, err := rest.InClusterConfig()

	if err != nil {
		klog.Info("in cluster config failed, trying from local")
		if home := homeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()

		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func main() {

	// creates the connection
	config, err := K8SConfig()
	if err != nil {
		klog.Fatal(err)
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	// creates the controller and starts it
	controller := NewController(clientset)

	err = controller.Run()
	if err != nil {
		klog.Fatal(err)
	}

	// installing close handler
	stop := make(chan struct{})
	defer close(stop)
	stop = controller.SetupCloseHandler(stop)
	<-stop
	klog.Info("Stopping controller")
}
