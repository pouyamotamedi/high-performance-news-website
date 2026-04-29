package maintenance

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// TestRelationshipManager manages relationships between tests
type TestRelationshipManager struct {
	db *sql.DB
}

// NewTestRelationshipManager creates a new test relationship manager
func NewTestRelationshipManager(db *sql.DB) *TestRelationshipManager {
	return &TestRelationshipManager{
		db: db,
	}
}

// UpdateRelationships updates relationships for a test
func (trm *TestRelationshipManager) UpdateRelationships(testID string, relationships []TestRelation) error {
	tx, err := trm.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing relationships
	_, err = tx.Exec(`
		DELETE FROM test_relationships 
		WHERE source_test_id = $1
	`, testID)
	if err != nil {
		return fmt.Errorf("failed to delete existing relationships: %w", err)
	}

	// Insert new relationships
	for _, relation := range relationships {
		_, err = tx.Exec(`
			INSERT INTO test_relationships (
				source_test_id, target_test_id, relation_type, 
				strength, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6)
		`, testID, relation.TargetTest, string(relation.Type), 
		   relation.Strength, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to insert relationship: %w", err)
		}
	}

	return tx.Commit()
}

// GetRelationships retrieves relationships for a test
func (trm *TestRelationshipManager) GetRelationships(testID string) ([]TestRelation, error) {
	rows, err := trm.db.Query(`
		SELECT target_test_id, relation_type, strength
		FROM test_relationships
		WHERE source_test_id = $1
		ORDER BY strength DESC
	`, testID)
	if err != nil {
		return nil, fmt.Errorf("failed to query relationships: %w", err)
	}
	defer rows.Close()

	var relationships []TestRelation
	for rows.Next() {
		var relation TestRelation
		var relationType string

		err := rows.Scan(&relation.TargetTest, &relationType, &relation.Strength)
		if err != nil {
			return nil, fmt.Errorf("failed to scan relationship: %w", err)
		}

		relation.Type = RelationType(relationType)
		relationships = append(relationships, relation)
	}

	return relationships, rows.Err()
}

// FindSimilarTests finds tests similar to the given test
func (trm *TestRelationshipManager) FindSimilarTests(testID string, threshold float64) ([]TestRelation, error) {
	rows, err := trm.db.Query(`
		SELECT target_test_id, strength
		FROM test_relationships
		WHERE source_test_id = $1 
		AND relation_type = 'similar_to'
		AND strength >= $2
		ORDER BY strength DESC
	`, testID, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to query similar tests: %w", err)
	}
	defer rows.Close()

	var similarTests []TestRelation
	for rows.Next() {
		var relation TestRelation
		err := rows.Scan(&relation.TargetTest, &relation.Strength)
		if err != nil {
			return nil, fmt.Errorf("failed to scan similar test: %w", err)
		}

		relation.Type = RelationSimilarTo
		similarTests = append(similarTests, relation)
	}

	return similarTests, rows.Err()
}

// FindDependentTests finds tests that depend on the given test
func (trm *TestRelationshipManager) FindDependentTests(testID string) ([]TestRelation, error) {
	rows, err := trm.db.Query(`
		SELECT source_test_id, strength
		FROM test_relationships
		WHERE target_test_id = $1 
		AND relation_type = 'depends_on'
		ORDER BY strength DESC
	`, testID)
	if err != nil {
		return nil, fmt.Errorf("failed to query dependent tests: %w", err)
	}
	defer rows.Close()

	var dependentTests []TestRelation
	for rows.Next() {
		var relation TestRelation
		err := rows.Scan(&relation.TargetTest, &relation.Strength)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dependent test: %w", err)
		}

		relation.Type = RelationDependsOn
		dependentTests = append(dependentTests, relation)
	}

	return dependentTests, rows.Err()
}

