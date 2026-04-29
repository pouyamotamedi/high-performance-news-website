package testing

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// RelationshipManager manages relationships between test data entities
type RelationshipManager struct {
	relationships map[string][]Relationship
	constraints   []RelationshipConstraint
	mutex         sync.RWMutex
}

// Relationship represents a relationship between entities
type Relationship struct {
	ID           string                 `json:"id"`
	SourceID     uint64                 `json:"source_id"`
	TargetID     uint64                 `json:"target_id"`
	Type         RelationshipType       `json:"type"`
	Strength     float64                `json:"strength"` // 0.0 to 1.0
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	LanguageCode string                 `json:"language_code"`
}

// RelationshipType defines the type of relationship
type RelationshipType string

const (
	RelationshipRelated     RelationshipType = "related"
	RelationshipSimilar     RelationshipType = "similar"
	RelationshipTranslation RelationshipType = "translation"
	RelationshipParent      RelationshipType = "parent"
	RelationshipChild       RelationshipType = "child"
	RelationshipReference   RelationshipType = "reference"
	RelationshipDuplicate   RelationshipType = "duplicate"
	RelationshipUpdate      RelationshipType = "update"
)

// RelationshipConstraint defines constraints for relationships
type RelationshipConstraint struct {
	Name        string           `json:"name"`
	Type        RelationshipType `json:"type"`
	MaxCount    int              `json:"max_count"`
	MinStrength float64          `json:"min_strength"`
	Required    bool             `json:"required"`
	Enabled     bool             `json:"enabled"`
}

// NewRelationshipManager creates a new relationship manager
func NewRelationshipManager() *RelationshipManager {
	manager := &RelationshipManager{
		relationships: make(map[string][]Relationship),
		constraints:   make([]RelationshipConstraint, 0),
	}

	manager.initializeDefaultConstraints()
	return manager
}

// initializeDefaultConstraints sets up default relationship constraints
func (rm *RelationshipManager) initializeDefaultConstraints() {
	defaultConstraints := []RelationshipConstraint{
		{
			Name:        "max_related_articles",
			Type:        RelationshipRelated,
			MaxCount:    5,
			MinStrength: 0.3,
			Required:    false,
			Enabled:     true,
		},
		{
			Name:        "translation_group_limit",
			Type:        RelationshipTranslation,
			MaxCount:    10, // Max 10 languages per translation group
			MinStrength: 0.8,
			Required:    false,
			Enabled:     true,
		},
		{
			Name:        "parent_child_hierarchy",
			Type:        RelationshipParent,
			MaxCount:    1, // Only one parent
			MinStrength: 0.9,
			Required:    false,
			Enabled:     true,
		},
		{
			Name:        "duplicate_detection",
			Type:        RelationshipDuplicate,
			MaxCount:    3,
			MinStrength: 0.7,
			Required:    false,
			Enabled:     true,
		},
	}

	rm.constraints = defaultConstraints
}

