package testing

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// ExampleEnvironmentUsage demonstrates how to use the TestEnvironmentManager
func ExampleEnvironmentUsage() {
	// Create a new environment manager
	manager, err := NewTestEnvironmentManager()
	if err != nil {
		log.Fatalf("Failed to create environment manager: %v", err)
	}
	defer manager.Shutdown()

	// Create an isolated environment for integration tests
	env, err := manager.CreateIsolatedEnvironment("integration-tests")
	if err != nil {
		log.Fatalf("Failed to create environment: %v", err)
	}

	fmt.Printf("Created environment %s for test suite: %s\n", env.ID, env.TestSuite)

	// Wait for environment to be ready
	fmt.Println("Waiting for environment to be ready...")
	for {
		currentEnv, err := manager.GetEnvironment(env.ID)
		if err != nil {
			log.Fatalf("Failed to get environment: %v", err)
		}

		if currentEnv.Status == EnvironmentStatusReady {
			fmt.Println("Environment is ready!")
			break
		} else if currentEnv.Status == EnvironmentStatusFailed {
			log.Fatalf("Environment creation failed: %s", currentEnv.ErrorMessage)
		}

		time.Sleep(5 * time.Second)
	}

	// Use the environment for testing
	if err := runTestsWithEnvironment(env); err != nil {
		log.Printf("Tests failed: %v", err)
	}

	// List all environments
	environments := manager.ListEnvironments()
	fmt.Printf("Total environments: %d\n", len(environments))

	// Check resource utilization
	memUtil, cpuUtil, envUtil := manager.resourcePool.GetUtilization()
	fmt.Printf("Resource utilization - Memory: %.2f%%, CPU: %.2f%%, Environments: %.2f%%\n",
		memUtil*100, cpuUtil*100, envUtil*100)

	// Cleanup the environment
	if err := manager.CleanupEnvironment(env.ID); err != nil {
		log.Printf("Failed to cleanup environment: %v", err)
	} else {
		fmt.Printf("Environment %s cleaned up successfully\n", env.ID)
	}
}

