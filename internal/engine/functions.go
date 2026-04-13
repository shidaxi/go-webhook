package engine

import (
	"os"
	"strings"
	"time"

	"github.com/expr-lang/expr"
)

// ExprOptions returns expr.Option slice with custom functions registered.
// Pass the environment map so expr can infer types from it.
func ExprOptions(env map[string]any) []expr.Option {
	return []expr.Option{
		expr.Env(env),
		expr.Function("now", func(params ...any) (any, error) {
			return time.Now().UTC().Format(time.RFC3339), nil
		},
			new(func() string),
		),
		expr.Function("env", func(params ...any) (any, error) {
			key := params[0].(string)
			return os.Getenv(key), nil
		},
			new(func(string) string),
		),
		expr.Function("lower", func(params ...any) (any, error) {
			return strings.ToLower(params[0].(string)), nil
		},
			new(func(string) string),
		),
		expr.Function("upper", func(params ...any) (any, error) {
			return strings.ToUpper(params[0].(string)), nil
		},
			new(func(string) string),
		),
		expr.Function("join", func(params ...any) (any, error) {
			items := params[0].([]any)
			sep := params[1].(string)
			strs := make([]string, 0, len(items))
			for _, item := range items {
				strs = append(strs, item.(string))
			}
			return strings.Join(strs, sep), nil
		},
			new(func([]any, string) string),
		),
		expr.Function("split", func(params ...any) (any, error) {
			s := params[0].(string)
			sep := params[1].(string)
			parts := strings.Split(s, sep)
			result := make([]any, len(parts))
			for i, p := range parts {
				result[i] = p
			}
			return result, nil
		},
			new(func(string, string) []any),
		),
	}
}
