package testing

import (
	"go/ast"
	"go/token"
)

// createMutators returns all available mutators
func createMutators() []Mutator {
	return []Mutator{
		&ConditionalBoundaryMutator{},
		&ArithmeticOperatorMutator{},
		&LogicalOperatorMutator{},
		&ReturnValueMutator{},
		&NullCheckMutator{},
		&SecurityMutator{},
		&PerformanceMutator{},
	}
}

// ConditionalBoundaryMutator mutates conditional boundary operators
type ConditionalBoundaryMutator struct{}

func (m *ConditionalBoundaryMutator) Name() string {
	return "ConditionalBoundaryMutator"
}

func (m *ConditionalBoundaryMutator) Category() string {
	return "business_logic"
}

func (m *ConditionalBoundaryMutator) CanMutate(node ast.Node) bool {
	if binExpr, ok := node.(*ast.BinaryExpr); ok {
		switch binExpr.Op {
		case token.LSS, token.GTR, token.LEQ, token.GEQ, token.EQL, token.NEQ:
			return true
		}
	}
	return false
}

func (m *ConditionalBoundaryMutator) Mutate(node ast.Node) (ast.Node, error) {
	binExpr := node.(*ast.BinaryExpr)
	mutated := *binExpr
	
	// Mutate boundary conditions
	switch binExpr.Op {
	case token.LSS: // < becomes <=
		mutated.Op = token.LEQ
	case token.GTR: // > becomes >=
		mutated.Op = token.GEQ
	case token.LEQ: // <= becomes <
		mutated.Op = token.LSS
	case token.GEQ: // >= becomes >
		mutated.Op = token.GTR
	case token.EQL: // == becomes !=
		mutated.Op = token.NEQ
	case token.NEQ: // != becomes ==
		mutated.Op = token.EQL
	}
	
	return &mutated, nil
}

// ArithmeticOperatorMutator mutates arithmetic operators
type ArithmeticOperatorMutator struct{}

func (m *ArithmeticOperatorMutator) Name() string {
	return "ArithmeticOperatorMutator"
}

func (m *ArithmeticOperatorMutator) Category() string {
	return "business_logic"
}

func (m *ArithmeticOperatorMutator) CanMutate(node ast.Node) bool {
	if binExpr, ok := node.(*ast.BinaryExpr); ok {
		switch binExpr.Op {
		case token.ADD, token.SUB, token.MUL, token.QUO, token.REM:
			return true
		}
	}
	return false
}

func (m *ArithmeticOperatorMutator) Mutate(node ast.Node) (ast.Node, error) {
	binExpr := node.(*ast.BinaryExpr)
	mutated := *binExpr
	
	// Mutate arithmetic operators
	switch binExpr.Op {
	case token.ADD: // + becomes -
		mutated.Op = token.SUB
	case token.SUB: // - becomes +
		mutated.Op = token.ADD
	case token.MUL: // * becomes /
		mutated.Op = token.QUO
	case token.QUO: // / becomes *
		mutated.Op = token.MUL
	case token.REM: // % becomes *
		mutated.Op = token.MUL
	}
	
	return &mutated, nil
}

// LogicalOperatorMutator mutates logical operators
type LogicalOperatorMutator struct{}

func (m *LogicalOperatorMutator) Name() string {
	return "LogicalOperatorMutator"
}

func (m *LogicalOperatorMutator) Category() string {
	return "business_logic"
}

func (m *LogicalOperatorMutator) CanMutate(node ast.Node) bool {
	if binExpr, ok := node.(*ast.BinaryExpr); ok {
		switch binExpr.Op {
		case token.LAND, token.LOR:
			return true
		}
	}
	if unaryExpr, ok := node.(*ast.UnaryExpr); ok {
		return unaryExpr.Op == token.NOT
	}
	return false
}

