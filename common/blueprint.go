package common

import (
	"github.com/bmizerany/pat"
)

type Blueprint interface {
	AddRoutes(p *pat.PatternServeMux)
}
