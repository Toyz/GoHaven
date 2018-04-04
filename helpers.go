package GoHaven

import (
	h "encoding/hex"
	"fmt"
	"strings"
)

func HexToRGB(hex string) (int, int, int) {
	if strings.HasPrefix(hex, "#") {
		hex = strings.Replace(hex, "#", "", 1)
	}

	if len(hex) == 3 {
		hex = fmt.Sprintf("%c%c%c%c%c%c", hex[0], hex[0], hex[1], hex[1], hex[2], hex[2])
	}

	d, _ := h.DecodeString(hex)

	return int(d[0]), int(d[1]), int(d[2])
}
