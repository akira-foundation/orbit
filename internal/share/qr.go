package share

import (
	"encoding/base64"

	qrcode "github.com/skip2/go-qrcode"
)

func QRDataURI(text string) (string, error) {
	png, err := qrcode.Encode(text, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}
