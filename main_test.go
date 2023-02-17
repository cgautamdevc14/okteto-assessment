package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

var K8sMockAPI KubernetesAPI

func SetUpTests() {
	K8sMockAPI = KubernetesAPI{MockClientSet: testclient.NewSimpleClientset()}

	// Create a few pods
	// Feb 1
	str := "2023-02-15T00:00:00.000Z"
	t1, _ := time.Parse(time.RFC3339, str)
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "pod1",
			Namespace:         "cgautamdevc14",
			CreationTimestamp: metav1.Time{t1},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "okteto:go",
					Image:           "okteto:go",
					ImagePullPolicy: "Always",
				},
			},
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name:         "okteto:go",
					RestartCount: 0,
				},
			},
		},
	}
	_, err := K8sMockAPI.MockClientSet.CoreV1().Pods(pod.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}
}

func TestGetNPods(t *testing.T) {
	SetUpTests()

	npods, err := K8sMockAPI.GetNPods("cgautadevc14")
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, npods)
}

func TestGetPods(t *testing.T) {
	SetUpTests()

	pods, err := K8sMockAPI.GetPods("cgautadevc14")
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(pods))

	// Check the first pod
	K8sMockAPI.SortPods(pods, SortName, SortAscending)
	assert.Equal(t, "pod1", pods[0].Name)
	assert.Equal(t, "2023-02-15T00:00:00Z", pods[0].CreatedTS.Format(time.RFC3339))
	assert.Equal(t, 0, pods[0].Restarts)

}
