package qr_test

import (
	"image/png"
	"log"
	"os"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

func ExampleEncode() {
	qrcode, err := qr.Encode("hello world", qr.L, qr.Auto)
	if err != nil {
		log.Fatal("encode qr code: ", err)
	}

	qrcode, err = barcode.Scale(qrcode, 100, 100)
	if err != nil {
		log.Fatal("scale qr code: ", err)
	}

	f, err := os.Create("qrcode.png")
	if err != nil {
		log.Fatal("create image file: ", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal("close image file: ", err)
		}
	}()

	if err := png.Encode(f, qrcode); err != nil {
		log.Fatal("encode png: ", err)
	}
}
