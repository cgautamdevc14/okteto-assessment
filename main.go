package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// import (
// 	"context"
// 	"fmt"
// 	"net/http"
// 	"time"

// 	v1 "k8s.io/api/core/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// 	"k8s.io/client-go/kubernetes"
// 	"k8s.io/client-go/rest"
// )

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

// func helloServer(w http.ResponseWriter, r *http.Request) {
// 	fmt.Fprint(w, "Hello, world!")
// 	// config, err := rest.InClusterConfig()
// 	// if err != nil {
// 	// 	fmt.Fprint(w, "E1")
// 	// 	panic(err.Error())
// 	// }

// 	// clientset, err := kubernetes.NewForConfig(config)
// 	// if err != nil {
// 	// 	fmt.Fprint(w, "E2")
// 	// 	panic(err.Error())
// 	// }

// 	// pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
// 	// if err != nil {
// 	// 	fmt.Fprint(w, "E3")
// 	// 	panic(err.Error())
// 	// }

// 	// fmt.Fprint(w, "Reached here now...")
// 	// fmt.Fprint(w, "There are %d pods in the cluster\n", len(pods.Items))

// }
func podsCounter(w http.ResponseWriter, r *http.Request) {
	config := getInClusterConfig()
	fmt.Fprint(w, "The number of pods running in your current namespace: "+string(len(getPods(getKubeClientset(config)).Items))+"	")
	// clientset := getKubeClientset(config)
	// pods := getPods(clientset)
	// kubePods := createKubePods(pods)
	// fmt.Fprintf(w, "There are %d pods in the cluster\n", len(kubePods))
	// for _, pod := range kubePods {
	// 	fmt.Fprintf(w, "Pod name: %s, restarts: %d, age: %s\n", pod.name, pod.restarts, pod.age)
	// }

}

func getPods(cs *kubernetes.Clientset) *v1.PodList {
	pods, err := cs.CoreV1().Pods("cgautamdevc14").List(context.TODO(), metav1.ListOptions{})

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

func main() {
	fmt.Println("Starting hello-world server...")
	// Create kubernetes client config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	ClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	// Create controller
	controller := OKtetoAPIController{K8sApi: &KubernetesAPI{ClientSet: ClientSet}}
	// Number of pods
	http.HandleFunc("/npods", controller.Npods)

	//sort pods
	http.HandleFunc("/pods", controller.Pods)
	// Prometheus metrics

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "cgautamdevc14",
		Name:      "pod_count",
		Help:      "Number of pods in the gautam namespace",
	}, func() float64 {
		npods, err := controller.K8sApi.GetNPods("cgautamdevc14")
		if err != nil {
			return -1
		}
		return float64(npods)
	})
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

// Define a "controller" so k8sAPI can be available
type OKtetoAPIController struct {
	K8sApi *KubernetesAPI
}

// Return number of pods in cgautamdevc14 namespace
func (controller *OKtetoAPIController) Npods(w http.ResponseWriter, r *http.Request) {
	npods, err := controller.K8sApi.GetNPods("cgautamdevc14")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, err.Error())))
		return
	}
	fmt.Fprintf(w, "%d", npods)
}

// Return pods with sort functionality
func (controller *OKtetoAPIController) Pods(w http.ResponseWriter, r *http.Request) {
	// Get sort parameter
	podResp, err := controller.K8sApi.GetPods("cgautamdevc14")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"message": "%s"}`, err.Error())))
		return
	}
	// Get sort type if present
	sort := strings.ToLower(r.URL.Query().Get("sort"))
	if sort != "" && sort != string(SortName) && sort != string(SortAge) && sort != string(SortRestarts) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"message": "invalid sort option. Valid options are \"%s\", \"%s\", or \"%s\""}`, SortName, SortAge, SortRestarts)))
		return
	} else if sort != "" {
		sortDirection := strings.ToLower(r.URL.Query().Get("order"))
		if sortDirection != "" && sortDirection != string(SortDescending) && sortDirection != string(SortAscending) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"message": "invalid sort direction. Valid options are \"%s\" or \"%s\""}`, SortAscending, SortDescending)))
			return
		} else if sortDirection == "" {
			// Ascending by default
			sortDirection = string(SortAscending)
		}
		// Perform sort
		controller.K8sApi.SortPods(podResp, PodSortMethod(sort), PodSortDirection(sortDirection))
	}
	// Serialize and return response
	marshalled, err := json.Marshal(podResp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"message": "%s"}`, err.Error())))
		return
	}
	w.Write(marshalled)
}
