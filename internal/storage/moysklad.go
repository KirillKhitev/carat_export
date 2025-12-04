package storage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/KirillKhitev/carat_export/internal/config"
	"github.com/KirillKhitev/carat_export/internal/logger"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type MoySklad struct {
	client   *resty.Client
	m        *sync.RWMutex
	Products map[string]Product
}

type MetaAttribute struct {
	Href string `json:"href"`
	Type string `json:"type"`
}
type MoySkladAttribute struct {
	Meta  MetaAttribute `json:"meta"`
	Value any           `json:"value"`
}

type MoySkladAttributeRequest struct {
	Attributes []MoySkladAttribute `json:"attributes"`
}

type VKMetadata struct {
	Images map[string]int `json:"images"`
}

type ProductAttributeIDs struct {
	VkID       string
	VKMetadata string
}

var AttributeIDs ProductAttributeIDs = ProductAttributeIDs{}

func NewMoySklad() *MoySklad {
	return &MoySklad{
		client:   resty.New(),
		m:        &sync.RWMutex{},
		Products: make(map[string]Product),
	}
}

type Product struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Article         string                `json:"article"`
	Description     string                `json:"description"`
	ImagesResponse  ProductImagesResponse `json:"images"`
	VideoURL        string                `json:"video_url"`
	Images          []Image               `json:"images_array"`
	ExportAvito     bool                  `json:"-"`
	AvitoId         string                `json:"-"`
	AvitoClickCost  float64               `json:"-"`
	AvitoDailyLimit float64               `json:"-"`
	ExportVK        bool                  `json:"exportvk"`
	VKId            string                `json:"vkid"`
	VKMetadata      VKMetadata            `json:"vkmetadata"`
	Price           int                   `json:"price"`
	Stock           float32               `json:"stock"`
	Quantity        float32               `json:"quantity"`
	Attributes      []MoySkladAttribute   `json:"attributes"`
	SalePrices      []SalePrice           `json:"saleprices"`
	PathName        string                `json:"pathName"`
}

func (p *Product) prepareMoySkladAttributeRequest() *MoySkladAttributeRequest {
	var attributes []MoySkladAttribute

	attributes = append(attributes, MoySkladAttribute{
		Meta: MetaAttribute{
			Href: fmt.Sprintf("%sentity/product/metadata/attributes/%s", config.Config.MoySkladUrl, AttributeIDs.VkID),
			Type: "attributemetadata",
		},
		Value: p.VKId,
	})

	val, _ := json.MarshalIndent(p.VKMetadata, "", "    ")

	attributes = append(attributes, MoySkladAttribute{
		Meta: MetaAttribute{
			Href: fmt.Sprintf("%sentity/product/metadata/attributes/%s", config.Config.MoySkladUrl, AttributeIDs.VKMetadata),
			Type: "attributemetadata",
		},
		Value: string(val),
	})

	var result = new(MoySkladAttributeRequest)

	result.Attributes = attributes

	return result
}

type Image struct {
	Filename string `json:"filename"`
	Url      string `json:"url"`
}

type Attribute struct {
	Meta  MetaAttribute `json:"meta"`
	Id    string        `json:"id"`
	Name  string        `json:"name"`
	Type  string        `json:"type"`
	Value any           `json:"value,omitempty"`
}

type ProductListResponse struct {
	Meta MetaList  `json:"meta"`
	Rows []Product `json:"rows"`
}

type ProductResponse struct {
	Meta MetaProduct `json:"meta"`
	Rows []Product   `json:"rows"`
}

type MetaProduct struct {
	Href     string `json:"href"`
	UuidHref string `json:"uuidHref"`
}

type ProductImageListResponse struct {
	Meta MetaList   `json:"meta"`
	Rows []ImageRow `json:"rows"`
}

type ProductImagesResponse struct {
	Meta MetaList `json:"meta"`
}

type ImageRow struct {
	Meta     MetaImage `json:"meta,omitempty"`
	Filename string    `json:"filename,omitempty"`
}

type MetaImage struct {
	Href         string `json:"href,omitempty"`
	DownloadHref string `json:"downloadHref,omitempty"`
}
type SalePrice struct {
	Value float64 `json:"value"`
}

