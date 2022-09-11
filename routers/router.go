package routers

import (
  beego "github.com/beego/beego/v2/server/web"
  "github.com/kbcx/k8s-client-go-demo/controllers"
)

func init() {
  beego.Router("/", &controllers.MainController{})
  beego.Router("/namespaces", &controllers.K8sNamespaceController{})
  beego.Router("/namespace/:namespaceName/pods", &controllers.K8sPodController{}, "GET:ListPods")
  beego.Router("/namespace/:namespaceName/pod/:podName", &controllers.K8sPodController{}, "GET:GetPodDetail")
  beego.Router("/namespace/:namespaceName/pod/:podName/container/:containerName", &controllers.K8sPodController{}, "GET:WebsocketTerminal")
}