// GenerateRelationships generates relationships for an entity
func (rm *RelationshipManager) GenerateRelationships(entityID uint64, languageCode string) map[string]interface{} {
	relationships := make(map[string]interface{})

	// Generate related articles (20% chance)
	if rm.randomFloat() < 0.2 {
		relatedCount := 1 + rm.randomInt(3) // 1-3 related articles
		relatedIDs := make([]uint64, relatedCount)
		for i := 0; i < relatedCount; i++ {
			relatedIDs[i] = rm.generateRelatedEntityID(entityID)
		}
		relationships["related"] = relatedIDs

		// Create relationship records
		for _, relatedID := range relatedIDs {
			rm.AddRelationship(entityID, relatedID, RelationshipRelated, 0.5+rm.randomFloat()*0.4, languageCode)
		}
	}

	// Generate similar articles (15% chance)
	if rm.randomFloat() < 0.15 {
		similarCount := 1 + rm.randomInt(2) // 1-2 similar articles
		similarIDs := make([]uint64, similarCount)
		for i := 0; i < similarCount; i++ {
			similarIDs[i] = rm.generateRelatedEntityID(entityID)
		}
		relationships["similar"] = similarIDs

		// Create relationship records
		for _, similarID := range similarIDs {
			rm.AddRelationship(entityID, similarID, RelationshipSimilar, 0.6+rm.randomFloat()*0.3, languageCode)
		}
	}

	// Generate references (10% chance)
	if rm.randomFloat() < 0.1 {
		referenceCount := 1 + rm.randomInt(2) // 1-2 references
		referenceIDs := make([]uint64, referenceCount)
		for i := 0; i < referenceCount; i++ {
			referenceIDs[i] = rm.generateRelatedEntityID(entityID)
		}
		relationships["references"] = referenceIDs

		// Create relationship records
		for _, refID := range referenceIDs {
			rm.AddRelationship(entityID, refID, RelationshipReference, 0.4+rm.randomFloat()*0.3, languageCode)
		}
	}

	// Generate update relationships (5% chance)
	if rm.randomFloat() < 0.05 {
		updateID := rm.generateRelatedEntityID(entityID)
		relationships["updates"] = updateID
		rm.AddRelationship(entityID, updateID, RelationshipUpdate, 0.8+rm.randomFloat()*0.2, languageCode)
	}

	return relationships
}

// AddRelationship adds a new relationship
func (rm *RelationshipManager) AddRelationship(sourceID, targetID uint64, relType RelationshipType, strength float64, languageCode string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	// Validate constraints
	if err := rm.validateRelationshipConstraints(sourceID, relType); err != nil {
		return err
	}

	relationship := Relationship{
		ID:           fmt.Sprintf("rel_%d_%d_%d", sourceID, targetID, time.Now().Unix()),
		SourceID:     sourceID,
		TargetID:     targetID,
		Type:         relType,
		Strength:     strength,
		Metadata:     make(map[string]interface{}),
		CreatedAt:    time.Now(),
		LanguageCode: languageCode,
	}

	// Add metadata based on relationship type
	switch relType {
	case RelationshipTranslation:
		relationship.Metadata["translation_quality"] = 0.8 + rm.randomFloat()*0.2
	case RelationshipSimilar:
		relationship.Metadata["similarity_score"] = strength
	case RelationshipRelated:
		relationship.Metadata["relevance_score"] = strength
	}

	key := fmt.Sprintf("%d", sourceID)
	rm.relationships[key] = append(rm.relationships[key], relationship)

	return nil
}

// GetRelationships returns all relationships for an entity
func (rm *RelationshipManager) GetRelationships(entityID uint64) []Relationship {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	key := fmt.Sprintf("%d", entityID)
	relationships, exists := rm.relationships[key]
	if !exists {
		return make([]Relationship, 0)
	}

	// Return a copy to prevent external modification
	result := make([]Relationship, len(relationships))
	copy(result, relationships)
	return result
}

// GetRelationshipsByType returns relationships of a specific type for an entity
func (rm *RelationshipManager) GetRelationshipsByType(entityID uint64, relType RelationshipType) []Relationship {
	allRelationships := rm.GetRelationships(entityID)
	var filtered []Relationship

	for _, rel := range allRelationships {
		if rel.Type == relType {
			filtered = append(filtered, rel)
		}
	}

	return filtered
}

// RemoveRelationship removes a specific relationship
func (rm *RelationshipManager) RemoveRelationship(relationshipID string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	for key, relationships := range rm.relationships {
		for i, rel := range relationships {
			if rel.ID == relationshipID {
				// Remove relationship from slice
				rm.relationships[key] = append(relationships[:i], relationships[i+1:]...)
				log.Printf("Removed relationship: %s", relationshipID)
				return nil
			}
		}
	}

	return fmt.Errorf("relationship %s not found", relationshipID)
}

