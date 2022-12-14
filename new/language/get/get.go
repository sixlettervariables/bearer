package get

import (
	"fmt"

	"github.com/bearer/curio/new/language/implementations/ruby"
	"github.com/bearer/curio/new/language/types"
)

func Get(name string) (types.Language, error) {
	switch name {
	case "ruby":
		return ruby.Get(), nil
	default:
		return nil, fmt.Errorf("unsupported language '%s'", name)
	}
}