package engine

import (
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// CompileExpr compiles a general-purpose expr expression.
// Used for both URL and body transformations.
func CompileExpr(expression string) (*vm.Program, error) {
	env := map[string]any{
		"payload": map[string]any{},
	}
	return expr.Compile(expression, ExprOptions(env)...)
}

// TransformURL evaluates a compiled expression to produce a target URL string.
func TransformURL(program *vm.Program, payload map[string]any) (string, error) {
	env := map[string]any{
		"payload": payload,
	}

	result, err := expr.Run(program, env)
	if err != nil {
		return "", fmt.Errorf("URL expression evaluation failed: %w", err)
	}

	url, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("URL expression must return string, got %T", result)
	}

	return url, nil
}

// TransformBody evaluates a compiled expression to produce a new JSON body.
// The expression should return a map (e.g., expr map literal {}).
func TransformBody(program *vm.Program, payload map[string]any) (map[string]any, error) {
	env := map[string]any{
		"payload": payload,
	}

	result, err := expr.Run(program, env)
	if err != nil {
		return nil, fmt.Errorf("body expression evaluation failed: %w", err)
	}

	body, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("body expression must return map, got %T", result)
	}

	return body, nil
}