// UpdateRelationshipStrength updates the strength of a relationship
func (rm *RelationshipManager) UpdateRelationshipStrength(relationshipID string, newStrength float64) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	for key, relationships := range rm.relationships {
		for i, rel := range relationships {
			if rel.ID == relationshipID {
				rm.relationships[key][i].Strength = newStrength
				log.Printf("Updated relationship %s strength to %.2f", relationshipID, newStrength)
				return nil
			}
		}
	}

	return fmt.Errorf("relationship %s not found", relationshipID)
}

// validateRelationshipConstraints validates relationship constraints
func (rm *RelationshipManager) validateRelationshipConstraints(entityID uint64, relType RelationshipType) error {
	for _, constraint := range rm.constraints {
		if !constraint.Enabled || constraint.Type != relType {
			continue
		}

		existingRelationships := rm.GetRelationshipsByType(entityID, relType)
		
		if len(existingRelationships) >= constraint.MaxCount {
			return fmt.Errorf("maximum relationship count (%d) exceeded for type %s", 
				constraint.MaxCount, relType)
		}
	}

	return nil
}

// GenerateTranslationRelationships generates translation relationships between articles
func (rm *RelationshipManager) GenerateTranslationRelationships(translationGroupID uint64, articleIDs []uint64, languages []string) error {
	if len(articleIDs) != len(languages) {
		return fmt.Errorf("article IDs and languages count mismatch")
	}

	// Create bidirectional translation relationships
	for i, sourceID := range articleIDs {
		for j, targetID := range articleIDs {
			if i != j {
				strength := 0.9 + rm.randomFloat()*0.1 // High strength for translations
				err := rm.AddRelationship(sourceID, targetID, RelationshipTranslation, strength, languages[i])
				if err != nil {
					log.Printf("Warning: failed to create translation relationship: %v", err)
				}
			}
		}
	}

	return nil
}

// AnalyzeRelationshipPatterns analyzes patterns in relationships
func (rm *RelationshipManager) AnalyzeRelationshipPatterns() RelationshipAnalysis {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	analysis := RelationshipAnalysis{
		TotalRelationships: 0,
		TypeDistribution:   make(map[RelationshipType]int),
		StrengthDistribution: make(map[string]int),
		LanguageDistribution: make(map[string]int),
		AverageStrength:    0.0,
	}

	var totalStrength float64

	for _, relationships := range rm.relationships {
		for _, rel := range relationships {
			analysis.TotalRelationships++
			analysis.TypeDistribution[rel.Type]++
			analysis.LanguageDistribution[rel.LanguageCode]++
			totalStrength += rel.Strength

			// Categorize strength
			switch {
			case rel.Strength < 0.3:
				analysis.StrengthDistribution["weak"]++
			case rel.Strength < 0.7:
				analysis.StrengthDistribution["medium"]++
			default:
				analysis.StrengthDistribution["strong"]++
			}
		}
	}

	if analysis.TotalRelationships > 0 {
		analysis.AverageStrength = totalStrength / float64(analysis.TotalRelationships)
	}

	return analysis
}

// RelationshipAnalysis represents analysis of relationship patterns
type RelationshipAnalysis struct {
	TotalRelationships   int                        `json:"total_relationships"`
	TypeDistribution     map[RelationshipType]int   `json:"type_distribution"`
	StrengthDistribution map[string]int             `json:"strength_distribution"`
	LanguageDistribution map[string]int             `json:"language_distribution"`
	AverageStrength      float64                    `json:"average_strength"`
}

// OptimizeRelationships optimizes relationships by removing weak or redundant ones
func (rm *RelationshipManager) OptimizeRelationships(minStrength float64) int {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	removedCount := 0

	for key, relationships := range rm.relationships {
		var optimized []Relationship

		for _, rel := range relationships {
			// Keep relationships above minimum strength
			if rel.Strength >= minStrength {
				optimized = append(optimized, rel)
			} else {
				removedCount++
			}
		}

		rm.relationships[key] = optimized
	}

	log.Printf("Optimized relationships: removed %d weak relationships", removedCount)
	return removedCount
}

