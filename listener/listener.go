package listener

import (
	"context"
	"istio.io/client-go/pkg/clientset/versioned"
	versionedclient "istio.io/client-go/pkg/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"os"
	"sd-for-vm-telemetry/event"
	"sd-for-vm-telemetry/watcher"
	"time"
)

var (
	defaultWatchNS = false
)

const (
	enabledDefaultWatchNS = "ENABLED_DEFAULT_WATCH_NS"
	LabelWatchNS          = "istio-vm-watch"

	StatusEnabled  = "enabled"
	StatusDisabled = "disabled"
)

type Listener struct {
	istioClient *versioned.Clientset
	k8sClient   *kubernetes.Clientset

	watcherMap   map[string]watchManager
	namespaceMap map[string]*v1.Namespace
	eventChan    chan event.Event
}

type watchManager struct {
	watcher *watcher.Watcher
	stop    chan<- struct{}
}

func NewListener(restConfig *rest.Config) *Listener {
	nsWatchEnabled := os.Getenv(enabledDefaultWatchNS)
	if nsWatchEnabled == StatusEnabled {
		defaultWatchNS = true
		log.Println("Enable watch of unlabeled namespace")
	} else {
		log.Printf("Listen to the namespace where the isti-vm-watch value is enabled")
	}

	// istio client
	ic, err := versionedclient.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("Failed to create istio client: %s", err)
	}

	// k8s client
	kc, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("Failed to create k8s client: %s", err)
	}

	lis := &Listener{
		istioClient: ic,
		k8sClient:   kc,

		eventChan: make(chan event.Event),
	}
	return lis
}

func (l *Listener) Run() {
	go func() {
	nsWatch:
		for {
			labelSelector := metav1.LabelSelector{}
			if !defaultWatchNS {
				labelSelector.MatchLabels[LabelWatchNS] = StatusEnabled
			}

			nsWatch, err := l.k8sClient.CoreV1().Namespaces().Watch(context.TODO(), metav1.ListOptions{
				LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
			})
			if err != nil {
				log.Printf("Watch namespace error, retry at 15s, err: %s\n", err.Error())
				<-time.After(15 * time.Second)
				continue
			}

			for e := range nsWatch.ResultChan() {
				ns := e.Object.(*v1.Namespace)
				if ns != nil && ns.Labels[LabelWatchNS] == StatusDisabled {
					continue
				}

				switch e.Type {
				case watch.Error:
					// watch error
					status := e.Object.(*metav1.Status)
					nsWatch.Stop()
					log.Printf("Watch namespace error, retry at 15s, status: %s\n", status.String())
					<-time.After(15 * time.Second)
					continue nsWatch
				case watch.Deleted:
					// delete namespace
					log.Printf("Namespace %s delete\n", ns.Name)
					if w, ok := l.watcherMap[ns.Name]; ok {
						log.Printf("Stop namespace %s watcher\n", ns.Name)
						stopWatcher(w)
						log.Printf("Send namespace %s delete event\n", ns.Name)
						l.eventChan <- event.Event{
							Type:      event.TypeDelete,
							Namespace: ns.Name,
						}
						delete(l.watcherMap, ns.Name)
						delete(l.namespaceMap, ns.Name)
					}
				default:
					// add or modify
					log.Printf("Listen namespace event, type is %s, name: %s\n", e.Type, ns.Name)
					if !isCareNamespace(ns) {
						if w, ok := l.watcherMap[ns.Name]; ok {
							log.Printf("Namespace %s cancel watcher\n", ns.Name)
							stopWatcher(w)

							delete(l.watcherMap, ns.Name)
							delete(l.namespaceMap, ns.Name)
							continue
						}

						log.Printf("Namespace %s is not enabled for monitoring\n", ns.Name)
						continue
					}

					if l.isDuplicate(ns.Name) {
						log.Printf("Namespace %s has been assigned watcher\n", ns.Name)
						continue
					}

					log.Printf("Namespace %s assigned watcher\n", ns.Name)
					l.watch(ns)
				}
			}
		}
	}()
}

func isCareNamespace(ns *v1.Namespace) bool {
	if defaultWatchNS {
		if ns.Labels[LabelWatchNS] == StatusDisabled {
			return false
		}
		return true
	}

	return ns.Labels[LabelWatchNS] == StatusEnabled
}

func stopWatcher(w watchManager) {
	close(w.stop)
}

func (l *Listener) isDuplicate(ns string) bool {
	_, ok := l.watcherMap[ns]
	return ok
}

func (l *Listener) watch(ns *v1.Namespace) {
	stop := make(chan struct{})
	w := watcher.NewBuild().
		WithK8sClient(l.k8sClient).
		WithIstioClient(l.istioClient).
		WithNamespace(ns.Name).
		WithEventChan(l.eventChan).
		WithStop(stop).
		Build()

	wm := watchManager{
		watcher: w,
		stop:    stop,
	}

	l.watcherMap[ns.Name] = wm
	l.namespaceMap[ns.Name] = ns

	w.Start()
}
