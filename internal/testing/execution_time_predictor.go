package testing

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"time"
)

// ExecutionTimePredictor predicts test execution times
type ExecutionTimePredictor struct {
	historicalData map[string][]ExecutionRecord
	modelCache     map[string]*PredictionModel
}

// PredictionModel represents a prediction model for a test
type PredictionModel struct {
	TestID          string        `json:"test_id"`
	BaseTime        time.Duration `json:"base_time"`
	Variance        float64       `json:"variance"`
	Trend           float64       `json:"trend"`
	Confidence      float64       `json:"confidence"`
	LastUpdated     time.Time     `json:"last_updated"`
	SampleSize      int           `json:"sample_size"`
}

// TimePrediction represents a time prediction result
type TimePrediction struct {
	EstimatedTime   time.Duration `json:"estimated_time"`
	MinTime         time.Duration `json:"min_time"`
	MaxTime         time.Duration `json:"max_time"`
	Confidence      float64       `json:"confidence"`
	Factors         []PredictionFactor `json:"factors"`
	Model           *PredictionModel `json:"model"`
}

// PredictionFactor represents a factor affecting prediction
type PredictionFactor struct {
	Name        string  `json:"name"`
	Impact      float64 `json:"impact"`
	Description string  `json:"description"`
}

// NewExecutionTimePredictor creates a new execution time predictor
func NewExecutionTimePredictor() *ExecutionTimePredictor {
	return &ExecutionTimePredictor{
		historicalData: make(map[string][]ExecutionRecord),
		modelCache:     make(map[string]*PredictionModel),
	}
}

// PredictExecutionTime predicts the execution time for a test
func (p *ExecutionTimePredictor) PredictExecutionTime(ctx context.Context, test *TestCase) (*TimePrediction, error) {
	log.Printf("Predicting execution time for test %s", test.ID)
	
	// Get or create prediction model
	model, err := p.getOrCreateModel(test.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get prediction model: %w", err)
	}
	
	// Calculate base prediction
	basePrediction := model.BaseTime
	
	// Apply prediction factors
	factors := p.calculatePredictionFactors(test, model)
	adjustedTime := p.applyFactors(basePrediction, factors)
	
	// Calculate confidence intervals
	minTime, maxTime := p.calculateConfidenceInterval(adjustedTime, model.Variance, model.Confidence)
	
	prediction := &TimePrediction{
		EstimatedTime: adjustedTime,
		MinTime:       minTime,
		MaxTime:       maxTime,
		Confidence:    model.Confidence,
		Factors:       factors,
		Model:         model,
	}
	
	log.Printf("Predicted time for test %s: %v (confidence: %.2f)", test.ID, adjustedTime, model.Confidence)
	return prediction, nil
}

// getOrCreateModel gets or creates a prediction model for a test
func (p *ExecutionTimePredictor) getOrCreateModel(testID string) (*PredictionModel, error) {
	// Check cache first
	if model, exists := p.modelCache[testID]; exists {
		// Update model if it's stale
		if time.Since(model.LastUpdated) > 24*time.Hour {
			return p.updateModel(testID)
		}
		return model, nil
	}
	
	// Create new model
	return p.createModel(testID)
}

// createModel creates a new prediction model for a test
func (p *ExecutionTimePredictor) createModel(testID string) (*PredictionModel, error) {
	history := p.historicalData[testID]
	
	if len(history) == 0 {
		// No historical data, use default estimates
		return &PredictionModel{
			TestID:      testID,
			BaseTime:    30 * time.Second, // Default estimate
			Variance:    0.3,              // 30% variance
			Trend:       0.0,              // No trend
			Confidence:  0.5,              // Low confidence
			LastUpdated: time.Now(),
			SampleSize:  0,
		}, nil
	}
	
	// Calculate statistics from historical data
	durations := make([]float64, len(history))
	for i, record := range history {
		durations[i] = float64(record.Duration.Nanoseconds())
	}
	
	mean := p.calculateMean(durations)
	variance := p.calculateVariance(durations, mean)
	trend := p.calculateTrend(history)
	confidence := p.calculateConfidence(len(history), variance)
	
	model := &PredictionModel{
		TestID:      testID,
		BaseTime:    time.Duration(int64(mean)),
		Variance:    variance,
		Trend:       trend,
		Confidence:  confidence,
		LastUpdated: time.Now(),
		SampleSize:  len(history),
	}
	
	p.modelCache[testID] = model
	return model, nil
}

