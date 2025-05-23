package normalization

import (
	"encoding/json"
	"github.com/openshift/cluster-logging-operator/test/framework/functional"
	testruntime "github.com/openshift/cluster-logging-operator/test/runtime/observability"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"
	"github.com/openshift/cluster-logging-operator/internal/utils"
	"github.com/openshift/cluster-logging-operator/test/helpers/types"
	"github.com/openshift/cluster-logging-operator/test/matchers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
)

var _ = Describe("[Functional][Normalization] Messages from EventRouter", func() {

	const timestamp string = "1985-10-21T09:00:00.00000+00:00"
	var (
		framework                          *functional.CollectorFunctionalFramework
		writeMsg                           func(msg string) error
		templateForAnyKubernetesWithEvents = types.KubernetesWithEvent{
			Kubernetes: functional.TemplateForAnyKubernetes,
		}
		NewEventDataBuilder = func(verb string, podRef *corev1.ObjectReference) types.EventData {
			newEvent := types.NewEvent(podRef, corev1.EventTypeNormal, "reason", "amessage")
			if verb == "UPDATED" {
				oldEvent := types.NewEvent(podRef, corev1.EventTypeWarning, "old_reason", "old_message")
				return types.EventData{Verb: "UPDATED", Event: newEvent, OldEvent: oldEvent}
			} else {
				return types.EventData{Verb: "ADDED", Event: newEvent}
			}
		}

		ExpectedLogTemplateBuilder = func(event, oldEvent *corev1.Event) types.EventRouterLog {
			tmpl := types.EventRouterLog{
				Kubernetes: templateForAnyKubernetesWithEvents,
				ViaQCommon: types.ViaQCommon{
					Message:          event.Message,
					Level:            types.AnyString,
					Hostname:         types.AnyString,
					PipelineMetadata: types.PipelineMetadata{},
					Timestamp:        time.Time{},
					TimestampLegacy:  time.Time{},
					LogSource:        string(obs.InfrastructureSourceContainer),
					LogType:          string(obs.InputTypeApplication),
					Openshift: types.OpenshiftMeta{
						ClusterID: types.AnyString,
						Sequence:  types.NewOptionalInt(""),
					},
				},
			}
			//optional for test given we are mocking and these values may not map to actual meta
			tmpl.Kubernetes.ContainerImage = types.OptionalString
			tmpl.Kubernetes.ContainerImageID = types.OptionalString
			tmpl.Kubernetes.PodID = types.OptionalString
			tmpl.Kubernetes.Event = types.ViaqEventRouterEvent{
				Event: *event,
				Verb:  types.AnyString,
			}
			tmpl.Kubernetes.Event.Event.Message = ""
			if oldEvent != nil {
				tmpl.OldEvent = oldEvent
			}

			return tmpl
		}
	)

	BeforeEach(func() {
		framework = functional.NewCollectorFunctionalFramework()
		testruntime.NewClusterLogForwarderBuilder(framework.Forwarder).
			FromInput(obs.InputTypeApplication).
			ToHttpOutput()
		// vector only collects logs using pods, namespaces, containers it knows about.
		writeMsg = func(msg string) error {
			return framework.WriteMessagesToApplicationLog(msg, 1)
		}
		framework.VisitConfig = func(conf string) string {
			return strings.Replace(conf, `"eventrouter-"`, `"functional"`, 1)
		}
		Expect(framework.Deploy()).To(BeNil())

	})
	AfterEach(func() {
		framework.Cleanup()
	})

	DescribeTable("should be normalized to the VIAQ data model", func(verb string) {
		podRef, err := reference.GetReference(scheme.Scheme, types.NewMockPod())
		Expect(err).To(BeNil())
		newEventData := NewEventDataBuilder(verb, podRef)
		jsonBytes, _ := json.Marshal(newEventData)
		jsonStr := string(jsonBytes)
		msg := functional.NewCRIOLogMessage(timestamp, jsonStr, false)
		err = writeMsg(msg)
		Expect(err).To(BeNil())

		raw, err := framework.ReadRawApplicationLogsFrom(string(obs.OutputTypeHTTP))
		Expect(err).To(BeNil(), "Expected no errors reading the logs")
		var logs []types.EventRouterLog
		err = types.StrictlyParseLogs(utils.ToJsonLogs(raw), &logs)
		Expect(err).To(BeNil(), "Expected no errors parsing the logs")
		var expectedLogTemplate = ExpectedLogTemplateBuilder(newEventData.Event, newEventData.OldEvent)
		Expect(logs[0]).To(matchers.FitLogFormatTemplate(expectedLogTemplate))
	},
		Entry("for ADDED events", "ADDED"),
		Entry("for UPDATED events", "UPDATED"),
	)

})
