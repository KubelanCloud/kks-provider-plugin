package provisioner

type LoadBalancer struct {
	ID        string `json:"id"`
	IP        string `json:"ip"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type AllocateRequest struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}
