package k8s

import (
  "errors"
  "flag"
  "fmt"
  "os"
  "path/filepath"
  "sync"

  "k8s.io/client-go/kubernetes"
  "k8s.io/client-go/rest"
  "k8s.io/client-go/tools/clientcmd"
  "k8s.io/client-go/util/homedir"
  "k8s.io/klog/v2"
)

var (
  onceConfig    sync.Once
  onceClientSet sync.Once
  kubeConfig    *rest.Config
  kubeClientSet *kubernetes.Clientset
)

func GetK8sConfig() (config *rest.Config, err error) {
  onceConfig.Do(func() {
    var kubeConfigFile *string
    if home := homedir.HomeDir(); home != "" {
      kubeConfigFile = flag.String("kubeConfig", filepath.Join(home, ".kube", "config"), "")
      if _, err := os.Stat(*kubeConfigFile); err != nil {
        err = errors.New(fmt.Sprintf("%#v not found error", *kubeConfig))
        return
      }
    } else {
      err = errors.New(fmt.Sprintf("read config error, `~/.kube/config` home is not found"))
      return
    }
    flag.Parse()

    kubeConfig, err = clientcmd.BuildConfigFromFlags("", *kubeConfigFile)
    if err != nil {
      klog.Fatal(err)
      return
    }
  })

  return kubeConfig, nil
}

func GetK8sClientSet() (*kubernetes.Clientset, *rest.Config, error) {
  onceClientSet.Do(func() {
    config, err := GetK8sConfig()
    if err != nil {
      klog.Fatal(err)
      return
    }
    kubeClientSet, err = kubernetes.NewForConfig(config)
    if err != nil {
      klog.Fatal(err)
      return
    }
  })
  return kubeClientSet, kubeConfig, nil
}
