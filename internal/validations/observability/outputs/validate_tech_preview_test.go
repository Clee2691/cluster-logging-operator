package outputs

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"
	internalcontext "github.com/openshift/cluster-logging-operator/internal/api/context"
	"github.com/openshift/cluster-logging-operator/internal/constants"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Validating tech-preview annotation", func() {
	Context("#ValidateTechPreviewAnnotation", func() {
		var (
			k8sClient client.Client
			forwarder obs.ClusterLogForwarder
			context   internalcontext.ForwarderContext
			out       obs.OutputSpec
		)

		When("output type is OTLP", func() {
			BeforeEach(func() {
				out = obs.OutputSpec{
					Name: "my-output",
					Type: obs.OutputTypeOTLP,
				}
				forwarder = obs.ClusterLogForwarder{
					Spec: obs.ClusterLogForwarderSpec{
						Outputs: []obs.OutputSpec{out},
					},
				}
				forwarder.Annotations = map[string]string{"some.other.annotation/for-testing": "true"}
				k8sClient = fake.NewFakeClient()
				context = internalcontext.ForwarderContext{
					Client:    k8sClient,
					Reader:    k8sClient,
					Forwarder: &forwarder,
				}
			})
			It("should pass validation when annotation is included with either value", func() {
				forwarder.Annotations[constants.AnnotationOtlpOutputTechPreview] = "true"
				Expect(ValidateTechPreviewAnnotation(context)).To(BeEmpty())

				forwarder.Annotations[constants.AnnotationOtlpOutputTechPreview] = "enabled"
				Expect(ValidateTechPreviewAnnotation(context)).To(BeEmpty())
			})
			It("should pass validation when including additional types", func() {
				forwarder.Annotations[constants.AnnotationOtlpOutputTechPreview] = "enabled"
				out2 := obs.OutputSpec{
					Name: "my-out2",
					Type: obs.OutputTypeCloudwatch,
				}
				out3 := obs.OutputSpec{
					Name: "my-out3",
					Type: obs.OutputTypeLoki,
				}
				forwarder.Spec.Outputs = []obs.OutputSpec{out, out2, out3}
				Expect(ValidateTechPreviewAnnotation(context)).To(BeEmpty())
			})
			It("should fail validation when missing the annotation", func() {
				results := ValidateTechPreviewAnnotation(context)
				Expect(results).To(ContainElement(ContainSubstring(MissingAnnotationMessage)))
			})
			It("should fail validation when annotation has incorrect value", func() {
				forwarder.Annotations[constants.AnnotationOtlpOutputTechPreview] = "false"
				results := ValidateTechPreviewAnnotation(context)
				Expect(results).To(ContainElement(ContainSubstring(MissingAnnotationMessage)))
			})
		})

		When("output type is not OTEL related", func() {
			BeforeEach(func() {
				out = obs.OutputSpec{
					Name: "my-output",
					Type: obs.OutputTypeHTTP,
				}
				forwarder = obs.ClusterLogForwarder{
					Spec: obs.ClusterLogForwarderSpec{
						Outputs: []obs.OutputSpec{out},
					},
				}
				forwarder.Annotations = map[string]string{"some.other.annotation/for-testing": "true"}
				k8sClient = fake.NewFakeClient()
				context = internalcontext.ForwarderContext{
					Client:    k8sClient,
					Reader:    k8sClient,
					Forwarder: &forwarder,
				}
			})
			It("should pass validation when type is not OTLP", func() {
				out.Type = obs.OutputTypeHTTP
				// Return value is empty when validation passes
				Expect(ValidateTechPreviewAnnotation(context)).To(BeEmpty())
			})
		})

		When("output type is LokiStack", func() {
			// removing since lokistack type migrates internally to become either otlp or loki type
		})
	})
})