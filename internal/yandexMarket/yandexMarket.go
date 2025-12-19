package yandexMarket

import (
	"encoding/xml"
	"github.com/KirillKhitev/carat_export/internal/config"
	"github.com/KirillKhitev/carat_export/internal/logger"
	"github.com/KirillKhitev/carat_export/internal/storage"
	"github.com/sirupsen/logrus"
	"maps"
	"os"
	"strings"
	"time"
)

type ProductsExport struct {
	XMLName xml.Name `xml:"yml_catalog"`
	Date    string   `xml:"date,attr"`
	Shop    Shop     `xml:"shop"`
}

type Shop struct {
	XMLName    xml.Name   `xml:"shop"`
	Name       string     `xml:"name"`
	Company    string     `xml:"company"`
	Categories []Category `xml:"categories>category"`
	Offers     Offers     `xml:"offers"`
}

type Category struct {
	XMLName  xml.Name `xml:"category"`
	ID       int      `xml:"id,attr"`
	Name     string   `xml:",chardata"`
	ParentID int      `xml:"parentId,attr,omitempty"`
}

type Offers struct {
	XMLName xml.Name `xml:"offers"`
	Offers  []Offer  `xml:"offers"`
}
type Offer struct {
	XMLName     xml.Name           `xml:"offer"`
	ID          string             `xml:"id,attr"`
	Available   bool               `xml:"available,attr"`
	Name        string             `xml:"name"`
	CategoryID  int                `xml:"categoryId"`
	Description ProductDescription `xml:"description"`
	Picture     []string           `xml:"picture"`
	Price       int                `xml:"price"`
	Video       []string           `xml:"video"`
	Archived    bool               `xml:"archived"`
	Count       int                `xml:"count"`
	Disabled    bool               `xml:"disabled"`
}

type ProductDescription struct {
	Text string `xml:",cdata"`
}

// ConvertProducts готовит массив Товаров из МойСклад к виду, требуемуму Yandex Market.
func ConvertProducts(source map[string]storage.Product) []Offer {
	products := maps.Clone(source)
	for i, p := range products {
		if p.ExportYMarket == false || p.ImagesResponse.Meta.Size == 0 || p.Price == 0 || p.Quantity == 0 {
			delete(products, i)
		}
	}

	result := make([]Offer, 0, len(products))

	for _, p := range products {
		if len(p.Images) == 0 {
			logger.Log.Logf(logrus.ErrorLevel, "У товара '%s' не смогли загрузить картинки, убираем его из выгрузки в Яндекс", p.Name)
			continue
		}

		p.Description = strings.Join([]string{p.Article, p.Description, config.Config.ProductDescriptionAdd}, "\n")

		offer := Offer{
			ID:          p.ID,
			Available:   true,
			Name:        p.Name,
			Description: ProductDescription{Text: p.Description},
			Price:       p.Price,
			Count:       int(p.Quantity),
			CategoryID:  config.Config.YMarketCategoryID,
		}

		for _, img := range p.Images {
			offer.Picture = append(offer.Picture, img.Url)
		}

		if p.VideoURL != "" {
			offer.Video = append(offer.Video, p.VideoURL)
		}

		result = append(result, offer)
	}

	return result
}

func CreateAutoloadFile(products []Offer) error {
	logger.Log.Logln(logrus.InfoLevel, "Сохраняем товары в файл yandex market")
	logger.Log.WithFields(logrus.Fields{
		"products": products,
	}).Logln(logrus.DebugLevel, "Подготовленный список товаров")

	f, err := os.Create(config.Config.YMarketFilepath)
	defer f.Close()

	if err != nil {
		return err
	}

	pe := ProductsExport{}
	pe.Date = time.Now().Format(time.RFC3339)
	pe.Shop = Shop{
		Name:    "CaratExport",
		Company: "ООО КАРАТ ЭКСПОРТ",
		Categories: []Category{
			{
				ID:   config.Config.YMarketCategoryID,
				Name: config.Config.YMarketCategoryName,
			},
		},
		Offers: Offers{
			Offers: products,
		},
	}

	var data []byte
	data, err = xml.MarshalIndent(pe, "", "   ")
	if err != nil {
		return err
	}

	result := []byte(xml.Header + string(data))

	f.Write(result)

	return nil
}
