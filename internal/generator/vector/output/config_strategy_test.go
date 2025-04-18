package output

import (
	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift/cluster-logging-operator/internal/generator/framework"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/output/common"
	"github.com/openshift/cluster-logging-operator/internal/utils"
	. "github.com/openshift/cluster-logging-operator/test/matchers"
	"k8s.io/apimachinery/pkg/api/resource"
)

type fakeSink struct {
	Compression string
}

func (s *fakeSink) SetCompression(algo string) {
	s.Compression = algo
}

var _ = Describe("ConfigStrategy for tuning Outputs", func() {

	const (
		ID = "id"
	)

	Context("Compression", func() {
		It("should not set the compression when empty", func() {
			output := NewOutput(obs.OutputSpec{
				Type: obs.OutputTypeElasticsearch,
				Elasticsearch: &obs.Elasticsearch{
					Tuning: &obs.ElasticsearchTuningSpec{
						Compression: "",
					},
				},
			}, nil, framework.NoOptions)
			sink := &fakeSink{}
			output.VisitSink(sink)
			Expect(sink.Compression).To(BeEmpty())
		})
		It("should set the compression when not empty or none", func() {
			output := NewOutput(obs.OutputSpec{
				Type: obs.OutputTypeElasticsearch,
				Elasticsearch: &obs.Elasticsearch{
					Tuning: &obs.ElasticsearchTuningSpec{
						Compression: "gzip",
					},
				},
			}, nil, framework.NoOptions)
			sink := &fakeSink{}
			output.VisitSink(sink)
			Expect(sink.Compression).To(Equal("gzip"))
		})
		It("should set the compression when not empty or none for Splunk", func() {
			output := NewOutput(obs.OutputSpec{
				Type: obs.OutputTypeSplunk,
				Splunk: &obs.Splunk{
					Tuning: &obs.SplunkTuningSpec{
						Compression: "gzip",
					},
				},
			}, nil, framework.NoOptions)
			sink := &fakeSink{}
			output.VisitSink(sink)
			Expect(sink.Compression).To(Equal("gzip"))
		})
		It("should set the compression when not empty or none for Kafka", func() {
			output := NewOutput(obs.OutputSpec{
				Type: obs.OutputTypeKafka,
				Kafka: &obs.Kafka{
					Tuning: &obs.KafkaTuningSpec{
						Compression: "snappy",
					},
				},
			}, nil, framework.NoOptions)
			sink := &fakeSink{}
			output.VisitSink(sink)
			Expect(sink.Compression).To(Equal("snappy"))
		})
		It("should set the compression when not empty or none for CloudWatch", func() {
			output := NewOutput(obs.OutputSpec{
				Type: obs.OutputTypeCloudwatch,
				Cloudwatch: &obs.Cloudwatch{
					Tuning: &obs.CloudwatchTuningSpec{
						Compression: "gzip",
					},
				},
			}, nil, framework.NoOptions)
			sink := &fakeSink{}
			output.VisitSink(sink)
			Expect(sink.Compression).To(Equal("gzip"))
		})
		It("should set the compression when not empty or none for Http", func() {
			output := NewOutput(obs.OutputSpec{
				Type: obs.OutputTypeHTTP,
				HTTP: &obs.HTTP{
					Tuning: &obs.HTTPTuningSpec{
						Compression: "gzip",
					},
				},
			}, nil, framework.NoOptions)
			sink := &fakeSink{}
			output.VisitSink(sink)
			Expect(sink.Compression).To(Equal("gzip"))
		})
		It("should set the compression when not empty or none for OTLP", func() {
			output := NewOutput(obs.OutputSpec{
				Type: obs.OutputTypeOTLP,
				OTLP: &obs.OTLP{
					Tuning: &obs.OTLPTuningSpec{
						Compression: "gzip",
					},
				},
			}, nil, framework.NoOptions)
			sink := &fakeSink{}
			output.VisitSink(sink)
			Expect(sink.Compression).To(Equal("gzip"))
		})

	})
	Context("MaxRetryDuration", func() {

		It("should rely upon the defaults and generate nothing when zero", func() {
			output := NewOutput(obs.OutputSpec{
				Type: obs.OutputTypeElasticsearch,
				Elasticsearch: &obs.Elasticsearch{
					Tuning: &obs.ElasticsearchTuningSpec{},
				},
			}, nil, nil)
			Expect(``).To(EqualConfigFrom(common.NewRequest(ID, output)))
		})

		It("should set request.retry_max_duration_secs for values greater then zero", func() {
			output := NewOutput(obs.OutputSpec{
				Type: obs.OutputTypeElasticsearch,
				Elasticsearch: &obs.Elasticsearch{
					Tuning: &obs.ElasticsearchTuningSpec{
						BaseOutputTuningSpec: obs.BaseOutputTuningSpec{
							MaxRetryDuration: utils.GetPtr(time.Duration(35)),
						},
					},
				},
			}, nil, nil)

			Expect(`
[sinks.id.request]
retry_max_duration_secs = 35
`).To(EqualConfigFrom(common.NewRequest(ID, output)))

		})
	})
	Context("MinRetryDuration", func() {

		It("should rely upon the defaults and generate nothing when zero", func() {
			output := NewOutput(obs.OutputSpec{
				Type: obs.OutputTypeElasticsearch,
				Elasticsearch: &obs.Elasticsearch{
					Tuning: &obs.ElasticsearchTuningSpec{},
				},
			}, nil, nil)
			Expect(``).To(EqualConfigFrom(common.NewRequest(ID, output)))
		})

		It("should set request.retry_initial_backoff_secs for values greater then zero", func() {
			output := NewOutput(obs.OutputSpec{
				Type: obs.OutputTypeElasticsearch,
				Elasticsearch: &obs.Elasticsearch{
					Tuning: &obs.ElasticsearchTuningSpec{
						BaseOutputTuningSpec: obs.BaseOutputTuningSpec{
							MinRetryDuration: utils.GetPtr(time.Duration(25)),
						},
					},
				},
			}, nil, nil)

			Expect(`
[sinks.id.request]
retry_initial_backoff_secs = 25
`).To(EqualConfigFrom(common.NewRequest(ID, output)))

		})
	})
	Context("MaxWrite", func() {

		It("should rely upon the defaults and generate nothing when zero", func() {
			output := NewOutput(obs.OutputSpec{
				Type: obs.OutputTypeElasticsearch,
				Elasticsearch: &obs.Elasticsearch{
					Tuning: &obs.ElasticsearchTuningSpec{},
				},
			}, nil, nil)
			Expect(``).To(EqualConfigFrom(common.NewBatch(ID, output)))
		})

		It("should set batch.max_bytes for values greater then zero", func() {
			output := NewOutput(obs.OutputSpec{
				Type: obs.OutputTypeElasticsearch,
				Elasticsearch: &obs.Elasticsearch{
					Tuning: &obs.ElasticsearchTuningSpec{
						BaseOutputTuningSpec: obs.BaseOutputTuningSpec{
							MaxWrite: utils.GetPtr(resource.MustParse("1Ki")),
						},
					},
				},
			}, nil, nil)

			Expect(`
[sinks.id.batch]
max_bytes = 1024
`).To(EqualConfigFrom(common.NewBatch(ID, output)))

		})
	})

	Context("when delivery is spec'd", func() {

		Context("AtLeastOnce", func() {
			var output = NewOutput(obs.OutputSpec{
				Type: obs.OutputTypeElasticsearch,
				Elasticsearch: &obs.Elasticsearch{
					Tuning: &obs.ElasticsearchTuningSpec{
						BaseOutputTuningSpec: obs.BaseOutputTuningSpec{
							DeliveryMode: obs.DeliveryModeAtLeastOnce,
						},
					},
				},
			}, nil, nil)
			It("should do nothing to enable acknowledgments", func() {
				Expect(``).To(EqualConfigFrom(common.NewAcknowledgments(ID, output)))
			})
			It("should block when the buffer becomes full", func() {
				Expect(`
[sinks.id.buffer]
type = "disk"
when_full = "block"
max_size = 268435488
`).To(EqualConfigFrom(common.NewBuffer(ID, output)))
			})
		})

		Context("AtMostOnce", func() {

			var output = NewOutput(obs.OutputSpec{
				Type: obs.OutputTypeElasticsearch,
				Elasticsearch: &obs.Elasticsearch{
					Tuning: &obs.ElasticsearchTuningSpec{
						BaseOutputTuningSpec: obs.BaseOutputTuningSpec{
							DeliveryMode: obs.DeliveryModeAtMostOnce,
						},
					},
				},
			}, nil, nil)

			It("should not enable acknowledgements and not be present", func() {
				Expect("").To(EqualConfigFrom(common.NewAcknowledgments(ID, output)))
				Expect("").To(EqualConfigFrom(common.NewAcknowledgments(ID, nil)), "exp it to handle a nil config strategy")
			})
			It("should drop_newest when the buffer becomes full", func() {
				Expect(`
[sinks.id.buffer]
when_full = "drop_newest"
`).To(EqualConfigFrom(common.NewBuffer(ID, output)))
			})
		})
	})
})
