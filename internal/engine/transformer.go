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

// CompileExprWithItem compiles an expr expression that can reference `item`.
// Used for URL and body expressions inside a forEach rule.
func CompileExprWithItem(expression string) (*vm.Program, error) {
	env := map[string]any{
		"payload": map[string]any{},
		"item":    "",
	}
	return expr.Compile(expression, ExprOptions(env)...)
}

// TransformURL evaluates a compiled expression to produce a target URL string.
func TransformURL(program *vm.Program, payload map[string]any) (string, error) {
	return TransformURLWithItem(program, payload, nil)
}

// TransformURLWithItem evaluates a URL expression with an optional `item` variable.
func TransformURLWithItem(program *vm.Program, payload map[string]any, item any) (string, error) {
	env := map[string]any{
		"payload": payload,
	}
	if item != nil {
		env["item"] = item
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
	return TransformBodyWithItem(program, payload, nil)
}

// TransformBodyWithItem evaluates a body expression with an optional `item` variable.
func TransformBodyWithItem(program *vm.Program, payload map[string]any, item any) (map[string]any, error) {
	env := map[string]any{
		"payload": payload,
	}
	if item != nil {
		env["item"] = item
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

// EvalForEach evaluates a forEach expression and returns the result as a slice.
func EvalForEach(program *vm.Program, payload map[string]any) ([]any, error) {
	env := map[string]any{
		"payload": payload,
	}

	result, err := expr.Run(program, env)
	if err != nil {
		return nil, fmt.Errorf("forEach expression evaluation failed: %w", err)
	}

	items, ok := result.([]any)
	if !ok {
		return nil, fmt.Errorf("forEach expression must return array, got %T", result)
	}

	return items, nil
}