type MetaList struct {
	Href   string `json:"href,omitempty"`
	Size   int    `json:"size"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type APIServiceResult[T any] struct {
	Code     int
	Response T
	Error    error
}

type APIError struct {
	Code      int       `json:"code"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func (p *Product) UnmarshalJSON(data []byte) (err error) {
	type ProductAlias Product

	aliasValue := &struct {
		*ProductAlias
		Attributes []Attribute `json:"attributes,omitempty"`
		SalePrices []SalePrice `json:"salePrices,omitempty"`
	}{
		ProductAlias: (*ProductAlias)(p),
	}

	if err = json.Unmarshal(data, aliasValue); err != nil {
		return
	}

	for _, v := range aliasValue.Attributes {
		val := fmt.Sprintf("%v", v.Value)

		switch v.Name {
		case `Выгружать на Авито`:
			val, _ := strconv.ParseBool(val)
			p.ExportAvito = val
		case `AvitoId`:
			p.AvitoId = val
		case `Цена клика Avito`:
			val, err := strconv.ParseFloat(val, 32)
			if err != nil {
				logger.Log.WithFields(logrus.Fields{
					"Value":       val,
					"productID":   p.ID,
					"productName": p.Name,
					"error":       err,
				}).Logln(logrus.ErrorLevel, "Ошибка при парсинге Цены клика Avito товара из Мойсклад")
			}
			p.AvitoClickCost = math.Round(val*100) / 100
		case `Суточный лимит Avito`:
			val, err := strconv.ParseFloat(val, 32)
			if err != nil {
				logger.Log.WithFields(logrus.Fields{
					"Value":       val,
					"productID":   p.ID,
					"productName": p.Name,
					"error":       err,
				}).Logln(logrus.ErrorLevel, "Ошибка при парсинге Суточного лимита Avito товара из Мойсклад")
			}
			p.AvitoDailyLimit = math.Round(val*100) / 100
		case `Выгружать в VK`:
			val, _ := strconv.ParseBool(val)
			p.ExportVK = val
		case `VKId`:
			p.VKId = val
			AttributeIDs.VkID = v.Id
		case `Метаданные VK`:
			b := []byte(val)
			if err := json.Unmarshal(b, &p.VKMetadata); err != nil {
				logger.Log.WithFields(logrus.Fields{
					"Value":       val,
					"productID":   p.ID,
					"productName": p.Name,
					"error":       err,
				}).Logln(logrus.ErrorLevel, "Ошибка при парсинге VKMetadata товара из Мойсклад")
			}
			AttributeIDs.VKMetadata = v.Id

		case `VideoURL`:
			p.VideoURL = val
		}
	}

	p.Price = int(aliasValue.SalePrices[0].Value) / 100

	return
}

// GetProductsList формирует список товаров
func (s *MoySklad) GetProductsList(ctx context.Context) error {
	offset := 0
	needQuery := true

	for needQuery {
		url := fmt.Sprintf("%sentity/assortment?expand=images&offset=%d", config.Config.MoySkladUrl, offset)
		response := queryData[ProductListResponse, any](s, ctx, url, nil)

		if response.Error != nil {
			return response.Error
		}

		logger.Log.WithFields(logrus.Fields{
			"url":      url,
			"response": response,
		}).Logln(logrus.InfoLevel, "Получили ассортимент товаров из МойСклад")

		needQuery = len(response.Response.Rows) >= response.Response.Meta.Limit
		offset = offset + response.Response.Meta.Limit

		for _, product := range response.Response.Rows {
			if product.VKMetadata.Images == nil {
				product.VKMetadata.Images = make(map[string]int)
			}

			s.m.Lock()
			s.Products[product.ID] = product
			s.m.Unlock()
		}
	}

	return nil
}

// GetImagesListProduct получает массив картинок товаров
func (s *MoySklad) GetImagesListProduct(ctx context.Context, productId string, idImageWorker int) error {
	url := fmt.Sprintf("%sentity/product/%s/images", config.Config.MoySkladUrl, productId)
	response := queryData[ProductImageListResponse, any](s, ctx, url, nil)

	if response.Error != nil {
		return response.Error
	}

	product := s.Products[productId]
	product.Images = make([]Image, 0, len(response.Response.Rows))

	logger.Log.WithFields(logrus.Fields{
		"response": response,
	}).Logf(logrus.InfoLevel, "ImageWorker #%d получил список картинок товара '%s'", idImageWorker, productId)

	for _, imageRequest := range response.Response.Rows {
		s.getImage(ctx, imageRequest, &product, idImageWorker)
	}

	return nil
}

func (s *MoySklad) UpdateAttributesProduct(ctx context.Context, product Product) error {
	data := product.prepareMoySkladAttributeRequest()

	url := fmt.Sprintf("%sentity/product/%s", config.Config.MoySkladUrl, product.ID)
	response := queryData[Product, MoySkladAttributeRequest](s, ctx, url, data)

	if response.Error != nil {
		return response.Error
	}

	logger.Log.WithFields(logrus.Fields{
		"url":      url,
		"response": response,
	}).Logf(logrus.DebugLevel, "Обновили аттрибуты товару %s в МойСклад", product.Name)

	return nil
}

// getImage скачивает изображение на сервер, если его там нет, и заполняет массив картинок у товаров.
func (s *MoySklad) getImage(ctx context.Context, imageRequest ImageRow, product *Product, idImageWorker int) {
	image := Image{
		Filename: imageRequest.Filename,
		Url:      strings.Join([]string{"http://" + config.Config.ServerURL, config.Config.ImagesDir, imageRequest.Filename}, "/"),
	}

	product.Images = append(product.Images, image)

	s.m.Lock()
	s.Products[product.ID] = *product
	s.m.Unlock()

	filepath := strings.Join([]string{config.Config.ImagesPath, image.Filename}, string(os.PathSeparator))
	_, err := os.Stat(filepath)
	if err == nil {
		return
	}

	contextWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(20*time.Second))
	defer cancel()

	resp, errImg := s.client.R().
		SetHeader(`Authorization`, s.getAuthString()).
		SetHeader(`Accept-Encoding`, `gzip`).
		SetContext(contextWithTimeout).
		Get(imageRequest.Meta.DownloadHref)

	if errImg != nil {
		logger.Log.WithFields(logrus.Fields{
			"productId":     product.ID,
			"idImageWorker": idImageWorker,
			"filename":      image.Filename,
			"error":         errImg,
		}).Log(logrus.ErrorLevel, "ошибка скачивания картинки")

		return
	}

	out, err := os.Create(filepath)
	defer out.Close()

	if _, err := out.Write(resp.Body()); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"productId":     product.ID,
			"idImageWorker": idImageWorker,
			"filename":      image.Filename,
			"filepath":      filepath,
			"error":         errImg,
		}).Log(logrus.ErrorLevel, "ошибка сохранения картинки на диск")

		return
	}

	logger.Log.WithFields(logrus.Fields{
		"productId": product.ID,
		"filepath":  filepath,
		"error":     errImg,
	}).Logf(logrus.DebugLevel, "ImageWorker #%d загрузил изображение %s", idImageWorker, image.Filename)
}

