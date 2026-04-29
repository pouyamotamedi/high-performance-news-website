package testing

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ResultStorage handles storage and retrieval of test results
type ResultStorage struct {
	storageDir string
}

// NewResultStorage creates a new result storage instance
func NewResultStorage() *ResultStorage {
	return &ResultStorage{
		storageDir: "test-results/storage",
	}
}

// StoreReport stores a test report
func (r *ResultStorage) StoreReport(report *TestReport) error {
	if err := os.MkdirAll(r.storageDir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	filename := fmt.Sprintf("report-%s.json", report.ID)
	filepath := filepath.Join(r.storageDir, filename)
	
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("failed to encode report: %w", err)
	}
	
	log.Printf("Report stored: %s", filepath)
	return nil
}

// StorePipelineResult stores a pipeline result
func (r *ResultStorage) StorePipelineResult(result *PipelineResult) error {
	if err := os.MkdirAll(r.storageDir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	filename := fmt.Sprintf("pipeline-%s.json", result.ID)
	filepath := filepath.Join(r.storageDir, filename)
	
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create pipeline file: %w", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to encode pipeline result: %w", err)
	}
	
	log.Printf("Pipeline result stored: %s", filepath)
	return nil
}

// GetRecentReports retrieves recent test reports
func (r *ResultStorage) GetRecentReports(limit int) ([]TestReport, error) {
	files, err := filepath.Glob(filepath.Join(r.storageDir, "report-*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list report files: %w", err)
	}
	
	// Sort files by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		infoI, _ := os.Stat(files[i])
		infoJ, _ := os.Stat(files[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})
	
	var reports []TestReport
	for i, file := range files {
		if i >= limit {
			break
		}
		
		report, err := r.loadReport(file)
		if err != nil {
			log.Printf("Warning: failed to load report %s: %v", file, err)
			continue
		}
		
		reports = append(reports, *report)
	}
	
	return reports, nil
}

// GetRecentPipelineResults retrieves recent pipeline results
func (r *ResultStorage) GetRecentPipelineResults(limit int) ([]PipelineResult, error) {
	files, err := filepath.Glob(filepath.Join(r.storageDir, "pipeline-*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list pipeline files: %w", err)
	}
	
	// Sort files by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		infoI, _ := os.Stat(files[i])
		infoJ, _ := os.Stat(files[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})
	
	var results []PipelineResult
	for i, file := range files {
		if i >= limit {
			break
		}
		
		result, err := r.loadPipelineResult(file)
		if err != nil {
			log.Printf("Warning: failed to load pipeline result %s: %v", file, err)
			continue
		}
		
		results = append(results, *result)
	}
	
	return results, nil
}

// GetReportsByPeriod retrieves reports within a time period
func (r *ResultStorage) GetReportsByPeriod(start, end time.Time) ([]TestReport, error) {
	files, err := filepath.Glob(filepath.Join(r.storageDir, "report-*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list report files: %w", err)
	}
	
	var reports []TestReport
	for _, file := range files {
		report, err := r.loadReport(file)
		if err != nil {
			continue
		}
		
		if report.GeneratedAt.After(start) && report.GeneratedAt.Before(end) {
			reports = append(reports, *report)
		}
	}
	
	// Sort by generation time
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].GeneratedAt.Before(reports[j].GeneratedAt)
	})
	
	return reports, nil
}

// loadReport loads a report from file
func (r *ResultStorage) loadReport(filename string) (*TestReport, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var report TestReport
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&report); err != nil {
		return nil, err
	}
	
	return &report, nil
}

// loadPipelineResult loads a pipeline result from file
func (r *ResultStorage) loadPipelineResult(filename string) (*PipelineResult, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var result PipelineResult
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	
	return &result, nil
}

// CleanupOldResults removes old test results to manage storage
func (r *ResultStorage) CleanupOldResults(retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	
	// Clean up reports
	reportFiles, err := filepath.Glob(filepath.Join(r.storageDir, "report-*.json"))
	if err != nil {
		return fmt.Errorf("failed to list report files: %w", err)
	}
	
	for _, file := range reportFiles {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(file); err != nil {
				log.Printf("Warning: failed to remove old report %s: %v", file, err)
			} else {
				log.Printf("Removed old report: %s", file)
			}
		}
	}
	
	// Clean up pipeline results
	pipelineFiles, err := filepath.Glob(filepath.Join(r.storageDir, "pipeline-*.json"))
	if err != nil {
		return fmt.Errorf("failed to list pipeline files: %w", err)
	}
	
	for _, file := range pipelineFiles {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(file); err != nil {
				log.Printf("Warning: failed to remove old pipeline result %s: %v", file, err)
			} else {
				log.Printf("Removed old pipeline result: %s", file)
			}
		}
	}
	
	return nil
}