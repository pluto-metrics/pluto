package sql

import "fmt"

func ArrayElement(a string, el string) string {
	return fmt.Sprintf("arrayElement(%s, %s)", a, el)
}

func Eq(a, b string) string {
	return fmt.Sprintf("%s = %s", a, b)
}

func Ne(a, b string) string {
	return fmt.Sprintf("%s != %s", a, b)
}

func Gt(a, b string) string {
	return fmt.Sprintf("%s > %s", a, b)
}

func Gte(a, b string) string {
	return fmt.Sprintf("%s >= %s", a, b)
}

func Lt(a, b string) string {
	return fmt.Sprintf("%s < %s", a, b)
}

func Lte(a, b string) string {
	return fmt.Sprintf("%s <= %s", a, b)
}
