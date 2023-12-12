package daemonsets_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/cluster-logging-operator/internal/utils/comparators/daemonsets"
)

var _ = Describe("daemonset#AreSame", func() {

	var (
		current, desired *apps.DaemonSet
	)

	BeforeEach(func() {
		current = &apps.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"foo": "bar",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind: "Foo",
						Name: "Bar",
					},
				},
			},
			Spec: apps.DaemonSetSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{},
						},
						InitContainers: []v1.Container{
							{},
						},
					},
				},
			},
		}
		desired = current.DeepCopy()

	})
	Context("when evaluating daemonSetSpec", func() {

		It("should recognize the specs are different", func() {
			container := v1.Container{}
			desired.Spec.Template.Spec.Containers = append(desired.Spec.Template.Spec.Containers, container)
			ok, _ := daemonsets.AreSame(current, desired)
			Expect(ok).To(BeFalse())
		})

		It("should recognize the specs are same", func() {
			ok, _ := daemonsets.AreSame(current, desired)
			Expect(ok).To(BeTrue())
		})
	})

	Context("when evaluating labels", func() {

		It("should recognize the labels are different", func() {
			desired.Labels = map[string]string{"foo": "baz"}
			ok, _ := daemonsets.AreSame(current, desired)
			Expect(ok).To(BeFalse())
		})

		It("should recognize labels are same", func() {
			ok, _ := daemonsets.AreSame(current, desired)
			Expect(ok).To(BeTrue())
		})
	})

	Context("when evaluating ownerRefs", func() {

		It("should recognize ownerRefs are different", func() {
			desired.OwnerReferences = []metav1.OwnerReference{
				{
					Kind: "Foo",
					Name: "Baz",
				},
			}
			ok, _ := daemonsets.AreSame(current, desired)
			Expect(ok).To(BeFalse())
		})

		It("should recognize ownerRefs are same", func() {
			ok, _ := daemonsets.AreSame(current, desired)
			Expect(ok).To(BeTrue())
		})
	})
})
