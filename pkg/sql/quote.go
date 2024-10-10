package sql

import (
	"fmt"
	"strings"
	"unsafe"
)

var stringQuoteReplacer = strings.NewReplacer(`\`, `\\`, `'`, `\'`)
var colEscape = strings.NewReplacer("`", "\\`", "\\", "\\\\")

func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func Quote(value interface{}) string {
	switch v := value.(type) {
	case int, uint32, int64:
		return fmt.Sprintf("%d", v)
	case string:
		return fmt.Sprintf("'%s'", stringQuoteReplacer.Replace(v))
	case []byte:
		return fmt.Sprintf("'%s'", stringQuoteReplacer.Replace(unsafeString(v)))
	default:
		panic("not implemented")
	}
}

func Column(c string) string {
	return fmt.Sprintf("`%s`", colEscape.Replace(c))
}
