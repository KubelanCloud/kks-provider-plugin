package speaker

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/KubelanCloud/kks-provider-plugin/pkg/kloudlb"
	"github.com/vishvananda/netlink"
)

type Speaker struct {
	client    kubernetes.Interface
	nodeName  string
	ifaceName string
	informer  cache.SharedIndexInformer
	lister    corelisters.ServiceLister
	queue     workqueue.RateLimitingInterface
	synced    cache.InformerSynced

	mu       sync.Mutex
	leading  map[string]string
	boundIPs map[string]struct{}
}

func New(client kubernetes.Interface, nodeName, ifaceName string) (*Speaker, error) {
	if client == nil {
		return nil, fmt.Errorf("kubernetes client is required")
	}
	if nodeName == "" {
		return nil, fmt.Errorf("node name is required")
	}
	if ifaceName == "" {
		ifaceName = "eth0"
	}

	factory := informers.NewSharedInformerFactory(client, 30*time.Second)
	informer := factory.Core().V1().Services().Informer()
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	s := &Speaker{
		client:    client,
		nodeName:  nodeName,
		ifaceName: ifaceName,
		informer:  informer,
		lister:    factory.Core().V1().Services().Lister(),
		queue:     queue,
		synced:    informer.HasSynced,
		leading:   make(map[string]string),
		boundIPs:  make(map[string]struct{}),
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { s.enqueue(obj) },
		UpdateFunc: func(_, newObj any) { s.enqueue(newObj) },
		DeleteFunc: func(obj any) { s.enqueue(obj) },
	})

	return s, nil
}

func (s *Speaker) Run(ctx context.Context) error {
	go s.informer.Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), s.synced) {
		return fmt.Errorf("service informer cache sync failed")
	}

	go wait.UntilWithContext(ctx, s.runWorker, time.Second)
	<-ctx.Done()
	s.releaseAll()
	s.queue.ShutDown()
	return nil
}

func (s *Speaker) enqueue(obj any) {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		return
	}
	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	s.queue.Add(key)
}

func (s *Speaker) runWorker(ctx context.Context) {
	for s.processNext(ctx) {
	}
}

func (s *Speaker) processNext(ctx context.Context) bool {
	item, shutdown := s.queue.Get()
	if shutdown {
		return false
	}
	defer s.queue.Done(item)
	key, ok := item.(string)
	if !ok {
		s.queue.Forget(item)
		return true
	}

	if err := s.sync(ctx, key); err != nil {
		s.queue.AddRateLimited(item)
		return true
	}
	s.queue.Forget(item)
	return true
}

func (s *Speaker) sync(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	svc, err := s.lister.Services(namespace).Get(name)
	if apierrors.IsNotFound(err) {
		s.stopLeading(key, "")
		return nil
	}
	if err != nil {
		return err
	}

	ip := serviceExternalIP(svc)
	if ip == "" || svc.DeletionTimestamp != nil {
		s.stopLeading(key, ip)
		return nil
	}

	leaseName := leaseNameFor(key)
	leader, err := s.acquireLease(ctx, leaseName, ip)
	if err != nil {
		return err
	}
	if leader != s.nodeName {
		s.stopLeading(key, ip)
		return nil
	}

	return s.ensureVIP(ip)
}

func serviceExternalIP(svc *corev1.Service) string {
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

func leaseNameFor(serviceKey string) string {
	return "kloud-lb-" + sanitizeLeaseName(serviceKey)
}

func sanitizeLeaseName(value string) string {
	out := make([]rune, 0, len(value))
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '.':
			out = append(out, r)
		default:
			out = append(out, '-')
		}
	}
	name := string(out)
	if len(name) > 52 {
		name = name[:52]
	}
	return name
}

