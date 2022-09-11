package k8s

import (
  "context"

  v1 "k8s.io/api/core/v1"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/klog/v2"
)

type Namespace struct{}

func (n *Namespace) ListNamespaces() ([]v1.Namespace, error) {
  ctx := context.Background()
  clientSet, _, err := GetK8sClientSet()
  if err != nil {
    klog.Fatal(err)
    return nil, err
  }

  namespaceList, err := clientSet.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
  if err != nil {
    klog.Fatal(err)
    return nil, err
  }
  return namespaceList.Items, nil
}