// AnalyzeImpact analyzes the impact of changing or removing a test
func (trm *TestRelationshipManager) AnalyzeImpact(testID string) (*ImpactAnalysis, error) {
	analysis := &ImpactAnalysis{
		TestID:    testID,
		Timestamp: time.Now(),
	}

	// Find dependent tests
	dependentTests, err := trm.FindDependentTests(testID)
	if err != nil {
		return nil, fmt.Errorf("failed to find dependent tests: %w", err)
	}
	analysis.DependentTests = dependentTests

	// Find similar tests that could be affected
	similarTests, err := trm.FindSimilarTests(testID, 0.5)
	if err != nil {
		return nil, fmt.Errorf("failed to find similar tests: %w", err)
	}
	analysis.SimilarTests = similarTests

	// Calculate impact score
	analysis.ImpactScore = trm.calculateImpactScore(dependentTests, similarTests)

	// Generate recommendations
	analysis.Recommendations = trm.generateImpactRecommendations(analysis)

	return analysis, nil
}

// ImpactAnalysis represents the impact analysis of a test change
type ImpactAnalysis struct {
	TestID          string         `json:"test_id"`
	Timestamp       time.Time      `json:"timestamp"`
	DependentTests  []TestRelation `json:"dependent_tests"`
	SimilarTests    []TestRelation `json:"similar_tests"`
	ImpactScore     float64        `json:"impact_score"`
	Recommendations []string       `json:"recommendations"`
}

// calculateImpactScore calculates an impact score based on relationships
func (trm *TestRelationshipManager) calculateImpactScore(dependent, similar []TestRelation) float64 {
	score := 0.0

	// High impact for dependent tests
	for _, dep := range dependent {
		score += dep.Strength * 2.0
	}

	// Medium impact for similar tests
	for _, sim := range similar {
		score += sim.Strength * 1.0
	}

	// Normalize score (0-10 scale)
	if score > 10 {
		score = 10
	}

	return score
}

// generateImpactRecommendations generates recommendations based on impact analysis
func (trm *TestRelationshipManager) generateImpactRecommendations(analysis *ImpactAnalysis) []string {
	var recommendations []string

	if analysis.ImpactScore > 7 {
		recommendations = append(recommendations, "High impact change - requires careful review and coordination")
	}

	if len(analysis.DependentTests) > 0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("Update %d dependent tests before making changes", len(analysis.DependentTests)))
	}

	if len(analysis.SimilarTests) > 3 {
		recommendations = append(recommendations, 
			"Consider consolidating similar tests to reduce maintenance overhead")
	}

	if analysis.ImpactScore < 2 {
		recommendations = append(recommendations, "Low impact change - safe to proceed")
	}

	return recommendations
}

// UpdateTestMetadata updates test metadata in the database
func (trm *TestRelationshipManager) UpdateTestMetadata(test *TestMetadata) error {
	// Serialize complex fields
	dependenciesJSON, _ := json.Marshal(test.Dependencies)
	tagsJSON, _ := json.Marshal(test.Tags)
	annotationsJSON, _ := json.Marshal(test.Annotations)
	relationshipsJSON, _ := json.Marshal(test.Relationships)

	_, err := trm.db.Exec(`
		INSERT INTO test_metadata (
			test_id, file_path, test_name, test_type, dependencies,
			code_coverage, last_modified, last_executed, execution_count,
			failure_rate, average_runtime_ms, complexity, relationships,
			status, tags, annotations, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		) ON CONFLICT (test_id) DO UPDATE SET
			file_path = EXCLUDED.file_path,
			test_name = EXCLUDED.test_name,
			test_type = EXCLUDED.test_type,
			dependencies = EXCLUDED.dependencies,
			code_coverage = EXCLUDED.code_coverage,
			last_modified = EXCLUDED.last_modified,
			last_executed = EXCLUDED.last_executed,
			execution_count = EXCLUDED.execution_count,
			failure_rate = EXCLUDED.failure_rate,
			average_runtime_ms = EXCLUDED.average_runtime_ms,
			complexity = EXCLUDED.complexity,
			relationships = EXCLUDED.relationships,
			status = EXCLUDED.status,
			tags = EXCLUDED.tags,
			annotations = EXCLUDED.annotations,
			updated_at = EXCLUDED.updated_at
	`, test.ID, test.FilePath, test.TestName, test.TestType, dependenciesJSON,
	   test.CodeCoverage, test.LastModified, test.LastExecuted, test.ExecutionCount,
	   test.FailureRate, test.AverageRuntime.Milliseconds(), test.Complexity, relationshipsJSON,
	   string(test.Status), tagsJSON, annotationsJSON, time.Now(), time.Now())

	if err != nil {
		return fmt.Errorf("failed to update test metadata: %w", err)
	}

	return nil
}

