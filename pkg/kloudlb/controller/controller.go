package controller

import (
	"context"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/KubelanCloud/kks-provider-plugin/pkg/kloudlb"
	lbapi "github.com/KubelanCloud/kks-provider-plugin/pkg/lb/api"
	"github.com/KubelanCloud/kks-provider-plugin/pkg/lb/provisioner"
)

type Controller struct {
	client   kubernetes.Interface
	lbClient *lbapi.Client
	informer cache.SharedIndexInformer
	lister   corelisters.ServiceLister
	queue    workqueue.RateLimitingInterface
	synced   cache.InformerSynced

	identity string
	leader   *leaderelection.LeaderElector
}

func New(client kubernetes.Interface, lbClient *lbapi.Client) (*Controller, error) {
	if client == nil {
		return nil, fmt.Errorf("kubernetes client is required")
	}
	if lbClient == nil {
		return nil, fmt.Errorf("lb api client is required")
	}

	factory := informers.NewSharedInformerFactory(client, 30*time.Second)
	informer := factory.Core().V1().Services().Informer()
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	c := &Controller{
		client:   client,
		lbClient: lbClient,
		informer: informer,
		lister:   factory.Core().V1().Services().Lister(),
		queue:    queue,
		synced:   informer.HasSynced,
		identity: controllerIdentity(),
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { c.enqueue(obj) },
		UpdateFunc: func(_, newObj any) { c.enqueue(newObj) },
		DeleteFunc: func(obj any) { c.enqueue(obj) },
	})

	return c, nil
}

func (c *Controller) Run(ctx context.Context, workers int) error {
	if workers <= 0 {
		workers = 1
	}

	go c.informer.Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), c.synced) {
		return fmt.Errorf("service informer cache sync failed")
	}

	elector, err := c.newLeaderElector()
	if err != nil {
		return err
	}
	c.leader = elector
	go c.leader.Run(ctx)

	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	<-ctx.Done()
	c.queue.ShutDown()
	return nil
}

func (c *Controller) enqueue(obj any) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	c.queue.Add(key)
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNext(ctx) {
	}
}

func (c *Controller) processNext(ctx context.Context) bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(item)
	key, ok := item.(string)
	if !ok {
		c.queue.Forget(item)
		return true
	}

	if err := c.sync(ctx, key); err != nil {
		c.queue.AddRateLimited(item)
		return true
	}
	c.queue.Forget(item)
	return true
}

func (c *Controller) sync(ctx context.Context, key string) error {
	if !c.isLeader() {
		return nil
	}

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	svc, err := c.lister.Services(namespace).Get(name)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return nil
	}

	if svc.DeletionTimestamp != nil {
		return c.finalize(ctx, svc)
	}

	if !containsString(svc.Finalizers, kloudlb.Finalizer) {
		patch := svc.DeepCopy()
		patch.Finalizers = append(patch.Finalizers, kloudlb.Finalizer)
		if _, err := c.client.CoreV1().Services(namespace).Update(ctx, patch, metav1.UpdateOptions{}); err != nil {
			return err
		}
		return nil
	}

	if ingressIP(svc) != "" {
		return nil
	}

	lb, err := c.lbClient.Allocate(ctx, provisioner.AllocateRequest{
		Namespace: namespace,
		Name:      name,
	})
	if err != nil {
		return err
	}

	patch := svc.DeepCopy()
	if patch.Annotations == nil {
		patch.Annotations = map[string]string{}
	}
	patch.Annotations[kloudlb.AnnotationIP] = lb.IP
	patch.Annotations[kloudlb.AnnotationLBID] = lb.ID
	patch.Status = corev1.ServiceStatus{
		LoadBalancer: corev1.LoadBalancerStatus{
			Ingress: []corev1.LoadBalancerIngress{{IP: lb.IP}},
		},
	}

	_, err = c.client.CoreV1().Services(namespace).UpdateStatus(ctx, patch, metav1.UpdateOptions{})
	return err
}

func (c *Controller) finalize(ctx context.Context, svc *corev1.Service) error {
	if !containsString(svc.Finalizers, kloudlb.Finalizer) {
		return nil
	}

	if id := svc.Annotations[kloudlb.AnnotationLBID]; id != "" {
		if err := c.lbClient.Release(ctx, id); err != nil {
			return err
		}
	}

	patch := svc.DeepCopy()
	patch.Finalizers = removeString(patch.Finalizers, kloudlb.Finalizer)
	_, err := c.client.CoreV1().Services(svc.Namespace).Update(ctx, patch, metav1.UpdateOptions{})
	return err
}

func ingressIP(svc *corev1.Service) string {
	for _, ing := range svc.Status.LoadBalancer.Ingress {
		if ing.IP != "" {
			return ing.IP
		}
	}
	if svc.Annotations != nil {
		return svc.Annotations[kloudlb.AnnotationIP]
	}
	return ""
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func removeString(items []string, target string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item != target {
			out = append(out, item)
		}
	}
	return out
}

func (c *Controller) isLeader() bool {
	return c.leader != nil && c.leader.IsLeader()
}

func (c *Controller) newLeaderElector() (*leaderelection.LeaderElector, error) {
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Namespace: kloudlb.Namespace,
			Name:      "kloud-lb-controller",
		},
		Client: c.client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: c.identity,
		},
	}

	return leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		ReleaseOnCancel: true,
		Name:            "kloud-lb-controller",
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(context.Context) {},
			OnStoppedLeading: func() {},
		},
	})
}

func controllerIdentity() string {
	if v := os.Getenv("NODE_NAME"); v != "" {
		return v
	}
	if v, err := os.Hostname(); err == nil && v != "" {
		return v
	}
	return fmt.Sprintf("controller-%d", time.Now().UnixNano())
}