// updateModel updates an existing prediction model
func (p *ExecutionTimePredictor) updateModel(testID string) (*PredictionModel, error) {
	// Remove from cache and recreate
	delete(p.modelCache, testID)
	return p.createModel(testID)
}

// calculatePredictionFactors calculates factors that affect prediction
func (p *ExecutionTimePredictor) calculatePredictionFactors(test *TestCase, model *PredictionModel) []PredictionFactor {
	var factors []PredictionFactor
	
	// Resource usage factor
	if test.ResourceUsage.CPU > 70 {
		factors = append(factors, PredictionFactor{
			Name:        "high_cpu_usage",
			Impact:      1.2, // 20% slower
			Description: "High CPU usage may slow down execution",
		})
	}
	
	if test.ResourceUsage.Memory > 2*1024*1024*1024 { // 2GB
		factors = append(factors, PredictionFactor{
			Name:        "high_memory_usage",
			Impact:      1.15, // 15% slower
			Description: "High memory usage may cause GC pressure",
		})
	}
	
	// Network factor
	if test.ResourceUsage.Network {
		factors = append(factors, PredictionFactor{
			Name:        "network_dependency",
			Impact:      1.3, // 30% slower due to network latency
			Description: "Network dependencies add latency and variability",
		})
	}
	
	// Database factor
	if p.usesDatabase(test) {
		factors = append(factors, PredictionFactor{
			Name:        "database_dependency",
			Impact:      1.25, // 25% slower
			Description: "Database operations add execution time",
		})
	}
	
	// Parallel execution factor
	if p.canRunInParallel(test) {
		factors = append(factors, PredictionFactor{
			Name:        "parallel_execution",
			Impact:      0.8, // 20% faster when run in parallel
			Description: "Test can benefit from parallel execution",
		})
	}
	
	// Flakiness factor
	if test.Flakiness > 0.1 {
		retryFactor := 1.0 + test.Flakiness*2 // Assume retries due to flakiness
		factors = append(factors, PredictionFactor{
			Name:        "flakiness_retries",
			Impact:      retryFactor,
			Description: fmt.Sprintf("Flaky test may require retries (%.1f%% failure rate)", test.Flakiness*100),
		})
	}
	
	// Time of day factor (if we have historical data showing patterns)
	timeOfDayFactor := p.calculateTimeOfDayFactor(test.ID)
	if timeOfDayFactor != 1.0 {
		factors = append(factors, PredictionFactor{
			Name:        "time_of_day",
			Impact:      timeOfDayFactor,
			Description: "Historical data shows time-of-day performance variation",
		})
	}
	
	return factors
}

// applyFactors applies prediction factors to base time
func (p *ExecutionTimePredictor) applyFactors(baseTime time.Duration, factors []PredictionFactor) time.Duration {
	multiplier := 1.0
	
	for _, factor := range factors {
		multiplier *= factor.Impact
	}
	
	return time.Duration(float64(baseTime.Nanoseconds()) * multiplier)
}

// calculateConfidenceInterval calculates confidence interval for prediction
func (p *ExecutionTimePredictor) calculateConfidenceInterval(estimatedTime time.Duration, variance, confidence float64) (time.Duration, time.Duration) {
	// Calculate standard deviation
	stdDev := math.Sqrt(variance)
	
	// Use confidence level to determine interval
	confidenceMultiplier := 1.96 // 95% confidence interval
	if confidence < 0.7 {
		confidenceMultiplier = 2.58 // 99% confidence interval for low confidence
	}
	
	deviation := time.Duration(float64(estimatedTime.Nanoseconds()) * stdDev * confidenceMultiplier)
	
	minTime := estimatedTime - deviation
	if minTime < 0 {
		minTime = estimatedTime / 4 // Minimum 25% of estimated time
	}
	
	maxTime := estimatedTime + deviation
	
	return minTime, maxTime
}

// Statistical calculation functions
func (p *ExecutionTimePredictor) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	
	return sum / float64(len(values))
}

func (p *ExecutionTimePredictor) calculateVariance(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0.3 // Default variance
	}
	
	sumSquaredDiff := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	
	variance := sumSquaredDiff / float64(len(values)-1)
	
	// Normalize variance (coefficient of variation)
	if mean > 0 {
		return math.Sqrt(variance) / mean
	}
	
	return 0.3
}

