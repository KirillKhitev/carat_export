package storage

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/KirillKhitev/carat_export/internal/config"
	"github.com/KirillKhitev/carat_export/internal/logger"
	vk "github.com/SevereCloud/vksdk/v3/api"
	vkobject "github.com/SevereCloud/vksdk/v3/object"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type VK struct {
	vkApi                  *vk.VK
	vkImageUploadServerURL string
	Products               map[int]vkobject.MarketMarketItem
	client                 *resty.Client
	Albums                 map[string]int
}

var MAX_IMAGES_COUNT int = 4

func NewVK(ctx context.Context) *VK {
	v := &VK{
		Products: make(map[int]vkobject.MarketMarketItem),
		client:   resty.New(),
		Albums:   make(map[string]int),
	}

	err := v.UpdateAccessToken()
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Log(logrus.ErrorLevel, "Ошибка при обновлении VK Access_token")
	}

	logger.Log.Log(logrus.InfoLevel, "Успешно обновили VK Access_token")

	v.vkApi = vk.NewVK(config.Config.VkUserToken)
	v.vkApi.Limit = vk.LimitUserToken

	err = v.UpdateImageUploadServerURL(ctx)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Log(logrus.ErrorLevel, "Ошибка при получении адреса VK для загрузки Картинок")
	}

	logger.Log.Log(logrus.InfoLevel, "Успешно получили адрес VK для загрузки Картинок")

	err = v.GetAlbums(ctx)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Log(logrus.ErrorLevel, "Ошибка при получении списка подборок товаров")
	}

	logger.Log.Log(logrus.InfoLevel, "Успешно получили список подборок товаров")

	return v
}

func (v *VK) UpdateImageUploadServerURL(ctx context.Context) error {
	params := vk.Params{
		"group_id": config.Config.VkGroupID,
	}

	response, err := v.getProductPhotoUploadServer(params)
	if err != nil {
		return err
	}

	v.vkImageUploadServerURL = response.UploadURL

	return nil
}

func (v *VK) getProductPhotoUploadServer(params vk.Params) (response vk.PhotosGetMarketUploadServerResponse, err error) {
	err = v.vkApi.RequestUnmarshal("market.getProductPhotoUploadServer", &response, params)
	return
}

func (v *VK) GetAlbums(ctx context.Context) error {
	params := vk.Params{
		"owner_id": -config.Config.VkGroupID,
	}

	response, err := v.vkApi.MarketGetAlbums(params)
	if err != nil {
		return err
	}

	for _, item := range response.Items {
		title := strings.TrimSpace(item.Title)
		v.Albums[strings.ToUpper(title)] = item.ID
	}

	return nil
}

func (v *VK) GetProductList(ctx context.Context) error {
	offset := 0
	needQuery := true

	params := vk.Params{
		"owner_id": -config.Config.VkGroupID,
		"extended": 1,
		"offset":   offset,
	}

	for needQuery {
		response, err := v.vkApi.MarketGet(params)
		if err != nil {
			return err
		}

		logger.Log.WithFields(logrus.Fields{
			"response": response,
		}).Logln(logrus.DebugLevel, "Получили ассортимент товаров из ВК")

		params["offset"] = params["offset"].(int) + len(response.Items)

		for _, product := range response.Items {
			v.Products[product.ID] = product
		}

		needQuery = (len(v.Products) < response.Count) && len(response.Items) > 0
	}

	return nil
}

func (v *VK) RemoveProduct(ctx context.Context, product Product) (Product, error) {
	product, err := v.syncImages(ctx, product)
	product.VKMetadata.Hash = product.GetHashVK()

	params := v.prepareParamsProduct(product, 1)

	response, err := v.vkApi.MarketEdit(params)
	if err != nil {
		return product, err
	}

	if response == 0 {
		return product, fmt.Errorf("Сервер VK ответил %s", response)
	}

	logger.Log.Logf(logrus.InfoLevel, "Удалили товар %s в VK", product.Name)

	return product, err
}

