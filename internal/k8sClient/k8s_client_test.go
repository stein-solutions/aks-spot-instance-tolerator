package k8sClient

import (
	"testing"
)

func TestK8sTest1(t *testing.T) {
	t.Parallel()

	//todo implement test

	// client := NewK8sClientDefault()

	// nodes, err := client.Clientset().CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	// if err != nil {
	// 	log.Fatalf("Error listing nodes: %v", err)
	// }

	// // Nodes ausgeben
	// fmt.Println("Nodes in the cluster:")
	// for _, node := range nodes.Items {
	// 	fmt.Printf("Name: %s\n", node.Name)
	// }

}
