package watcher

import (
	"istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"sd-for-vm-telemetry/event"
)

type Builder Watcher

func NewBuild() *Builder {
	return &Builder{}
}

func (b *Builder) WithIstioClient(istioClient *versioned.Clientset) *Builder {
	b.istioClient = istioClient
	return b
}

func (b *Builder) WithK8sClient(k8sClient *kubernetes.Clientset) *Builder {
	b.k8sClient = k8sClient
	return b
}

func (b *Builder) WithEventChan(eventChan chan<- event.Event) *Builder {
	b.eventChan = eventChan
	return b
}

func (b *Builder) WithNamespace(ns string) *Builder {
	b.namespace = ns
	return b
}

func (b *Builder) WithStop(stop <-chan struct{}) *Builder {
	b.stop = stop
	return b
}

func (b *Builder) Build() *Watcher {
	bs := *b
	w := Watcher(bs)
	return &w
}
