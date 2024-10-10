package sql

import "fmt"

type Where struct {
	where string
}

func NewWhere() *Where {
	return &Where{}
}

func (w *Where) And(exp string) {
	if exp == "" {
		return
	}
	if w.where != "" {
		w.where = fmt.Sprintf("(%s) AND (%s)", w.where, exp)
	} else {
		w.where = exp
	}
}

func (w *Where) Or(exp string) {
	if exp == "" {
		return
	}
	if w.where != "" {
		w.where = fmt.Sprintf("(%s) OR (%s)", w.where, exp)
	} else {
		w.where = exp
	}
}

func (w *Where) Andf(format string, obj ...interface{}) {
	w.And(fmt.Sprintf(format, obj...))
}

func (w *Where) String() string {
	return w.where
}

func (w *Where) SQL() string {
	if w.where == "" {
		return ""
	}
	return "WHERE " + w.where
}

func (w *Where) PreWhereSQL() string {
	if w.where == "" {
		return ""
	}
	return "PREWHERE " + w.where
}
