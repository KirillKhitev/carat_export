package avito

import (
	"encoding/xml"
	"github.com/KirillKhitev/carat_export/internal/config"
	"github.com/KirillKhitev/carat_export/internal/logger"
	"github.com/KirillKhitev/carat_export/internal/storage"
	"github.com/sirupsen/logrus"
	"maps"
	"os"
	"strings"
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
	VideoURL    string             `xml:"VideoURL"`
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

// ConvertProducts готовит массив Товаров из МойСклад к виду, требуемуму Avito.
func ConvertProducts(source map[string]storage.Product) []Product {
	products := maps.Clone(source)
	for i, p := range products {
		if p.ExportAvito == false || p.ImagesResponse.Meta.Size == 0 || p.Price == 0 || p.Quantity == 0 {
			delete(products, i)
		}
	}

	result := make([]Product, 0, len(products))

	for _, p := range products {
		if len(p.Images) == 0 {
			logger.Log.Logf(logrus.ErrorLevel, "У товара '%s' не смогли загрузить картинки, убираем его из выгрузки", p.Name)
			continue
		}

		p.Description = strings.Join([]string{p.Article, p.Description, config.Config.ProductDescriptionAdd}, "\n")

		product := Product{
			ID:          p.ID,
			Title:       p.Name,
			Description: ProductDescription{Text: p.Description},
			AvitoId:     p.AvitoId,
			Price:       p.Price,
			VideoURL:    p.VideoURL,
			Address:     "Свердловская обл., Екатеринбург, ул. Хохрякова, 74",
			Category:    "Часы и украшения",
			GoodsType:   "Ювелирные изделия",
			AdType:      "Товар от производителя",
			Condition:   "Новое",
		}

		for _, img := range p.Images {
			image := Image{
				Url: img.Url,
			}

			product.Images.Image = append(product.Images.Image, image)
		}

		result = append(result, product)
	}

	return result
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
