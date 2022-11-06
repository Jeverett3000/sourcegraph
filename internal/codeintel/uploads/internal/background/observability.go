package background

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/internal/honey"
	"github.com/sourcegraph/sourcegraph/internal/metrics"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

type operations struct {
	updateUploadsVisibleToCommits *observation.Operation

	// Worker metrics
	uploadProcessor *observation.Operation
	uploadSizeGuage prometheus.Gauge

	numReconcileScans   prometheus.Counter
	numReconcileDeletes prometheus.Counter
}

func newOperations(observationContext *observation.Context) *operations {
	m := metrics.NewREDMetrics(
		observationContext.Registerer,
		"codeintel_uploads_background",
		metrics.WithLabels("op"),
		metrics.WithCountHelp("Total number of method invocations."),
	)

	op := func(name string) *observation.Operation {
		return observationContext.Operation(observation.Op{
			Name:              fmt.Sprintf("codeintel.uploads.background.%s", name),
			MetricLabelValues: []string{name},
			Metrics:           m,
		})
	}

	counter := func(name, help string) prometheus.Counter {
		counter := prometheus.NewCounter(prometheus.CounterOpts{
			Name: name,
			Help: help,
		})

		observationContext.Registerer.MustRegister(counter)
		return counter
	}

	numReconcileScans := counter(
		"src_codeintel_uploads_reconciliation_records_scanned_total",
		"The number of upload records read from codeintel-db for reconciliation.",
	)
	numReconcileDeletes := counter(
		"src_codeintel_uploads_reconciliation_records_deleted_total",
		"The number of abandoned uploads deleted from the codeintel-db.",
	)

	honeyObservationContext := *observationContext
	honeyObservationContext.HoneyDataset = &honey.Dataset{Name: "codeintel-worker"}
	uploadProcessor := honeyObservationContext.Operation(observation.Op{
		Name: "codeintel.uploadHandler",
		ErrorFilter: func(err error) observation.ErrorFilterBehaviour {
			return observation.EmitForTraces | observation.EmitForHoney
		},
	})

	uploadSizeGuage := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "src_codeintel_upload_processor_upload_size",
		Help: "The combined size of uploads being processed at this instant by this worker.",
	})
	observationContext.Registerer.MustRegister(uploadSizeGuage)

	return &operations{
		updateUploadsVisibleToCommits: op("UpdateUploadsVisibleToCommits"),

		// Worker metrics
		uploadProcessor: uploadProcessor,
		uploadSizeGuage: uploadSizeGuage,

		numReconcileScans:   numReconcileScans,
		numReconcileDeletes: numReconcileDeletes,
	}
}