func (v *VK) EditProduct(ctx context.Context, product Product, deleted int) (Product, bool, error) {
	product, err := v.syncImages(ctx, product)

	if !product.NeedSendToVK() {
		return product, true, fmt.Errorf("Товар не изменился, не шлем в VK")
	}

	product.VKMetadata.Hash = product.GetHashVK()

	params := v.prepareParamsProduct(product, deleted)

	if _, ok := params["main_photo_id"]; !ok {
		return product, false, fmt.Errorf("не определили главное фото")
	}

	response, err := v.vkApi.MarketEdit(params)
	if err != nil {
		return product, false, err
	}

	if response == 0 {
		return product, false, fmt.Errorf("Сервер VK ответил %s", response)
	}

	logger.Log.Logf(logrus.InfoLevel, "Отредактировали товар %s в VK", product.Name)

	return product, false, nil
}

func (v *VK) CreateProduct(ctx context.Context, product Product) (Product, error) {
	if len(product.Images) == 0 {
		return product, fmt.Errorf("У товара нет картинок, не создаем в VK")
	}

	product, err := v.syncImages(ctx, product)
	product.VKMetadata.Hash = product.GetHashVK()

	params := v.prepareParamsProduct(product, 0)

	if _, ok := params["main_photo_id"]; !ok {
		return product, fmt.Errorf("не определили главное фото")
	}

	response, err := v.vkApi.MarketAdd(params)
	if err != nil {
		return product, err
	}

	if response.MarketItemID == 0 {
		logger.Log.Logf(logrus.ErrorLevel, "Ошибка при создании товара %s в VK", product.Name)
		return product, nil
	}

	logger.Log.WithFields(logrus.Fields{
		"vkID": response.MarketItemID,
	}).Logf(logrus.InfoLevel, "Создали новый товар %s в VK", product.Name)

	product.VKId = strconv.Itoa(response.MarketItemID)

	return product, nil
}

func (v *VK) prepareParamsProduct(product Product, deleted int) vk.Params {
	if product.Price == 0 {
		deleted = 1
	}

	params := vk.Params{
		"owner_id":     -config.Config.VkGroupID,
		"name":         product.Name,
		"description":  product.Description,
		"category_id":  config.Config.VkCategoryID,
		"price":        product.Price,
		"sku":          product.Article,
		"stock_amount": int(product.Quantity),
		"deleted":      deleted,
	}

	if len(product.VKId) > 0 {
		params["item_id"] = product.VKId
	}

	photos := make([]string, 0)

	for index, image := range product.Images {
		imgID, ok := product.VKMetadata.Images[image.Filename]
		if !ok {
			continue
		}

		if index >= MAX_IMAGES_COUNT {
			break
		}

		if index == 0 {
			params["main_photo_id"] = imgID
		} else {
			photos = append(photos, strconv.Itoa(imgID))
		}
	}

	if len(photos) > 0 {
		params["photo_ids"] = strings.Join(photos, ",")
	}

	logger.Log.WithFields(logrus.Fields{
		"params": params,
	}).Logf(logrus.InfoLevel, "Подготовили параметры товара %s для отправки в VK", product.Name)

	return params
}

func (v *VK) AddProductToAlbum(ctx context.Context, product Product) (bool, error) {
	title := strings.TrimSpace(product.PathName)
	albumID, ok := v.Albums[strings.ToUpper(title)]
	if !ok {
		return false, nil
	}

	if product.VKId == "" {
		return false, nil
	}

	params := v.prepareParamsForAddAlbum(product, albumID)

	_, err := v.vkApi.MarketAddToAlbum(params)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":   err,
			"product": product.ID,
			"VKID":    product.VKId,
		}).Logf(logrus.ErrorLevel, "Ошибка при добавлении товара %s в альбом VK %s", product.Name, product.PathName)
		return false, fmt.Errorf("Ошибка при добавлении товара %s в альбом VK %s - %w", product.Name, product.PathName, err)
	}

	logger.Log.WithFields(logrus.Fields{
		"product": product.ID,
		"VKID":    product.VKId,
	}).Logf(logrus.InfoLevel, "Успешно добавили товар %s в альбом VK %s", product.Name, product.PathName)

	return true, nil
}

