package engine

import (
	"fmt"

	"github.com/expr-lang/expr/vm"
	"github.com/shidaxi/go-webhook/internal/config"
)

// CompiledRule holds a rule with its pre-compiled expr programs.
type CompiledRule struct {
	Rule         config.Rule
	MatchProgram *vm.Program
	URLProgram   *vm.Program
	BodyProgram  *vm.Program
	CompileError error
}

// CompileRules compiles all rules' expr expressions.
// Rules that fail to compile will have CompileError set.
func CompileRules(rules []config.Rule) []CompiledRule {
	compiled := make([]CompiledRule, 0, len(rules))

	for _, r := range rules {
		cr := CompiledRule{Rule: r}

		matchProg, err := CompileMatchExpr(r.Match)
		if err != nil {
			cr.CompileError = fmt.Errorf("rule %q match compile error: %w", r.Name, err)
			compiled = append(compiled, cr)
			continue
		}
		cr.MatchProgram = matchProg

		urlProg, err := CompileExpr(r.Target.URL)
		if err != nil {
			cr.CompileError = fmt.Errorf("rule %q URL compile error: %w", r.Name, err)
			compiled = append(compiled, cr)
			continue
		}
		cr.URLProgram = urlProg

		bodyProg, err := CompileExpr(r.Body)
		if err != nil {
			cr.CompileError = fmt.Errorf("rule %q body compile error: %w", r.Name, err)
			compiled = append(compiled, cr)
			continue
		}
		cr.BodyProgram = bodyProg

		compiled = append(compiled, cr)
	}

	return compiled
}
