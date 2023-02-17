package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type KubePod struct {
	name     string
	restarts int32
	age      time.Duration
}

func NewKubePod(name string, restarts int32, age time.Duration) *KubePod {
	return &KubePod{
		name:     name,
		restarts: restarts,
		age:      age,
	}
}
func main() {
	fmt.Println("Starting hello-world server...")
	http.HandleFunc("/", helloServer)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func helloServer(w http.ResponseWriter, r *http.Request) {

	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Fprint(w, "E1")
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprint(w, "E2")
		panic(err.Error())
	}

	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprint(w, "E3")
		panic(err.Error())
	}

	fmt.Fprint(w, "Reached here now...")
	fmt.Fprint(w, "There are %d pods in the cluster\n", len(pods.Items))
}

func getPods(cs *kubernetes.Clientset) *v1.PodList {
	pods, err := cs.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		panic(err.Error())
	}

	return pods
}

func getInClusterConfig() *rest.Config {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	return config
}

func getKubeClientset(cfg *rest.Config) *kubernetes.Clientset {
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err.Error())
	}

	return clientset
}

func createKubePods(podList *v1.PodList) []KubePod {
	kubePods := make([]KubePod, 0, len(podList.Items))

	for _, pod := range podList.Items {
		podCreationTime := pod.GetCreationTimestamp()
		podStatus := pod.Status
		var restarts int32

		name := pod.GetName()
		age := time.Since(podCreationTime.Time).Round(time.Second)

		for container := range pod.Spec.Containers {
			restarts += podStatus.ContainerStatuses[container].RestartCount
		}

		kube := NewKubePod(name, restarts, age)
		kubePods = append(kubePods, *kube)

	}

	return kubePods
}