// GetTestMetadata retrieves test metadata from the database
func (trm *TestRelationshipManager) GetTestMetadata(testID string) (*TestMetadata, error) {
	var test TestMetadata
	var dependenciesJSON, tagsJSON, annotationsJSON, relationshipsJSON []byte
	var status string
	var runtimeMs int64

	err := trm.db.QueryRow(`
		SELECT test_id, file_path, test_name, test_type, dependencies,
			   code_coverage, last_modified, last_executed, execution_count,
			   failure_rate, average_runtime_ms, complexity, relationships,
			   status, tags, annotations
		FROM test_metadata
		WHERE test_id = $1
	`, testID).Scan(
		&test.ID, &test.FilePath, &test.TestName, &test.TestType, &dependenciesJSON,
		&test.CodeCoverage, &test.LastModified, &test.LastExecuted, &test.ExecutionCount,
		&test.FailureRate, &runtimeMs, &test.Complexity, &relationshipsJSON,
		&status, &tagsJSON, &annotationsJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("test not found: %s", testID)
		}
		return nil, fmt.Errorf("failed to get test metadata: %w", err)
	}

	// Deserialize complex fields
	json.Unmarshal(dependenciesJSON, &test.Dependencies)
	json.Unmarshal(tagsJSON, &test.Tags)
	json.Unmarshal(annotationsJSON, &test.Annotations)
	json.Unmarshal(relationshipsJSON, &test.Relationships)

	test.Status = TestStatus(status)
	test.AverageRuntime = time.Duration(runtimeMs) * time.Millisecond

	return &test, nil
}

// GetAllTestMetadata retrieves all test metadata
func (trm *TestRelationshipManager) GetAllTestMetadata() (map[string]*TestMetadata, error) {
	rows, err := trm.db.Query(`
		SELECT test_id, file_path, test_name, test_type, dependencies,
			   code_coverage, last_modified, last_executed, execution_count,
			   failure_rate, average_runtime_ms, complexity, relationships,
			   status, tags, annotations
		FROM test_metadata
		ORDER BY test_id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query test metadata: %w", err)
	}
	defer rows.Close()

	tests := make(map[string]*TestMetadata)

	for rows.Next() {
		var test TestMetadata
		var dependenciesJSON, tagsJSON, annotationsJSON, relationshipsJSON []byte
		var status string
		var runtimeMs int64

		err := rows.Scan(
			&test.ID, &test.FilePath, &test.TestName, &test.TestType, &dependenciesJSON,
			&test.CodeCoverage, &test.LastModified, &test.LastExecuted, &test.ExecutionCount,
			&test.FailureRate, &runtimeMs, &test.Complexity, &relationshipsJSON,
			&status, &tagsJSON, &annotationsJSON,
		)
		if err != nil {
			log.Printf("Error scanning test metadata: %v", err)
			continue
		}

		// Deserialize complex fields
		json.Unmarshal(dependenciesJSON, &test.Dependencies)
		json.Unmarshal(tagsJSON, &test.Tags)
		json.Unmarshal(annotationsJSON, &test.Annotations)
		json.Unmarshal(relationshipsJSON, &test.Relationships)

		test.Status = TestStatus(status)
		test.AverageRuntime = time.Duration(runtimeMs) * time.Millisecond

		tests[test.ID] = &test
	}

	return tests, rows.Err()
}

// CreateRelationshipGraph creates a graph representation of test relationships
func (trm *TestRelationshipManager) CreateRelationshipGraph() (*TestRelationshipGraph, error) {
	graph := &TestRelationshipGraph{
		Nodes: make(map[string]*GraphNode),
		Edges: []GraphEdge{},
	}

	// Get all tests
	tests, err := trm.GetAllTestMetadata()
	if err != nil {
		return nil, fmt.Errorf("failed to get test metadata: %w", err)
	}

	// Create nodes
	for testID, test := range tests {
		graph.Nodes[testID] = &GraphNode{
			ID:       testID,
			Label:    test.TestName,
			Type:     test.TestType,
			Status:   string(test.Status),
			Metadata: test,
		}
	}

	// Get all relationships
	rows, err := trm.db.Query(`
		SELECT source_test_id, target_test_id, relation_type, strength
		FROM test_relationships
		ORDER BY strength DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query relationships: %w", err)
	}
	defer rows.Close()

	// Create edges
	for rows.Next() {
		var edge GraphEdge
		var relationType string

		err := rows.Scan(&edge.Source, &edge.Target, &relationType, &edge.Weight)
		if err != nil {
			log.Printf("Error scanning relationship: %v", err)
			continue
		}

		edge.Type = relationType
		graph.Edges = append(graph.Edges, edge)
	}

	return graph, rows.Err()
}