func (p *ExecutionTimePredictor) calculateTrend(history []ExecutionRecord) float64 {
	if len(history) < 3 {
		return 0.0 // Need at least 3 points for trend
	}
	
	// Sort by timestamp
	sort.Slice(history, func(i, j int) bool {
		return history[i].Timestamp.Before(history[j].Timestamp)
	})
	
	// Simple linear regression to find trend
	n := float64(len(history))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0
	
	for i, record := range history {
		x := float64(i)
		y := float64(record.Duration.Nanoseconds())
		
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	
	// Calculate slope (trend)
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0.0
	}
	
	slope := (n*sumXY - sumX*sumY) / denominator
	
	// Normalize slope relative to mean duration
	meanDuration := sumY / n
	if meanDuration > 0 {
		return slope / meanDuration
	}
	
	return 0.0
}

func (p *ExecutionTimePredictor) calculateConfidence(sampleSize int, variance float64) float64 {
	// Confidence increases with sample size and decreases with variance
	sizeConfidence := math.Min(float64(sampleSize)/50.0, 1.0) // Max confidence at 50+ samples
	varianceConfidence := math.Max(0.1, 1.0-variance)         // Lower confidence for high variance
	
	return (sizeConfidence + varianceConfidence) / 2.0
}

// Helper functions
func (p *ExecutionTimePredictor) usesDatabase(test *TestCase) bool {
	for _, tag := range test.Tags {
		if tag == "database" || tag == "integration" || tag == "repository" {
			return true
		}
	}
	return false
}

func (p *ExecutionTimePredictor) canRunInParallel(test *TestCase) bool {
	// Tests that don't use shared resources can run in parallel
	for _, tag := range test.Tags {
		if tag == "sequential" || tag == "exclusive" || tag == "migration" {
			return false
		}
	}
	return true
}

func (p *ExecutionTimePredictor) calculateTimeOfDayFactor(testID string) float64 {
	history := p.historicalData[testID]
	if len(history) < 10 {
		return 1.0 // Need sufficient data
	}
	
	// Group by hour of day and calculate average performance
	hourlyPerformance := make(map[int][]time.Duration)
	
	for _, record := range history {
		hour := record.Timestamp.Hour()
		hourlyPerformance[hour] = append(hourlyPerformance[hour], record.Duration)
	}
	
	// Calculate current hour factor
	currentHour := time.Now().Hour()
	if durations, exists := hourlyPerformance[currentHour]; exists && len(durations) > 0 {
		// Calculate average for current hour
		var sum time.Duration
		for _, d := range durations {
			sum += d
		}
		currentHourAvg := sum / time.Duration(len(durations))
		
		// Calculate overall average
		var totalSum time.Duration
		totalCount := 0
		for _, hourDurations := range hourlyPerformance {
			for _, d := range hourDurations {
				totalSum += d
				totalCount++
			}
		}
		
		if totalCount > 0 {
			overallAvg := totalSum / time.Duration(totalCount)
			if overallAvg > 0 {
				return float64(currentHourAvg) / float64(overallAvg)
			}
		}
	}
	
	return 1.0
}

// AddExecutionRecord adds an execution record for model training
func (p *ExecutionTimePredictor) AddExecutionRecord(record ExecutionRecord) {
	p.historicalData[record.TestID] = append(p.historicalData[record.TestID], record)
	
	// Keep only recent records (last 100 executions)
	if len(p.historicalData[record.TestID]) > 100 {
		p.historicalData[record.TestID] = p.historicalData[record.TestID][1:]
	}
	
	// Invalidate cached model
	delete(p.modelCache, record.TestID)
}

// GetPredictionAccuracy calculates prediction accuracy for a test
func (p *ExecutionTimePredictor) GetPredictionAccuracy(testID string) float64 {
	history := p.historicalData[testID]
	if len(history) < 5 {
		return 0.0 // Need sufficient data
	}
	
	// Use last 20% of data for validation
	validationSize := len(history) / 5
	if validationSize < 2 {
		validationSize = 2
	}
	
	trainingData := history[:len(history)-validationSize]
	validationData := history[len(history)-validationSize:]
	
	// Temporarily use training data to create model
	originalData := p.historicalData[testID]
	p.historicalData[testID] = trainingData
	
	model, err := p.createModel(testID)
	if err != nil {
		p.historicalData[testID] = originalData
		return 0.0
	}
	
	// Calculate accuracy on validation data
	totalError := 0.0
	for _, record := range validationData {
		predicted := model.BaseTime
		actual := record.Duration
		
		error := math.Abs(float64(predicted-actual)) / float64(actual)
		totalError += error
	}
	
	// Restore original data
	p.historicalData[testID] = originalData
	delete(p.modelCache, testID)
	
	accuracy := 1.0 - (totalError / float64(len(validationData)))
	if accuracy < 0 {
		accuracy = 0
	}
	
	return accuracy
}