// runTestsWithEnvironment demonstrates running tests with an isolated environment
func runTestsWithEnvironment(env *IsolatedEnvironment) error {
	// Connect to the isolated database
	db, err := sql.Open("postgres", env.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Test database connectivity
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	fmt.Println("Successfully connected to isolated database")

	// Create test schema
	schema := `
		CREATE TABLE IF NOT EXISTS articles (
			id SERIAL PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			author_id INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(100) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	fmt.Println("Created test schema")

	// Insert test data
	if err := insertTestData(db); err != nil {
		return fmt.Errorf("failed to insert test data: %w", err)
	}

	// Run test queries
	if err := runTestQueries(db); err != nil {
		return fmt.Errorf("failed to run test queries: %w", err)
	}

	fmt.Println("All tests passed successfully")
	return nil
}

// insertTestData inserts sample data for testing
func insertTestData(db *sql.DB) error {
	// Insert test users
	users := []struct {
		username, email string
	}{
		{"testuser1", "test1@example.com"},
		{"testuser2", "test2@example.com"},
		{"testuser3", "test3@example.com"},
	}

	for _, user := range users {
		_, err := db.Exec("INSERT INTO users (username, email) VALUES ($1, $2)", user.username, user.email)
		if err != nil {
			return fmt.Errorf("failed to insert user %s: %w", user.username, err)
		}
	}

	// Insert test articles
	articles := []struct {
		title, content string
		authorID       int
	}{
		{"Test Article 1", "This is the content of test article 1", 1},
		{"Test Article 2", "This is the content of test article 2", 2},
		{"Test Article 3", "This is the content of test article 3", 1},
	}

	for _, article := range articles {
		_, err := db.Exec("INSERT INTO articles (title, content, author_id) VALUES ($1, $2, $3)",
			article.title, article.content, article.authorID)
		if err != nil {
			return fmt.Errorf("failed to insert article %s: %w", article.title, err)
		}
	}

	fmt.Println("Inserted test data successfully")
	return nil
}

// runTestQueries runs various test queries to verify functionality
func runTestQueries(db *sql.DB) error {
	// Test 1: Count users
	var userCount int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}
	fmt.Printf("User count: %d\n", userCount)

	// Test 2: Count articles
	var articleCount int
	err = db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&articleCount)
	if err != nil {
		return fmt.Errorf("failed to count articles: %w", err)
	}
	fmt.Printf("Article count: %d\n", articleCount)

	// Test 3: Join query
	query := `
		SELECT a.title, u.username 
		FROM articles a 
		JOIN users u ON a.author_id = u.id 
		ORDER BY a.id
	`
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to execute join query: %w", err)
	}
	defer rows.Close()

	fmt.Println("Articles with authors:")
	for rows.Next() {
		var title, username string
		if err := rows.Scan(&title, &username); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}
		fmt.Printf("  - %s by %s\n", title, username)
	}

	// Test 4: Transaction test
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	_, err = tx.Exec("INSERT INTO users (username, email) VALUES ($1, $2)", "txuser", "tx@example.com")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to insert in transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Println("Transaction test passed")
	return nil
}

// ExampleParallelEnvironments demonstrates creating multiple environments in parallel
func ExampleParallelEnvironments() {
	manager, err := NewTestEnvironmentManager()
	if err != nil {
		log.Fatalf("Failed to create environment manager: %v", err)
	}
	defer manager.Shutdown()

	// Create multiple environments concurrently
	testSuites := []string{"unit-tests", "integration-tests", "performance-tests"}
	environments := make([]*IsolatedEnvironment, len(testSuites))

	// Create environments
	for i, suite := range testSuites {
		env, err := manager.CreateIsolatedEnvironment(suite)
		if err != nil {
			log.Fatalf("Failed to create environment for %s: %v", suite, err)
		}
		environments[i] = env
		fmt.Printf("Created environment %s for %s\n", env.ID, suite)
	}

	// Wait for all environments to be ready
	fmt.Println("Waiting for all environments to be ready...")
	for _, env := range environments {
		for {
			currentEnv, err := manager.GetEnvironment(env.ID)
			if err != nil {
				log.Printf("Failed to get environment %s: %v", env.ID, err)
				break
			}

			if currentEnv.Status == EnvironmentStatusReady {
				fmt.Printf("Environment %s (%s) is ready\n", env.ID, env.TestSuite)
				break
			} else if currentEnv.Status == EnvironmentStatusFailed {
				log.Printf("Environment %s (%s) failed: %s", env.ID, env.TestSuite, currentEnv.ErrorMessage)
				break
			}

			time.Sleep(2 * time.Second)
		}
	}

	// Run tests in parallel (simulated)
	fmt.Println("Running tests in parallel...")
	time.Sleep(10 * time.Second) // Simulate test execution

	// Cleanup all environments
	fmt.Println("Cleaning up environments...")
	for _, env := range environments {
		if err := manager.CleanupEnvironment(env.ID); err != nil {
			log.Printf("Failed to cleanup environment %s: %v", env.ID, err)
		} else {
			fmt.Printf("Cleaned up environment %s\n", env.ID)
		}
	}

	fmt.Println("All environments cleaned up successfully")
}

// ExampleHealthMonitoring demonstrates health monitoring functionality
func ExampleHealthMonitoring() {
	manager, err := NewTestEnvironmentManager()
	if err != nil {
		log.Fatalf("Failed to create environment manager: %v", err)
	}
	defer manager.Shutdown()

	// Create an environment
	env, err := manager.CreateIsolatedEnvironment("health-monitoring-test")
	if err != nil {
		log.Fatalf("Failed to create environment: %v", err)
	}

	// Wait for environment to be ready
	for {
		currentEnv, err := manager.GetEnvironment(env.ID)
		if err != nil {
			log.Fatalf("Failed to get environment: %v", err)
		}

		if currentEnv.Status == EnvironmentStatusReady {
			break
		} else if currentEnv.Status == EnvironmentStatusFailed {
			log.Fatalf("Environment creation failed: %s", currentEnv.ErrorMessage)
		}

		time.Sleep(5 * time.Second)
	}

	fmt.Printf("Environment %s is ready, monitoring health...\n", env.ID)

	// Monitor health for a while
	for i := 0; i < 6; i++ { // Monitor for ~3 minutes
		time.Sleep(30 * time.Second)

		currentEnv, err := manager.GetEnvironment(env.ID)
		if err != nil {
			log.Printf("Failed to get environment: %v", err)
			continue
		}

		fmt.Printf("Health check %d: Status=%s, LastCheck=%s\n",
			i+1, currentEnv.HealthStatus, currentEnv.LastHealthCheck.Format(time.RFC3339))
	}

	// Cleanup
	if err := manager.CleanupEnvironment(env.ID); err != nil {
		log.Printf("Failed to cleanup environment: %v", err)
	} else {
		fmt.Printf("Environment %s cleaned up\n", env.ID)
	}
}