func (m *LogicalOperatorMutator) Mutate(node ast.Node) (ast.Node, error) {
	if binExpr, ok := node.(*ast.BinaryExpr); ok {
		mutated := *binExpr
		switch binExpr.Op {
		case token.LAND: // && becomes ||
			mutated.Op = token.LOR
		case token.LOR: // || becomes &&
			mutated.Op = token.LAND
		}
		return &mutated, nil
	}
	
	if unaryExpr, ok := node.(*ast.UnaryExpr); ok {
		// Remove NOT operator (!)
		return unaryExpr.X, nil
	}
	
	return node, nil
}

// ReturnValueMutator mutates return values
type ReturnValueMutator struct{}

func (m *ReturnValueMutator) Name() string {
	return "ReturnValueMutator"
}

func (m *ReturnValueMutator) Category() string {
	return "business_logic"
}

func (m *ReturnValueMutator) CanMutate(node ast.Node) bool {
	if retStmt, ok := node.(*ast.ReturnStmt); ok {
		return len(retStmt.Results) > 0
	}
	return false
}

func (m *ReturnValueMutator) Mutate(node ast.Node) (ast.Node, error) {
	retStmt := node.(*ast.ReturnStmt)
	mutated := *retStmt
	
	// Mutate return values
	for i, result := range retStmt.Results {
		switch expr := result.(type) {
		case *ast.BasicLit:
			if expr.Kind == token.INT {
				// Change integer return values
				mutated.Results[i] = &ast.BasicLit{
					Kind:  token.INT,
					Value: "0",
				}
			} else if expr.Kind == token.STRING {
				// Change string return values
				mutated.Results[i] = &ast.BasicLit{
					Kind:  token.STRING,
					Value: `""`,
				}
			}
		case *ast.Ident:
			if expr.Name == "true" {
				mutated.Results[i] = &ast.Ident{Name: "false"}
			} else if expr.Name == "false" {
				mutated.Results[i] = &ast.Ident{Name: "true"}
			} else if expr.Name == "nil" {
				// Create a non-nil value (this is context-dependent)
				mutated.Results[i] = &ast.BasicLit{
					Kind:  token.STRING,
					Value: `"mutated"`,
				}
			}
		}
	}
	
	return &mutated, nil
}

// NullCheckMutator mutates null/nil checks
type NullCheckMutator struct{}

func (m *NullCheckMutator) Name() string {
	return "NullCheckMutator"
}

func (m *NullCheckMutator) Category() string {
	return "security"
}

func (m *NullCheckMutator) CanMutate(node ast.Node) bool {
	if binExpr, ok := node.(*ast.BinaryExpr); ok {
		// Check for nil comparisons
		if (isNilIdent(binExpr.X) || isNilIdent(binExpr.Y)) &&
			(binExpr.Op == token.EQL || binExpr.Op == token.NEQ) {
			return true
		}
	}
	return false
}

func (m *NullCheckMutator) Mutate(node ast.Node) (ast.Node, error) {
	binExpr := node.(*ast.BinaryExpr)
	mutated := *binExpr
	
	// Flip nil checks
	if binExpr.Op == token.EQL {
		mutated.Op = token.NEQ
	} else if binExpr.Op == token.NEQ {
		mutated.Op = token.EQL
	}
	
	return &mutated, nil
}

// SecurityMutator targets security-critical code patterns
type SecurityMutator struct{}

func (m *SecurityMutator) Name() string {
	return "SecurityMutator"
}

func (m *SecurityMutator) Category() string {
	return "security"
}

