package runtime_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift/cluster-logging-operator/internal/runtime"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ConfigMapBuilder", func() {

	var (
		configmap *corev1.ConfigMap
		builder   *runtime.ConfigMapBuilder
		expLabels = map[string]string{"foo": "bar"}
	)

	BeforeEach(func() {
		configmap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{},
		}
		builder = runtime.NewConfigMapBuilder(configmap)
	})

	Context("#AddLabel", func() {
		It("should add the label when labels is not initialized", func() {
			builder.AddLabel("foo", "bar")
			Expect(configmap.Labels).To(Equal(expLabels))
		})

		It("should add the label when labels is initialized", func() {
			configmap.Labels = map[string]string{}
			builder.AddLabel("foo", "bar")
			Expect(configmap.Labels).To(Equal(expLabels))
		})
	})
	Context("#AddAnnotation", func() {
		It("should add the label when labels is not initialized", func() {
			builder.AddAnnotation("foo", "bar")
			Expect(configmap.Annotations).To(Equal(expLabels))
		})

		It("should add the label when labels is initialized", func() {
			configmap.Labels = map[string]string{}
			builder.AddAnnotation("foo", "bar")
			Expect(configmap.Annotations).To(Equal(expLabels))
		})
	})

})
