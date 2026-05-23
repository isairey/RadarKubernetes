package k8s

import (
	"testing"

	"github.com/skyhook-io/radar/pkg/k8score"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

func TestIsMoreStableVersion(t *testing.T) {
	tests := []struct {
		newVer string
		oldVer string
		want   bool
	}{
		{"v1", "v1alpha1", true},
		{"v1", "v1beta1", true},
		{"v1beta1", "v1alpha1", true},
		{"v1alpha1", "v1beta1", false},
		{"v1beta1", "v1", false},
		{"v2", "v1", true},
		{"v1", "v2", false},
		{"v1beta2", "v1beta1", true},
		{"v1beta1", "v1beta2", false},
	}
	for _, tt := range tests {
		t.Run(tt.newVer+"_vs_"+tt.oldVer, func(t *testing.T) {
			got := isMoreStableVersion(tt.newVer, tt.oldVer)
			if got != tt.want {
				t.Errorf("isMoreStableVersion(%q, %q) = %v, want %v", tt.newVer, tt.oldVer, got, tt.want)
			}
		})
	}
}

func TestGetGVRWithGroup_DisambiguatesSameKind(t *testing.T) {
	// Create a fake clientset with two CRDs sharing the same Kind but different groups
	client := fakeclientset.NewSimpleClientset()
	fakeDisc := client.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDisc.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "argoproj.io/v1alpha1",
			APIResources: []metav1.APIResource{
				{Name: "applications", Kind: "Application", Namespaced: true, Verbs: metav1.Verbs{"list", "watch", "get"}},
			},
		},
		{
			GroupVersion: "app.k8s.io/v1beta1",
			APIResources: []metav1.APIResource{
				{Name: "applications", Kind: "Application", Namespaced: true, Verbs: metav1.Verbs{"list", "watch", "get"}},
			},
		},
	}

	// Build ResourceDiscovery via the real constructor
	core, err := k8score.NewResourceDiscovery(fakeDisc)
	if err != nil {
		t.Fatalf("NewResourceDiscovery failed: %v", err)
	}
	d := &ResourceDiscovery{ResourceDiscovery: core}

	// GetGVRWithGroup should return the correct group
	gvr, ok := d.GetGVRWithGroup("Application", "argoproj.io")
	if !ok {
		t.Fatal("expected to find Application in argoproj.io")
	}
	if gvr.Group != "argoproj.io" {
		t.Errorf("expected group argoproj.io, got %s", gvr.Group)
	}
	if gvr.Version != "v1alpha1" {
		t.Errorf("expected version v1alpha1, got %s", gvr.Version)
	}

	gvr, ok = d.GetGVRWithGroup("Application", "app.k8s.io")
	if !ok {
		t.Fatal("expected to find Application in app.k8s.io")
	}
	if gvr.Group != "app.k8s.io" {
		t.Errorf("expected group app.k8s.io, got %s", gvr.Group)
	}
	if gvr.Version != "v1beta1" {
		t.Errorf("expected version v1beta1, got %s", gvr.Version)
	}

	// Non-existent group should return false
	_, ok = d.GetGVRWithGroup("Application", "nonexistent.io")
	if ok {
		t.Error("expected not to find Application in nonexistent.io")
	}
}
