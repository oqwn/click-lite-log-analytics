package ingestion

import (
	"github.com/your-username/click-lite-log-analytics/backend/internal/errors"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
	"github.com/your-username/click-lite-log-analytics/backend/internal/tracing"
)

// LogProcessor processes logs through various analyzers
type LogProcessor struct {
	traceManager  *tracing.TraceManager
	errorDetector *errors.ErrorDetector
}

// NewLogProcessor creates a new log processor
func NewLogProcessor(traceManager *tracing.TraceManager, errorDetector *errors.ErrorDetector) *LogProcessor {
	return &LogProcessor{
		traceManager:  traceManager,
		errorDetector: errorDetector,
	}
}

// ProcessLog processes a log through all analyzers
func (p *LogProcessor) ProcessLog(log *models.Log) {
	// Process for trace correlation
	if p.traceManager != nil {
		p.traceManager.ProcessLog(log)
	}

	// Process for error detection
	if p.errorDetector != nil {
		detectedErrors := p.errorDetector.ProcessLog(log)
		if len(detectedErrors) > 0 {
			// Add error information to attributes
			if log.Attributes == nil {
				log.Attributes = make(map[string]interface{})
			}
			log.Attributes["detected_errors"] = detectedErrors
		}
	}
}

// ProcessBatch processes multiple logs
func (p *LogProcessor) ProcessBatch(logs []models.Log) {
	for i := range logs {
		p.ProcessLog(&logs[i])
	}
}