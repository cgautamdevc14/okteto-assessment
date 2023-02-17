package main

import (
	"context"
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

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