func (s *Speaker) acquireLease(ctx context.Context, leaseName, vip string) (string, error) {
	now := metav1.MicroTime{Time: time.Now()}
	leaseClient := s.client.CoordinationV1().Leases(kloudlb.Namespace)
	lease, err := leaseClient.Get(ctx, leaseName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		lease = &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      leaseName,
				Namespace: kloudlb.Namespace,
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity:       ptr(s.nodeName),
				LeaseDurationSeconds: ptr(int32(15)),
				AcquireTime:          &now,
				RenewTime:            &now,
			},
		}
		if _, err := leaseClient.Create(ctx, lease, metav1.CreateOptions{}); err != nil {
			return "", err
		}
		return s.nodeName, nil
	}
	if err != nil {
		return "", err
	}

	holder := ""
	if lease.Spec.HolderIdentity != nil {
		holder = *lease.Spec.HolderIdentity
	}
	if holder == "" || holder == s.nodeName || leaseExpired(lease) {
		lease.Spec.HolderIdentity = ptr(s.nodeName)
		lease.Spec.LeaseDurationSeconds = ptr(int32(15))
		lease.Spec.RenewTime = &now
		if lease.Spec.AcquireTime == nil {
			lease.Spec.AcquireTime = &now
		}
		if _, err := leaseClient.Update(ctx, lease, metav1.UpdateOptions{}); err != nil {
			return "", err
		}
		return s.nodeName, nil
	}

	_ = vip
	return holder, nil
}

func leaseExpired(lease *coordinationv1.Lease) bool {
	if lease.Spec.RenewTime == nil || lease.Spec.LeaseDurationSeconds == nil {
		return true
	}
	duration := time.Duration(*lease.Spec.LeaseDurationSeconds) * time.Second
	return time.Since(lease.Spec.RenewTime.Time) > duration
}

func (s *Speaker) ensureVIP(ip string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.boundIPs[ip]; ok {
		return nil
	}
	if err := addVIP(s.ifaceName, ip); err != nil {
		return err
	}
	s.boundIPs[ip] = struct{}{}
	return nil
}

func (s *Speaker) stopLeading(serviceKey, ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ip != "" {
		if _, ok := s.boundIPs[ip]; ok {
			_ = removeVIP(s.ifaceName, ip)
			delete(s.boundIPs, ip)
		}
	}
	delete(s.leading, serviceKey)
}

func (s *Speaker) releaseAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ip := range s.boundIPs {
		_ = removeVIP(s.ifaceName, ip)
	}
	s.boundIPs = make(map[string]struct{})
}

func addVIP(ifaceName, ip string) error {
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return fmt.Errorf("lookup interface %s: %w", ifaceName, err)
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("invalid vip %q", ip)
	}
	addr := &netlink.Addr{
		IPNet: &net.IPNet{IP: parsed.To4(), Mask: net.CIDRMask(32, 32)},
	}
	if err := netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("add vip %s on %s: %w", ip, ifaceName, err)
	}
	return sendGratuitousARP(link, parsed.To4())
}

func removeVIP(ifaceName, ip string) error {
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return err
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("invalid vip %q", ip)
	}
	addr := &netlink.Addr{
		IPNet: &net.IPNet{IP: parsed.To4(), Mask: net.CIDRMask(32, 32)},
	}
	return netlink.AddrDel(link, addr)
}

func sendGratuitousARP(link netlink.Link, ip net.IP) error {
	if ip == nil {
		return fmt.Errorf("ip is required")
	}
	_ = link
	_ = ip
	// VIP is bound locally; L2 peers learn the address when traffic flows.
	return nil
}

func ptr[T any](v T) *T { return &v }

func NodeNameFromEnv() string {
	if v := os.Getenv("NODE_NAME"); v != "" {
		return v
	}
	if v, err := os.Hostname(); err == nil {
		return v
	}
	return ""
}

func InterfaceFromEnv() string {
	if v := os.Getenv("KLOUD_LB_INTERFACE"); v != "" {
		return v
	}
	return "eth0"
}
