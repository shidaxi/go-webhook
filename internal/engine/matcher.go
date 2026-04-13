package engine

import (
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// CompileMatchExpr compiles an expr match expression.
// The expression must evaluate to a boolean when run.
func CompileMatchExpr(expression string) (*vm.Program, error) {
	env := map[string]any{
		"payload": map[string]any{},
	}
	return expr.Compile(expression, ExprOptions(env)...)
}

// MatchRule evaluates a compiled match expression against the given payload.
// Returns true if the rule matches, false otherwise.
func MatchRule(program *vm.Program, payload map[string]any) (bool, error) {
	env := map[string]any{
		"payload": payload,
	}

	result, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("match expression evaluation failed: %w", err)
	}

	matched, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("match expression must return bool, got %T", result)
	}

	return matched, nil
}
