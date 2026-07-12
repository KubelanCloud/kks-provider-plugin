package controller_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/KubelanCloud/kks-provider-plugin/pkg/kloudlb"
)

func TestIngressIPPrefersStatus(t *testing.T) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{kloudlb.AnnotationIP: "172.173.200.1"},
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{{IP: "172.173.200.2"}},
			},
		},
	}

	ip := ingressIP(svc)
	if ip != "172.173.200.2" {
		t.Fatalf("ingressIP = %q", ip)
	}
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
