package exporter

import (
	"spec-recon/internal/config"
	"spec-recon/internal/model"
)

// Exporter is the unified interface for all reporting strategies
type Exporter interface {
	Export(summary *model.Summary, tree []*model.Node, cfg *config.Config) error
}
