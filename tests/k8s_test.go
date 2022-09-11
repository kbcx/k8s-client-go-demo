package test

import (
  "context"
  "flag"
  "os"
  "path/filepath"
  "testing"

  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/client-go/kubernetes"
  "k8s.io/client-go/tools/clientcmd"
  "k8s.io/client-go/util/homedir"
)

func TestK8s(t *testing.T) {
  ctx := context.Background()
  var kubeConfig *string
  if home := homedir.HomeDir(); home != "" {
    kubeConfig = flag.String("kubeConfig", filepath.Join(home, ".kube", "config"), "")
    if _, err := os.Stat(*kubeConfig); err != nil {
      t.Logf("%s not exist, skip", *kubeConfig)
      return
    }
  } else {
    t.Log("un-support test, skip")
    return
  }
  flag.Parse()

  config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
  if err != nil {
    t.Fatal(err)
  }

  clientSet, err := kubernetes.NewForConfig(config)
  if err != nil {
    t.Fatal(err)
  }

  namespaceList, err := clientSet.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
  if err != nil {
    t.Fatal(err)
  }

  for _, namespace := range namespaceList.Items {
    t.Log(namespace.Name)
  }
}
