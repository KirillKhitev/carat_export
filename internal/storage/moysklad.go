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
	"os"
	"slices"
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

func NewMoySklad() *MoySklad {
	return &MoySklad{
		client:   resty.New(),
		m:        &sync.RWMutex{},
		Products: make(map[string]Product),
	}
}

type Product struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	Article        string                `json:"article"`
	Description    string                `json:"description"`
	ImagesResponse ProductImagesResponse `json:"images"`
	Images         []Image               `json:"-"`
	ExportAvito    bool                  `json:"-"`
	AvitoId        string                `json:"-"`
	Price          int                   `json:"-"`
	Stock          float32               `json:"stock"`
}

type Image struct {
	Filename string `json:"filename"`
	Url      string `json:"url"`
}

type Attribute struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Value any    `json:"value,omitempty"`
}

type ProductListResponse struct {
	Meta MetaList  `json:"meta"`
	Rows []Product `json:"rows"`
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
		response := queryData[ProductListResponse](s, ctx, url)

		if response.Error != nil {
			return response.Error
		}

		logger.Log.WithFields(logrus.Fields{
			"url":      url,
			"response": response,
		}).Logln(logrus.InfoLevel, "Получили ассортимент товаров из МойСклад")

		needQuery = len(response.Response.Rows) >= response.Response.Meta.Limit
		offset = offset + response.Response.Meta.Limit

		for _, product := range filterProducts(response.Response.Rows) {
			s.m.Lock()
			s.Products[product.ID] = product
			s.m.Unlock()
		}

		logger.Log.WithFields(logrus.Fields{
			"products": s.Products,
		}).Logln(logrus.DebugLevel, "Отфильтровали товары из МойСклад")
	}

	return nil
}

// GetImagesListProduct получает массив картинок товаров
func (s *MoySklad) GetImagesListProduct(ctx context.Context, productId string, idImageWorker int) error {
	url := fmt.Sprintf("%sentity/product/%s/images", config.Config.MoySkladUrl, productId)
	response := queryData[ProductImageListResponse](s, ctx, url)

	if response.Error != nil {
		return response.Error
	}

	product := s.Products[productId]
	product.Images = make([]Image, 0, len(response.Response.Rows))

	logger.Log.WithFields(logrus.Fields{
		"response": response,
	}).Logf(logrus.DebugLevel, "ImageWorker #%d получил список картинок товара '%s'", idImageWorker, productId)

	for _, imageRequest := range response.Response.Rows {
		s.getImage(ctx, imageRequest, &product, idImageWorker)
	}

	return nil
}

// getImage скачивает изображение на сервер, если его там нет, и заполняет массив картинок у товаров.
func (s *MoySklad) getImage(ctx context.Context, imageRequest ImageRow, product *Product, idImageWorker int) {
	image := Image{
		Filename: imageRequest.Filename,
		Url:      strings.Join([]string{"http://" + config.Config.ImagesURL, config.Config.ImagesDir, imageRequest.Filename}, "/"),
	}

	product.Images = append(product.Images, image)

	s.m.Lock()
	s.Products[product.ID] = *product
	s.m.Unlock()

	filepath := strings.Join([]string{config.Config.ImagesDir, image.Filename}, string(os.PathSeparator))
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

func filterProducts(rows []Product) []Product {
	rows = slices.DeleteFunc(rows, func(p Product) bool {
		return p.ExportAvito == false || p.ImagesResponse.Meta.Size == 0 || p.Price == 0 || p.Stock == 0
	})

	return rows
}

// queryData - запрос в API МойСклад
func queryData[T any](s *MoySklad, ctx context.Context, url string) APIServiceResult[T] {
	result := APIServiceResult[T]{}

	contextWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(20*time.Second))
	defer cancel()

	var responseErr APIError

	response, err := s.client.R().
		SetHeader(`Authorization`, s.getAuthString()).
		SetHeader(`Accept-Encoding`, `gzip`).
		SetContext(contextWithTimeout).
		SetError(&result.Error).
		SetResult(&result.Response).
		Get(url)

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
