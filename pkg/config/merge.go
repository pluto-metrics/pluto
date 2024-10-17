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
		if values[i] == nil {
			continue
		}
		ret.DSN = mergeAny(values[i].DSN, ret.DSN)
		ret.Params = merge(values[i].Params, ret.Params)
		ret.QueryLog.WhenStr = mergeAny(values[i].QueryLog.WhenStr, ret.QueryLog.WhenStr)
		ret.QueryLog.WhenExpr = merge(values[i].QueryLog.WhenExpr, ret.QueryLog.WhenExpr)
	}
	return ret
}
