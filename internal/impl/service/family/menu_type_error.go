package family

import "fmt"

func errBadMenuType(s string) error {
	return fmt.Errorf("type: must be one of directory/menu/button，得到 %q", s)
}