func (m *SecurityMutator) CanMutate(node ast.Node) bool {
	if callExpr, ok := node.(*ast.CallExpr); ok {
		if ident, ok := callExpr.Fun.(*ast.Ident); ok {
			// Target security-related function calls
			securityFunctions := []string{
				"bcrypt", "CompareHashAndPassword", "GenerateFromPassword",
				"ValidateToken", "SignToken", "CheckPermission",
				"Authenticate", "Authorize", "ValidateInput",
			}
			
			for _, secFunc := range securityFunctions {
				if ident.Name == secFunc {
					return true
				}
			}
		}
		
		// Check for selector expressions (method calls)
		if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			securityMethods := []string{
				"CompareHashAndPassword", "GenerateFromPassword",
				"ValidateToken", "SignToken", "CheckPermission",
			}
			
			for _, method := range securityMethods {
				if selExpr.Sel.Name == method {
					return true
				}
			}
		}
	}
	
	return false
}

func (m *SecurityMutator) Mutate(node ast.Node) (ast.Node, error) {
	callExpr := node.(*ast.CallExpr)
	
	// For security functions, we can mutate arguments or return handling
	// This is a simplified example - in practice, you'd want more sophisticated mutations
	
	if len(callExpr.Args) > 0 {
		mutated := *callExpr
		// Mutate first argument to empty string for string arguments
		if basicLit, ok := callExpr.Args[0].(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
			mutated.Args[0] = &ast.BasicLit{
				Kind:  token.STRING,
				Value: `""`,
			}
		}
		return &mutated, nil
	}
	
	return node, nil
}

// PerformanceMutator targets performance-critical code patterns
type PerformanceMutator struct{}

func (m *PerformanceMutator) Name() string {
	return "PerformanceMutator"
}

func (m *PerformanceMutator) Category() string {
	return "performance"
}

func (m *PerformanceMutator) CanMutate(node ast.Node) bool {
	if callExpr, ok := node.(*ast.CallExpr); ok {
		if ident, ok := callExpr.Fun.(*ast.Ident); ok {
			// Target performance-related functions
			perfFunctions := []string{
				"make", "append", "len", "cap",
				"Query", "Exec", "Prepare",
				"Get", "Set", "Del", // Cache operations
			}
			
			for _, perfFunc := range perfFunctions {
				if ident.Name == perfFunc {
					return true
				}
			}
		}
		
		// Check for method calls on performance-critical objects
		if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			perfMethods := []string{
				"Query", "Exec", "Prepare", "QueryRow",
				"Get", "Set", "Del", "Exists",
				"Append", "Prepend",
			}
			
			for _, method := range perfMethods {
				if selExpr.Sel.Name == method {
					return true
				}
			}
		}
	}
	
	// Target loop conditions that might affect performance
	if forStmt, ok := node.(*ast.ForStmt); ok {
		return forStmt.Cond != nil
	}
	
	return false
}

func (m *PerformanceMutator) Mutate(node ast.Node) (ast.Node, error) {
	if callExpr, ok := node.(*ast.CallExpr); ok {
		mutated := *callExpr
		
		// For make() calls, mutate the capacity
		if ident, ok := callExpr.Fun.(*ast.Ident); ok && ident.Name == "make" {
			if len(callExpr.Args) >= 2 {
				// Change capacity to a smaller value
				mutated.Args[1] = &ast.BasicLit{
					Kind:  token.INT,
					Value: "1",
				}
			}
		}
		
		return &mutated, nil
	}
	
	if forStmt, ok := node.(*ast.ForStmt); ok {
		mutated := *forStmt
		
		// Mutate loop condition to potentially create infinite loop or early termination
		if binExpr, ok := forStmt.Cond.(*ast.BinaryExpr); ok {
			mutatedBinExpr := *binExpr
			switch binExpr.Op {
			case token.LSS:
				mutatedBinExpr.Op = token.LEQ
			case token.LEQ:
				mutatedBinExpr.Op = token.LSS
			case token.GTR:
				mutatedBinExpr.Op = token.GEQ
			case token.GEQ:
				mutatedBinExpr.Op = token.GTR
			}
			mutated.Cond = &mutatedBinExpr
		}
		
		return &mutated, nil
	}
	
	return node, nil
}

// Helper functions
func isNilIdent(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "nil"
	}
	return false
}