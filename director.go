package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func newKubeDirector() *KubeDirector {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	kube, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	kd := &KubeDirector{
		kube: kube,
	}
	kd.Update()
	go func() {
		for range time.Tick(1 * time.Minute) {
			kd.Update()
		}
	}()
	return kd
}

type KubeDirector struct {
	kube kubernetes.Interface

	paths   map[string]string
	domains map[string]struct{}
	lock    sync.RWMutex
}

func (kd *KubeDirector) Direct(req *http.Request) {
	kd.lock.RLock()
	defer kd.lock.RUnlock()

	log.Printf("Lookup %q", req.Host)

	host := kd.paths[req.Host]

	if host == "" {
		req.URL = nil
		return
	}

	req.Header.Set("X-Forwarded-Proto", "https")

	old := req.URL.String()

	req.URL.Scheme = "http"
	req.URL.Host = host

	log.Printf("Routing %s to %s", old, req.URL.String())

	if _, ok := req.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		req.Header.Set("User-Agent", "")
	}
	req.Header.Del("X-Forwarded-For")
}

func (kd *KubeDirector) Update() {
	services, _ := kd.kube.CoreV1().Services(meta.NamespaceAll).List(meta.ListOptions{})

	newPaths := map[string]string{}
	newDomains := map[string]struct{}{}
	for _, svc := range services.Items {
		if target, ok := svc.ObjectMeta.Annotations["andyleap.dev/singress-target"]; ok {
			newPaths[target] = fmt.Sprintf("%s.%s:%d", svc.Name, svc.Namespace, svc.Spec.Ports[0].Port)
			splitDomain := strings.Split(target, "/")
			newDomains[splitDomain[0]] = struct{}{}
		}
		if target, ok := svc.ObjectMeta.Annotations["git.andyleap.dev/singress-target"]; ok {
			newPaths[target] = fmt.Sprintf("%s.%s:%d", svc.Name, svc.Namespace, svc.Spec.Ports[0].Port)
			splitDomain := strings.Split(target, "/")
			newDomains[splitDomain[0]] = struct{}{}
		}
	}

	kd.lock.Lock()
	defer kd.lock.Unlock()
	kd.paths = newPaths
	kd.domains = newDomains
}