// FindSimilarEntities finds entities with similar relationship patterns
func (rm *RelationshipManager) FindSimilarEntities(entityID uint64, threshold float64) []uint64 {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	sourceRelationships := rm.GetRelationships(entityID)
	if len(sourceRelationships) == 0 {
		return make([]uint64, 0)
	}

	var similarEntities []uint64

	for key, relationships := range rm.relationships {
		if key == fmt.Sprintf("%d", entityID) {
			continue
		}

		similarity := rm.calculateRelationshipSimilarity(sourceRelationships, relationships)
		if similarity >= threshold {
			// Parse entity ID from key
			var otherEntityID uint64
			fmt.Sscanf(key, "%d", &otherEntityID)
			similarEntities = append(similarEntities, otherEntityID)
		}
	}

	return similarEntities
}

// calculateRelationshipSimilarity calculates similarity between two sets of relationships
func (rm *RelationshipManager) calculateRelationshipSimilarity(rel1, rel2 []Relationship) float64 {
	if len(rel1) == 0 && len(rel2) == 0 {
		return 1.0
	}

	if len(rel1) == 0 || len(rel2) == 0 {
		return 0.0
	}

	// Create type frequency maps
	types1 := make(map[RelationshipType]int)
	types2 := make(map[RelationshipType]int)

	for _, rel := range rel1 {
		types1[rel.Type]++
	}

	for _, rel := range rel2 {
		types2[rel.Type]++
	}

	// Calculate Jaccard similarity
	intersection := 0
	union := 0

	allTypes := make(map[RelationshipType]bool)
	for t := range types1 {
		allTypes[t] = true
	}
	for t := range types2 {
		allTypes[t] = true
	}

	for relType := range allTypes {
		count1 := types1[relType]
		count2 := types2[relType]

		if count1 > 0 && count2 > 0 {
			intersection += min(count1, count2)
		}
		union += max(count1, count2)
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// Helper functions
func (rm *RelationshipManager) generateRelatedEntityID(baseID uint64) uint64 {
	// Generate a related entity ID (in practice, this would query existing entities)
	offset := uint64(1 + rm.randomInt(1000))
	return baseID + offset
}

func (rm *RelationshipManager) randomInt(max int) int {
	if max <= 0 {
		return 0
	}
	return rand.Intn(max)
}

func (rm *RelationshipManager) randomFloat() float64 {
	return rand.Float64()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// AddConstraint adds a new relationship constraint
func (rm *RelationshipManager) AddConstraint(constraint RelationshipConstraint) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	rm.constraints = append(rm.constraints, constraint)
	log.Printf("Added relationship constraint: %s", constraint.Name)
}

// GetConstraints returns all relationship constraints
func (rm *RelationshipManager) GetConstraints() []RelationshipConstraint {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	constraints := make([]RelationshipConstraint, len(rm.constraints))
	copy(constraints, rm.constraints)
	return constraints
}

// EnableConstraint enables or disables a constraint
func (rm *RelationshipManager) EnableConstraint(name string, enabled bool) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	for i, constraint := range rm.constraints {
		if constraint.Name == name {
			rm.constraints[i].Enabled = enabled
			log.Printf("Constraint '%s' enabled: %t", name, enabled)
			return nil
		}
	}

	return fmt.Errorf("constraint '%s' not found", name)
}

// GetRelationshipStats returns statistics about relationships
func (rm *RelationshipManager) GetRelationshipStats() map[string]interface{} {
	analysis := rm.AnalyzeRelationshipPatterns()

	return map[string]interface{}{
		"total_relationships":    analysis.TotalRelationships,
		"type_distribution":      analysis.TypeDistribution,
		"strength_distribution":  analysis.StrengthDistribution,
		"language_distribution":  analysis.LanguageDistribution,
		"average_strength":       analysis.AverageStrength,
		"total_entities":         len(rm.relationships),
	}
}