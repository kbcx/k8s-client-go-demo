package controllers

import (
  "net/http"

  "github.com/beego/beego/v2/core/logs"
  beego "github.com/beego/beego/v2/server/web"

  "github.com/kbcx/k8s-client-go-demo/k8s"
)

type K8sPodController struct {
  beego.Controller
}

func (c *K8sPodController) ListPods() {
  namespaceName := c.Ctx.Input.Param(":namespaceName")
  podClient := k8s.Pod{}
  pods, err := podClient.ListPods(namespaceName)
  logs.Debug("pods: %#v, err: %#v", pods, err)
  if err != nil {
    c.Data["json"] = map[string]string{"error": err.Error()}
    c.Ctx.Output.Status = http.StatusBadRequest
  } else {
    c.Data["json"] = pods
    c.Ctx.Output.Status = http.StatusOK
  }
  _ = c.ServeJSON()
}

func (c *K8sPodController) GetPodDetail() {
  namespaceName := c.Ctx.Input.Param(":namespaceName")
  podName := c.Ctx.Input.Param(":podName")
  logs.Debug("namespaceName: %#v, podName: %#v", namespaceName, podName)

  podClient := k8s.Pod{}
  detail, err := podClient.GetPodDetail(namespaceName, podName)
  logs.Debug("podDetail: %#v, err: %#v", detail, err)
  if err != nil {
    c.Data["json"] = map[string]string{"error": err.Error()}
    c.Ctx.Output.Status = http.StatusBadRequest
  } else {
    c.Data["json"] = detail
    c.Ctx.Output.Status = http.StatusOK
  }
  _ = c.ServeJSON()
}

func (c *K8sPodController) WebsocketTerminal() {
  namespaceName := c.Ctx.Input.Param(":namespaceName")
  podName := c.Ctx.Input.Param(":podName")
  containerName := c.Ctx.Input.Param(":containerName")
  command := c.Ctx.Input.Query(":command")
  if command == "" {
    command = "sh"
  }
  err := k8s.NewWebsocketTerminal(namespaceName, podName, containerName, command, c.Ctx.ResponseWriter, c.Ctx.Request)
  logs.Debug("err: %#v", err)
  if err != nil {
    c.Data["json"] = map[string]string{"error": err.Error()}
    c.Ctx.Output.Status = http.StatusBadRequest
  }
  _ = c.ServeJSON()
}
