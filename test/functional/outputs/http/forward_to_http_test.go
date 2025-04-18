package http

import (
	"fmt"
	"strings"
	"time"

	obstestruntime "github.com/openshift/cluster-logging-operator/test/runtime/observability"

	"k8s.io/apimachinery/pkg/api/resource"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"

	"github.com/openshift/cluster-logging-operator/internal/runtime"
	"github.com/openshift/cluster-logging-operator/internal/utils"
	"github.com/openshift/cluster-logging-operator/test/framework/functional"
	"github.com/openshift/cluster-logging-operator/test/helpers/types"
)

var _ = Describe("[Functional][Outputs][Http] Functional tests", func() {

	var (
		framework *functional.CollectorFunctionalFramework
	)

	BeforeEach(func() {
		framework = functional.NewCollectorFunctionalFramework()
		obstestruntime.NewClusterLogForwarderBuilder(framework.Forwarder).
			FromInput(obs.InputTypeApplication).
			ToHttpOutput(func(output *obs.OutputSpec) {
				output.HTTP.Tuning = &obs.HTTPTuningSpec{
					BaseOutputTuningSpec: obs.BaseOutputTuningSpec{
						DeliveryMode:     obs.DeliveryModeAtLeastOnce,
						MaxRetryDuration: utils.GetPtr(time.Duration(30)),
						MinRetryDuration: utils.GetPtr(time.Duration(5)),
						MaxWrite:         utils.GetPtr(resource.MustParse("1M")),
					},
				}
			})
	})

	AfterEach(func() {
		framework.Cleanup()
	})

	DescribeTable("", func(addDestinationContainer func(f *functional.CollectorFunctionalFramework) runtime.PodBuilderVisitor) {

		Expect(framework.DeployWithVisitors([]runtime.PodBuilderVisitor{
			addDestinationContainer(framework),
			func(builder *runtime.PodBuilder) error {
				builder.AddLabels(map[string]string{
					"app.kubernetes.io/name": "somevalue",
					"foo.bar":                "a123",
				})
				return nil
			},
		})).To(BeNil())

		message := "hello world"
		timestamp := "2020-11-04T18:13:59.061892+00:00"

		applicationLogLine := fmt.Sprintf("%s stdout F %s", timestamp, message)
		Expect(framework.WriteMessagesToApplicationLog(applicationLogLine, 10)).To(BeNil())
		// Read line from Destination Http output
		result, err := framework.ReadFileFrom("http", functional.ApplicationLogFile)
		Expect(err).To(BeNil(), "Expected no errors reading the logs")
		Expect(result).ToNot(BeEmpty())
		raw := strings.Split(strings.TrimSpace(result), "\n")
		logs, err := types.ParseLogs(utils.ToJsonLogs(raw))
		Expect(err).To(BeNil(), fmt.Sprintf("Expected no errors parsing the logs: %s", raw[0]))
		// Compare to expected template
		Expect(logs[0].Message).To(Equal(message))
		Expect(logs[0].Kubernetes.Labels).To(HaveKey(MatchRegexp("^([a-zA-Z0-9_]*)$")))
		Expect(logs[0].Kubernetes.Labels).To(HaveKey(MatchRegexp("foo")))
	},
		Entry("should send message over http to vector", func(f *functional.CollectorFunctionalFramework) runtime.PodBuilderVisitor {
			userName := "imauser"
			password := "iwonttell"
			secretName := "mysecrets"
			framework.Forwarder.Spec.Outputs[0].HTTP.Authentication = &obs.HTTPAuthentication{
				Username: &obs.SecretReference{
					Key:        "username",
					SecretName: secretName,
				},
				Password: &obs.SecretReference{
					Key:        "password",
					SecretName: secretName,
				},
			}
			framework.Secrets = append(framework.Secrets, runtime.NewSecret(framework.Namespace, secretName, map[string][]byte{
				"username": []byte(userName),
				"password": []byte(password),
			}))

			return func(b *runtime.PodBuilder) error {
				return f.AddVectorHttpOutput(b, f.Forwarder.Spec.Outputs[0], functional.Option{Name: "username", Value: userName}, functional.Option{Name: "password", Value: password})
			}
		}),
		Entry("should send message over http to fluentd", func(f *functional.CollectorFunctionalFramework) runtime.PodBuilderVisitor {
			return func(b *runtime.PodBuilder) error {
				return f.AddFluentdHttpOutput(b, f.Forwarder.Spec.Outputs[0])
			}
		}),
	)

	Context("with tuning parameters", func() {
		var (
			addDestinationContainer func(f *functional.CollectorFunctionalFramework) runtime.PodBuilderVisitor
		)
		DescribeTable("with compression", func(compression string) {
			framework = functional.NewCollectorFunctionalFramework()
			obstestruntime.NewClusterLogForwarderBuilder(framework.Forwarder).
				FromInput(obs.InputTypeApplication).
				ToHttpOutput(func(output *obs.OutputSpec) {
					output.HTTP.Tuning = &obs.HTTPTuningSpec{
						Compression: compression,
					}
				})

			addDestinationContainer = func(f *functional.CollectorFunctionalFramework) runtime.PodBuilderVisitor {
				return func(b *runtime.PodBuilder) error {
					return f.AddVectorHttpOutput(b, f.Forwarder.Spec.Outputs[0])
				}
			}

			Expect(framework.DeployWithVisitors([]runtime.PodBuilderVisitor{
				addDestinationContainer(framework),
				func(builder *runtime.PodBuilder) error {
					builder.AddLabels(map[string]string{
						"app.kubernetes.io/name": "somevalue",
						"foo.bar":                "a123",
					})
					return nil
				},
			})).To(BeNil())

			msg := functional.NewCRIOLogMessage(functional.CRIOTime(time.Now()), "This is my test message", false)
			Expect(framework.WriteMessagesToApplicationLog(msg, 1)).To(BeNil())

			raw, err := framework.ReadRawApplicationLogsFrom(string(obs.OutputTypeHTTP))
			Expect(err).To(BeNil(), "Expected no errors reading the logs for type")
			Expect(raw).ToNot(BeEmpty())
		},
			Entry("should pass with gzip", "gzip"),
			Entry("should pass with snappy", "snappy"),
			Entry("should pass with zlib", "zlib"),
			Entry("should pass with no compression", "none"))
	})

})
