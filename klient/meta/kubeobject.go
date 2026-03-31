package meta

// ObjectMeta holds metadata common to all resources.
type ObjectMeta struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// ListMeta holds metadata common to list resources.
// Reserved for future use (e.g. resourceVersion, continue token).
type ListMeta struct{}

// KubeObject contains the common fields for all Kubernetes-style API objects.
type KubeObject struct {
	APIVersion string     `json:"apiVersion,omitempty"`
	Kind       string     `json:"kind,omitempty"`
	Metadata   ObjectMeta `json:"metadata,omitempty"`
}

// KubeList contains the common fields for all Kubernetes-style list objects.
type KubeList struct {
	APIVersion string   `json:"apiVersion,omitempty"`
	Kind       string   `json:"kind,omitempty"`
	Metadata   ListMeta `json:"metadata,omitempty"`
}
