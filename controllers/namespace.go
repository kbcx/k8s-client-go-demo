package controllers

import (
  "net/http"

  "github.com/beego/beego/v2/core/logs"
  beego "github.com/beego/beego/v2/server/web"

  "github.com/kbcx/k8s-client-go-demo/k8s"
)

type K8sNamespaceController struct {
  beego.Controller
}

func (c *K8sNamespaceController) Get() {
  nsClient := k8s.Namespace{}
  namespaces, err := nsClient.ListNamespaces()
  logs.Debug("namespace: %#v, err: %#v", namespaces, err)
  if err != nil {
    c.Data["json"] = map[string]string{"error": err.Error()}
    c.Ctx.Output.Status = http.StatusBadRequest
  } else {
    c.Data["json"] = namespaces
    c.Ctx.Output.Status = http.StatusOK
  }
  _ = c.ServeJSON()
}