func (s *MoySklad) Clear() {
	s.Products = make(map[string]Product, 0)
}

// queryData - запрос в API МойСклад
func queryData[T any, D any](s *MoySklad, ctx context.Context, url string, data *D) APIServiceResult[T] {
	result := APIServiceResult[T]{}

	contextWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(30*time.Second))
	defer cancel()

	var responseErr APIError

	request := s.client.R().
		SetHeader(`Authorization`, s.getAuthString()).
		SetHeader(`Accept-Encoding`, `gzip`).
		SetContext(contextWithTimeout).
		SetError(&result.Error).
		SetResult(&result.Response)

	var err error
	var response *resty.Response

	if data != nil {
		request.SetBody(data)
		response, err = request.Put(url)
	} else {
		response, err = request.Get(url)
	}

	if response.StatusCode() != 200 {
		result.Error = fmt.Errorf(string(response.Body()))
	}

	if err != nil {
		result.Error = fmt.Errorf("%v", responseErr)
		return result
	}

	result.Code = response.StatusCode()

	return result
}

// getAuthString формирует строку для авторизации.
func (s *MoySklad) getAuthString() string {
	authStr := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", config.Config.MoySkladLogin, config.Config.MoySkladPassword)))

	return fmt.Sprintf("Basic: %s", authStr)
}
