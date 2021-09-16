package watcher

import (
	"context"
	"fmt"
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	"istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"log"
	"sd-for-vm-telemetry/event"
	"sync"
	"time"
)

type Watcher struct {
	istioClient *versioned.Clientset
	k8sClient   *kubernetes.Clientset
	eventChan   chan<- event.Event
	namespace   string
	stop        <-chan struct{}
	retry       chan int
	watch       watch.Interface
	initLock    sync.Mutex
}

func (w *Watcher) Start() {
	go w.handleRetry()

	go w.handleStop()

	watchWLE, err := w.istioClient.NetworkingV1beta1().WorkloadEntries(w.namespace).Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Namespace %s watch init error, err: %v", w.namespace, err)
		w.retry <- 1
	}
	w.watch = watchWLE
	go w.handleEvent(1)
}

func (w *Watcher) handleEvent(retryNum int) {
	clearMap := make(map[string]struct{}, 5)

	for e := range w.watch.ResultChan() {
		// handle events from the workload entries watch
		wle, ok := e.Object.(*v1beta1.WorkloadEntry)
		if !ok {
			log.Print("unexpected type")
		}
		switch e.Type {
		case watch.Error:
			status := e.Object.(*metav1.Status)
			log.Printf("Namespace %s watch return error, status: %s\n", w.namespace, status.String())
			w.retry <- retryNum
			return
		default: // add or update or delete
			if _, ok := clearMap[wle.Name]; ok {
				// 接收到重复的源事件  才清空重试计数器
				retryNum = 1
			}
			clearMap[wle.Name] = struct{}{}

			newTargetAddr := fmt.Sprintf("%s:15020", wle.Spec.Address)
			log.Printf("Namespace %s listens to event, type: %s, target: %s\n", w.namespace, e.Type, newTargetAddr)

			et := event.TypePut
			if e.Type == watch.Deleted {
				et = event.TypeDelete
			}
			w.eventChan <- event.Event{
				Type:      et,
				Name:      wle.Name,
				Namespace: w.namespace,
				Address:   newTargetAddr,
			}
		}
	}
}

func (w *Watcher) handleRetry() {
	for retryNum := range w.retry {
		w.initLock.Lock()
		w.watch = nil
		w.initLock.Unlock()

		delayTime := getTryDelayTime(retryNum)
		log.Printf("Namespace %s retry watch at %ds", w.namespace, delayTime/time.Second)
		select {
		case <-time.After(delayTime):
			break
		case <-w.stop:
			return
		}

		watchWLE, err := w.istioClient.NetworkingV1beta1().WorkloadEntries(w.namespace).Watch(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Printf("Namespace %s watch retry error, err: %v\n", w.namespace, err)
			go func() {
				w.retry <- retryNum + 1
			}()
			continue
		}
		w.initLock.Lock()
		w.watch = watchWLE
		w.initLock.Unlock()
		go w.handleEvent(retryNum + 1)
	}
}

func (w *Watcher) handleStop() {
	<-w.stop
	log.Printf("Namespace %s stop watch\n", w.namespace)
	close(w.retry)
	w.initLock.Lock()
	if w.watch != nil {
		w.watch.Stop()
	}
	w.initLock.Unlock()
}

func getTryDelayTime(retryNum int) time.Duration {
	const maxDelayTime = time.Minute * 5
	delayTime := 5 * time.Duration(retryNum*retryNum) * time.Second
	if delayTime > maxDelayTime {
		return maxDelayTime
	}
	return delayTime
}
