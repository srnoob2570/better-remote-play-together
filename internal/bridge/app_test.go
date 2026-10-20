package bridge

import "github.com/srnoob2570/better-rpt/internal/app"

var _ App = (*app.Service)(nil)
