package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

type KubernetesAPI struct {
	ClientSet     *kubernetes.Clientset
	MockClientSet *testclient.Clientset
}

// Represents the available methods to sort the pod list
type PodSortMethod string

const (
	SortName     PodSortMethod = "name"
	SortAge      PodSortMethod = "age"
	SortRestarts PodSortMethod = "restarts"
)

// Represents the direction the sort can be applied
type PodSortDirection string

const (
	SortAscending  PodSortDirection = "asc"
	SortDescending PodSortDirection = "desc"
)

// Represents a kubernetes Pod
type PodResponse struct {
	Name      string    `json:"name"`
	Age       string    `json:"age"`
	CreatedTS time.Time `json:"created_ts"`
	Restarts  int       `json:"restarts"`
}

// Marshal/Unmarshal methods that allow for proper serialization+deserialization of time.Time
func (p PodResponse) MarshalJSON() ([]byte, error) {
	type Alias PodResponse
	basicPodResponse := struct {
		Alias
		CreatedTS string `json:"created_ts"`
	}{
		Alias:     (Alias)(p),
		CreatedTS: p.CreatedTS.Format(time.RFC3339),
	}

	return json.Marshal(basicPodResponse)
}

func (p *PodResponse) UnmarshalJSON(j []byte) error {
	var rawStrings map[string]interface{}

	err := json.Unmarshal(j, &rawStrings)
	if err != nil {
		return err
	}

	for k, v := range rawStrings {
		if strings.ToLower(k) == "name" {
			p.Name = fmt.Sprintf("%s", v)
		} else if strings.ToLower(k) == "age" {
			p.Age = fmt.Sprintf("%s", v)
		} else if strings.ToLower(k) == "created_ts" {
			t, err := time.Parse(time.RFC3339, fmt.Sprintf("%s", v))
			if err != nil {
				return err
			}
			p.CreatedTS = t
		} else if strings.ToLower(k) == "restarts" {
			p.Restarts = int(v.(float64))
		}
	}

	return nil
}

// Kuberentes API Utilities

// Return []PodResponse for all pods in the given namespace
func (k *KubernetesAPI) GetPods(namespace string) ([]PodResponse, error) {
	// Get all pods in namespace
	var pods *v1.PodList
	var err error
	if k.ClientSet != nil {
		pods, err = k.ClientSet.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	} else if k.MockClientSet != nil {
		pods, err = k.MockClientSet.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	} else {
		panic("No kubernetes client set")
	}
	if err != nil {
		fmt.Printf("Error retrieving list of pods %s", err.Error())
		return nil, err
	}
	// Create response slice
	var podSlice = make([]PodResponse, len(pods.Items))
	for i, pod := range pods.Items {
		// Get restart count
		restarts := 0
		for _, cStatus := range pod.Status.ContainerStatuses {
			restarts += int(cStatus.RestartCount)
		}
		// PodResponse item
		podSlice[i] = PodResponse{
			Name:      pod.Name,
			Age:       FormatAgeString(pod.CreationTimestamp.Time, nil),
			CreatedTS: pod.GetCreationTimestamp().Time,
			Restarts:  restarts,
		}
	}

	return podSlice, nil
}

// FormatAgeString, allow custom now() func for testing
func FormatAgeString(createdTS time.Time, now func() time.Time) string {
	if now == nil {
		now = time.Now
	}
	age := now().Sub(createdTS)
	if age.Minutes() < 1 {
		return fmt.Sprintf("%.2f seconds", age.Seconds())
	} else if age.Hours() < 1 {
		return fmt.Sprintf("%.2f minutes", age.Minutes())
	} else if age.Hours() < 24 {
		return fmt.Sprintf("%.2f hours", age.Hours())
	}
	return fmt.Sprintf("%d days", int(age.Hours()/24))
}

// Sort a PodResponse slice given parameters
func (k *KubernetesAPI) SortPods(podResponse []PodResponse, sortMethod PodSortMethod, sortDirection PodSortDirection) {
	sort.Slice(podResponse, func(a, b int) bool {
		switch sortMethod {
		// Sort by name (string)
		case SortName:
			if sortDirection == SortAscending {
				return podResponse[a].Name < podResponse[b].Name
			}
			return podResponse[a].Name > podResponse[b].Name
		// Sort by age (timestamp)
		case SortAge:
			if sortDirection == SortAscending {
				return podResponse[a].CreatedTS.After(podResponse[b].CreatedTS)
			}
			return podResponse[a].CreatedTS.Before(podResponse[b].CreatedTS)
		// Sort by restarts (int)
		case SortRestarts:
			if sortDirection == SortAscending {
				return podResponse[a].Restarts < podResponse[b].Restarts
			}
			return podResponse[a].Restarts > podResponse[b].Restarts
		}
		return false
	})
}

// Return the number of pods in the namespace
func (k *KubernetesAPI) GetNPods(namespace string) (int, error) {
	var pods *v1.PodList
	var err error
	// Get all pods in namespace
	if k.ClientSet != nil {
		pods, err = k.ClientSet.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	} else if k.MockClientSet != nil {
		pods, err = k.MockClientSet.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	} else {
		panic("No client in GetNPOds")
	}
	if err != nil {
		fmt.Printf("Error retrieving list of pods %s", err.Error())
		return -1, err
	}
	return len(pods.Items), nil
}
