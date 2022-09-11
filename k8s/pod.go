package k8s

import (
  "context"
  "encoding/json"
  "errors"
  "net/http"
  "sync"

  "github.com/gorilla/websocket"
  "k8s.io/api/core/v1"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/client-go/kubernetes/scheme"
  "k8s.io/client-go/tools/remotecommand"
  "k8s.io/klog/v2"
)

type Pod struct{}

func (p *Pod) ListPods(namespaceName string) ([]v1.Pod, error) {
  ctx := context.Background()
  clientSet, _, err := GetK8sClientSet()
  if err != nil {
    klog.Warning(err)
    return nil, err
  }

  podList, err := clientSet.CoreV1().Pods(namespaceName).List(ctx, metav1.ListOptions{})
  if err != nil {
    klog.Warning(err)
    return nil, err
  }
  return podList.Items, nil
}

func (p *Pod) GetPodDetail(namespaceName, podName string) (*v1.Pod, error) {
  ctx := context.Background()
  clientSet, _, err := GetK8sClientSet()
  if err != nil {
    klog.Warning(err)
    return nil, err
  }

  podDetail, err := clientSet.CoreV1().Pods(namespaceName).Get(ctx, podName, metav1.GetOptions{})
  if err != nil {
    klog.Warning(err)
    return nil, err
  }
  return podDetail, nil
}

type WebsocketMsg struct {
  MsgType int
  Data    []byte
}

type WebsocketProxy struct {
  websocketConn *websocket.Conn
  inChan        chan *WebsocketMsg
  outChan       chan *WebsocketMsg
  mutex         sync.Mutex
  isClosed      bool
  closeChan     chan byte
}

func (p *WebsocketProxy) Read() (msg *WebsocketMsg, err error) {
  select {
  case msg = <-p.inChan:
    return
  case <-p.closeChan:
    err = errors.New("websocket proxy closed")
  }
  return
}

func (p *WebsocketProxy) Write(msg WebsocketMsg) (err error) {
  select {
  case p.outChan <- &msg:
    return
  case <-p.closeChan:
    err = errors.New("websocket closed")
  }
  return
}

func (p *WebsocketProxy) Close() {
  p.mutex.Lock()
  defer p.mutex.Unlock()
  err := p.websocketConn.Close()
  if err != nil {
    klog.Errorf("close WebsocketProxy error %#v", err)
    return
  }
  if !p.isClosed {
    p.isClosed = true
    close(p.closeChan)
  }
}

func (p *WebsocketProxy) ReadLoop() {
  var (
    msgType int
    data    []byte
    msg     *WebsocketMsg
    err     error
  )

  for {
    if msgType, data, err = p.websocketConn.ReadMessage(); err != nil {
      klog.Warning(err)
      p.Close()
      return
    }

    msg = &WebsocketMsg{
      MsgType: msgType,
      Data:    data,
    }

    select {
    case p.inChan <- msg:
      continue
    case <-p.closeChan:
      klog.Warning("conn.closeChan get close msg")
      return
    }
  }
}

func (p *WebsocketProxy) WriteLoop() {
  var (
    msg *WebsocketMsg
    err error
  )

  for {
    select {
    case msg = <-p.outChan:
      if err = p.websocketConn.WriteMessage(msg.MsgType, msg.Data); err != nil {
        klog.Warning(err)
        p.Close()
        return
      }
    case <-p.closeChan:
      klog.Warning("conn.closeChan get close msg")
      return
    }
  }
}

var websocketUpgrader = websocket.Upgrader{
  CheckOrigin: func(r *http.Request) bool {
    return true
  },
}

func CreateWebsocketProxy(respWriter http.ResponseWriter, req *http.Request) (p *WebsocketProxy, err error) {
  var (
    websocketConn *websocket.Conn
  )
  if websocketConn, err = websocketUpgrader.Upgrade(respWriter, req, nil); err != nil {
    return nil, err
  }

  p = &WebsocketProxy{
    websocketConn: websocketConn,
    inChan:        make(chan *WebsocketMsg, 1024*10),
    outChan:       make(chan *WebsocketMsg, 1024*10),
    closeChan:     make(chan byte),
    isClosed:      false,
  }

  // create new read/write goroutine
  go p.ReadLoop()
  go p.WriteLoop()
  return
}

type WebsocketStream struct {
  websocketProxy *WebsocketProxy
  resizeEvent    chan remotecommand.TerminalSize
}

func (stream *WebsocketStream) Write(bs []byte) (size int, err error) {
  size = len(bs)
  t := make([]byte, size)
  copy(t, bs)
  err = stream.websocketProxy.Write(WebsocketMsg{
    MsgType: websocket.TextMessage,
    Data:    t,
  })
  return
}

type XtermMessage struct {
  MsgType string `json:"type"`
  Input   string `json:"input"`
  Rows    uint16 `json:"rows"`
  Cols    uint16 `json:"cols"`
}

func (stream *WebsocketStream) Read(bs []byte) (size int, err error) {
  var (
    msg      *WebsocketMsg
    xtermMsg XtermMessage
  )
  size = len(bs)
  t := make([]byte, size)
  copy(t, bs)
  if msg, err = stream.websocketProxy.Read(); err != nil {
    klog.Errorln(err)
    return
  }

  if err = json.Unmarshal(msg.Data, &xtermMsg); err != nil {
    klog.Errorln(err)
    return
  }

  if xtermMsg.MsgType == "resize" {
    stream.resizeEvent <- remotecommand.TerminalSize{
      Width:  xtermMsg.Cols,
      Height: xtermMsg.Rows}
  } else if xtermMsg.MsgType == "input" {
    size = len(xtermMsg.Input)
    copy(bs, xtermMsg.Input)
  } else {
    klog.Warning("un-support xterm msg type", xtermMsg.MsgType)
  }
  return
}

func (stream *WebsocketStream) Next() (size *remotecommand.TerminalSize) {
  result := <-stream.resizeEvent
  size = &result
  return
}

func NewWebsocketTerminal(namespaceName, podName, containerName, command string, respWrite http.ResponseWriter, req *http.Request) error { // noqa
  var (
    err      error
    executor remotecommand.Executor
    proxy    *WebsocketProxy
  )
  clientSet, config, err := GetK8sClientSet()
  if err != nil {
    klog.Fatal(err)
    return err
  }

  reqTerm := clientSet.CoreV1().RESTClient().Post().Namespace(namespaceName).
    Resource("pods").Name(podName).SubResource("exec").
    VersionedParams(&v1.PodExecOptions{
      Container: containerName,
      Command:   []string{command},
      Stdin:     true,
      Stdout:    true,
      Stderr:    true,
      TTY:       true,
    }, scheme.ParameterCodec)

  if executor, err = remotecommand.NewSPDYExecutor(config, "POST", reqTerm.URL()); err != nil {
    klog.Errorln(err)
    return err
  }

  if proxy, err = CreateWebsocketProxy(respWrite, req); err != nil {
    klog.Errorln(err)
    return err
  }

  stream := &WebsocketStream{
    websocketProxy: proxy,
    resizeEvent:    make(chan remotecommand.TerminalSize),
  }
  if err = executor.Stream(remotecommand.StreamOptions{
    Stdin:             stream,
    Stdout:            stream,
    Stderr:            stream,
    TerminalSizeQueue: stream,
    Tty:               true,
  }); err != nil {
    klog.Errorln(err)
    proxy.Close()
    return err
  }
  return err
}
