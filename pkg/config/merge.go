package config

import (
	"github.com/expr-lang/expr/vm"
)

type nullable interface {
	map[string]string | *vm.Program
}

func mergeAny[T comparable](values ...T) T {
	var ret T
	for i := len(values) - 1; i >= 0; i-- {
		if values[i] != ret {
			return values[i]
		}
	}
	return ret
}

func merge[T nullable](values ...T) T {
	for i := len(values) - 1; i >= 0; i-- {
		if values[i] != nil {
			return values[i]
		}
	}
	var ret T
	return ret
}

func mergeClickHouse(values ...*ClickHouse) *ClickHouse {
	ret := &ClickHouse{}
	for i := len(values) - 1; i >= 0; i-- {
		ret.DSN = mergeAny(values[i].GetDSN(), ret.DSN)
		ret.Params = merge(values[i].GetParams(), ret.Params)
		ret.QueryLog = mergeAny(values[i].GetQueryLog(), ret.QueryLog)
		ret.QueryLogExpr = merge(values[i].GetQueryLogExpr(), ret.QueryLogExpr)
	}
	return ret
}