// TestRelationshipGraph represents a graph of test relationships
type TestRelationshipGraph struct {
	Nodes map[string]*GraphNode `json:"nodes"`
	Edges []GraphEdge           `json:"edges"`
}

// GraphNode represents a node in the test relationship graph
type GraphNode struct {
	ID       string        `json:"id"`
	Label    string        `json:"label"`
	Type     string        `json:"type"`
	Status   string        `json:"status"`
	Metadata *TestMetadata `json:"metadata,omitempty"`
}

// GraphEdge represents an edge in the test relationship graph
type GraphEdge struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Type   string  `json:"type"`
	Weight float64 `json:"weight"`
}

// FindClusters finds clusters of related tests
func (trm *TestRelationshipManager) FindClusters(minClusterSize int, minSimilarity float64) ([]TestCluster, error) {
	graph, err := trm.CreateRelationshipGraph()
	if err != nil {
		return nil, fmt.Errorf("failed to create relationship graph: %w", err)
	}

	var clusters []TestCluster
	visited := make(map[string]bool)

	// Simple clustering algorithm based on similarity relationships
	for nodeID := range graph.Nodes {
		if visited[nodeID] {
			continue
		}

		cluster := trm.exploreCluster(nodeID, graph, visited, minSimilarity)
		if len(cluster.TestIDs) >= minClusterSize {
			cluster.ID = fmt.Sprintf("cluster_%d", len(clusters)+1)
			cluster.CreatedAt = time.Now()
			clusters = append(clusters, cluster)
		}
	}

	return clusters, nil
}

// TestCluster represents a cluster of related tests
type TestCluster struct {
	ID          string    `json:"id"`
	TestIDs     []string  `json:"test_ids"`
	Similarity  float64   `json:"similarity"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// exploreCluster explores a cluster starting from a given node
func (trm *TestRelationshipManager) exploreCluster(startNode string, graph *TestRelationshipGraph, visited map[string]bool, minSimilarity float64) TestCluster {
	cluster := TestCluster{
		TestIDs: []string{startNode},
	}
	visited[startNode] = true

	// Find connected nodes with high similarity
	for _, edge := range graph.Edges {
		if edge.Type == string(RelationSimilarTo) && edge.Weight >= minSimilarity {
			var targetNode string
			if edge.Source == startNode && !visited[edge.Target] {
				targetNode = edge.Target
			} else if edge.Target == startNode && !visited[edge.Source] {
				targetNode = edge.Source
			}

			if targetNode != "" {
				visited[targetNode] = true
				cluster.TestIDs = append(cluster.TestIDs, targetNode)
				
				// Recursively explore connected nodes
				subCluster := trm.exploreCluster(targetNode, graph, visited, minSimilarity)
				cluster.TestIDs = append(cluster.TestIDs, subCluster.TestIDs...)
			}
		}
	}

	// Calculate average similarity
	totalSimilarity := 0.0
	count := 0
	for _, edge := range graph.Edges {
		if edge.Type == string(RelationSimilarTo) {
			sourceInCluster := trm.contains(cluster.TestIDs, edge.Source)
			targetInCluster := trm.contains(cluster.TestIDs, edge.Target)
			
			if sourceInCluster && targetInCluster {
				totalSimilarity += edge.Weight
				count++
			}
		}
	}

	if count > 0 {
		cluster.Similarity = totalSimilarity / float64(count)
	}

	return cluster
}

// contains checks if a slice contains a string
func (trm *TestRelationshipManager) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}