func (v *VK) prepareParamsForAddAlbum(product Product, albumID int) vk.Params {
	params := vk.Params{
		"owner_id":  -config.Config.VkGroupID,
		"item_ids":  product.VKId,
		"album_ids": albumID,
	}

	return params
}

func (v *VK) syncImages(ctx context.Context, product Product) (Product, error) {
	if len(product.Images) == 0 {
		product.VKMetadata.Images = make(map[string]int)

		return product, nil
	}

	for _, image := range product.Images {
		_, ok := product.VKMetadata.Images[image.Filename]
		if !ok {
			imgID, err := v.uploadImage(ctx, image.Filename)
			if err != nil {
				logger.Log.WithFields(logrus.Fields{
					"error":   err,
					"product": product.Name,
				}).Logf(logrus.ErrorLevel, "Ошибка загрузки в VK картинки %s", image.Filename)
				continue
			}

			logger.Log.WithFields(logrus.Fields{
				"image":    image.Filename,
				"product":  product.Name,
				"VK imgID": imgID,
			}).Log(logrus.InfoLevel, "Загрузили картинку в ВК")

			product.VKMetadata.Images[image.Filename] = imgID
		}
	}

	return product, nil
}

func (v *VK) uploadImage(ctx context.Context, imageName string) (imgID int, err error) {
	filepath := strings.Join([]string{config.Config.ImagesPath, imageName}, string(os.PathSeparator))
	f, err := os.OpenFile(filepath, os.O_RDWR, 0)
	if err != nil {
		return
	}

	defer f.Close()

	responseImg, err := v.uploadMarketPhoto(f)

	if responseImg.PhotoID == 0 {
		return
	}

	imgID = responseImg.PhotoID

	return
}

type PhotosSaveProductPhotoResponse struct {
	PhotoID int `json:"photo_id"`
}

func (v *VK) uploadMarketPhoto(file io.Reader) (response PhotosSaveProductPhotoResponse, err error) {
	bodyContent, err := v.vkApi.UploadFile(v.vkImageUploadServerURL, file, "file", "photo.jpeg")
	if err != nil {
		return
	}

	err = v.vkApi.RequestUnmarshal("market.saveProductPhoto", &response, vk.Params{
		"upload_response": string(bodyContent),
	})

	return
}

type AccessTokenResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	UserId       int    `json:"user_id"`
	State        string `json:"state"`
	Scope        string `json:"scope"`
}

func (v *VK) UpdateAccessToken() error {
	contextWithTimeout, cancel := context.WithTimeout(context.Background(), time.Duration(20*time.Second))
	defer cancel()

	var respErr error

	data := make(map[string]string)

	data["grant_type"] = "refresh_token"
	data["refresh_token"] = config.Config.VKRefreshToken
	data["client_id"] = strconv.Itoa(config.Config.VkClientID)
	data["device_id"] = config.Config.VKDeviceID
	data["state"] = rand.Text()
	data["scope"] = "market photos"

	result := AccessTokenResult{}

	response, err := v.client.R().
		SetHeader(`Content-Type`, `application/x-www-form-urlencoded`).
		SetContext(contextWithTimeout).
		SetMultipartFormData(data).
		SetResult(&result).
		SetError(&respErr).
		Post("https://id.vk.com/oauth2/auth")

	if err != nil {
		return err
	}

	if respErr != nil {
		return respErr
	}

	if response.StatusCode() != http.StatusOK {
		return errors.New("Ошибка в запросе")
	}

	if result.State != data["state"] {
		return errors.New("Неверная контрольная строка")
	}

	if result.RefreshToken == "" || result.AccessToken == "" {
		return errors.New("Пустой refresh или access token, не обновляем файл конфига")
	}

	config.Config.VKRefreshToken = result.RefreshToken
	config.Config.VkUserToken = result.AccessToken

	bytes, err := json.MarshalIndent(config.Config, "", "   ")

	f, err := os.OpenFile(config.DefaultConfigPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.Write(bytes)
	if err != nil {
		return err
	}

	return nil
}
