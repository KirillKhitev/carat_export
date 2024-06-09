package avito

import (
	"encoding/xml"
	"github.com/KirillKhitev/carat_export/internal/config"
	"github.com/KirillKhitev/carat_export/internal/logger"
	"github.com/sirupsen/logrus"
	"os"
)

type Product struct {
	XMLName     xml.Name           `xml:"Ad"`
	ID          string             `xml:"Id"`
	AvitoId     string             `xml:"AvitoId,omitempty"`
	Title       string             `xml:"Title"`
	Description ProductDescription `xml:"Description"`
	Images      Images             `xml:"Images"`
	Address     string             `xml:"Address"`
	Category    string             `xml:"Category"`
	GoodsType   string             `xml:"GoodsType"`
	AdType      string             `xml:"AdType"`
	Condition   string             `xml:"Condition"`
	Price       int                `xml:"Price"`
}

type ProductDescription struct {
	Text string `xml:",cdata"`
}
type Images struct {
	Image []Image `xml:"Image"`
}

type Image struct {
	Url string `xml:"url,attr"`
}

type ProductsExport struct {
	XMLName       xml.Name  `xml:"Ads"`
	FormatVersion int       `xml:"formatVersion,attr"`
	Target        string    `xml:"target,attr"`
	Products      []Product `xml:"Ad"`
}

func CreateAutoloadFile(products []Product) error {
	logger.Log.Logln(logrus.InfoLevel, "Сохраняем товары в файл авито")
	logger.Log.WithFields(logrus.Fields{
		"products": products,
	}).Logln(logrus.DebugLevel, "Подготовленный список товаров")

	f, err := os.Create(config.Config.AvitoFilePath)
	defer f.Close()

	if err != nil {
		return err
	}

	pe := ProductsExport{}
	pe.FormatVersion = 3
	pe.Target = "Avito.ru"
	pe.Products = products

	var data []byte
	data, err = xml.MarshalIndent(pe, "", "   ")
	if err != nil {
		return err
	}

	f.Write(data)

	return nil
}
