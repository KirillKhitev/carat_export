package controller

import (
	"context"
	"fmt"
	"github.com/KirillKhitev/carat_export/internal/avito"
	"github.com/KirillKhitev/carat_export/internal/config"
	"github.com/KirillKhitev/carat_export/internal/email"
	"github.com/KirillKhitev/carat_export/internal/logger"
	"github.com/KirillKhitev/carat_export/internal/statistic"
	"github.com/KirillKhitev/carat_export/internal/storage"
	"github.com/sirupsen/logrus"
	"log"
	"maps"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Controller struct {
	storage              *storage.MoySklad
	wgImageWorkers       *sync.WaitGroup
	wgImportVKWorkers    *sync.WaitGroup
	stopImageWorkersChan chan struct{}
	stopVKWorkersChan    chan struct{}
	productIdsChan       chan string
	productVKIdsChan     chan string
	vkStorage            *storage.VK
	statistics           map[string]map[string][]statistic.StatisticRow
}

func NewController(ctx context.Context) *Controller {
	c := &Controller{
		storage:              storage.NewMoySklad(),
		wgImageWorkers:       &sync.WaitGroup{},
		wgImportVKWorkers:    &sync.WaitGroup{},
		stopImageWorkersChan: make(chan struct{}),
		stopVKWorkersChan:    make(chan struct{}),
		productIdsChan:       make(chan string),
		productVKIdsChan:     make(chan string),
		statistics:           make(map[string]map[string][]statistic.StatisticRow),
	}

	c.statistics["VK"] = make(map[string][]statistic.StatisticRow)

	return c
}

func (c *Controller) Start(ctx context.Context) {
	c.startProductsProcess(ctx)
}

func (c *Controller) Close() error {
	close(c.stopImageWorkersChan)

	c.wgImageWorkers.Wait()
	logger.Log.Log(logrus.InfoLevel, "Все ImageWorkers остановлены")

	close(c.stopVKWorkersChan)

	c.wgImportVKWorkers.Wait()
	logger.Log.Log(logrus.InfoLevel, "Все ImportVKWorkers остановлены")

	logger.Log.Logln(logrus.InfoLevel, "Контроллер остановлен")

	return nil
}

func (c *Controller) Clear() {
	c.storage.Clear()
	c.statistics = make(map[string]map[string][]statistic.StatisticRow)
	c.stopImageWorkersChan = make(chan struct{})
	c.stopVKWorkersChan = make(chan struct{})
	c.productIdsChan = make(chan string)
	c.productVKIdsChan = make(chan string)
}

func (c *Controller) startProductsProcess(ctx context.Context) {
	ticker := time.NewTicker(time.Second * time.Duration(config.Config.MoySkladInterval))

	defer ticker.Stop()

	c.process(ctx, config.Config.NeedDownloadProducts)

	for {
		<-ticker.C

		logger.Log.Restart()
		c.process(ctx, true)
	}
}

func (c *Controller) process(ctx context.Context, needDo bool) {
	defer close(c.productIdsChan)

	if !needDo {
		return
	}

	logger.Log.Logln(logrus.InfoLevel, "Начинаем выгрузку")

	c.doBeforeProcess(ctx)
	c.startImageWorkers(ctx)
	c.startVKWorkers(ctx)

	if err := c.storage.GetProductsList(ctx); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Logln(logrus.ErrorLevel, "Ошибка при получении списка товаров MoySklad")

		return
	}

	//c.updateAllProductsFromVK(ctx)
	c.downLoadImagesProducts()
	c.createAvitoAutoloadFile()
	c.processVK(ctx)
	c.Clear()

	logger.Log.Logln(logrus.InfoLevel, "Закончили выгрузку")
}

func (c *Controller) doBeforeProcess(ctx context.Context) {
	c.vkStorage = storage.NewVK(ctx)
}

func (c *Controller) downLoadImagesProducts() {
	for id, _ := range c.storage.Products {
		time.Sleep(time.Millisecond * 300)
		c.productIdsChan <- id
	}

	close(c.stopImageWorkersChan)

	c.wgImageWorkers.Wait()
}

func (c *Controller) startImageWorkers(ctx context.Context) {
	for w := 1; w <= config.Config.ImageWorkers; w++ {
		c.wgImageWorkers.Add(1)
		go c.imageWorker(ctx, w)
	}
}

func (c *Controller) startVKWorkers(ctx context.Context) {
	for w := 1; w <= config.Config.ImportVKWorkers; w++ {
		c.wgImportVKWorkers.Add(1)
		go c.vkWorker(ctx, w)
	}
}

func (c *Controller) imageWorker(ctx context.Context, idImageWorker int) {
	for {
		select {
		case <-c.stopImageWorkersChan:
			c.wgImageWorkers.Done()
			logger.Log.Logf(logrus.DebugLevel, "Остановили imageWorker #%d", idImageWorker)
			return
		default:
			select {
			case productId := <-c.productIdsChan:
				if err := c.storage.GetImagesListProduct(ctx, productId, idImageWorker); err != nil {
					logger.Log.WithFields(logrus.Fields{
						"error":       err,
						"ImageWorker": idImageWorker,
						"productId":   productId,
					}).Log(logrus.ErrorLevel, "Ошибка при получении списка картинок товара")

					continue
				}
			default:
			}
		}
	}
}

func (c *Controller) processVK(ctx context.Context) {
	logger.Log.Log(logrus.InfoLevel, "Начали обработку VK")
	for id, _ := range c.storage.Products {
		time.Sleep(time.Millisecond * 400)
		c.productVKIdsChan <- id
	}

	close(c.stopVKWorkersChan)

	c.wgImportVKWorkers.Wait()

	email.Notify(c.storage.Products, c.statistics)
}

func (c *Controller) vkWorker(ctx context.Context, idVKWorker int) {
	for {
		select {
		case <-c.stopVKWorkersChan:
			c.wgImportVKWorkers.Done()
			logger.Log.Logf(logrus.DebugLevel, "Остановили vkWorker #%d", idVKWorker)
			return
		default:
			select {
			case productId := <-c.productVKIdsChan:
				c.processVKProduct(ctx, productId, idVKWorker)
			default:
			}
		}
	}
}

func (c *Controller) processVKProduct(ctx context.Context, productId string, idVKWorker int) {
	product, ok := c.storage.Products[productId]

	if !ok {
		logger.Log.Logf(logrus.ErrorLevel, "Не нашли товар с ID %s", productId)
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"productId": product.ID,
		"VKWorker":  idVKWorker,
	}).Logf(logrus.InfoLevel, "Обрабатываем товар %s", product.Name)

	newProduct := c.saveProductInVK(ctx, product)

	c.storage.Products[product.ID] = newProduct

	if newProduct.VKId != product.VKId || !reflect.DeepEqual(newProduct.VKMetadata, product.VKMetadata) {
		if err := c.storage.UpdateAttributesProduct(ctx, newProduct); err != nil {
			logger.Log.WithFields(logrus.Fields{
				"error":       err,
				"productId":   product.ID,
				"productName": product.Name,
			}).Log(logrus.ErrorLevel, "Ошибка при обновлении VK-аттрибутов товара в МойСклад")

			return
		}
	}
}

func (c *Controller) saveProductInVK(ctx context.Context, bproduct storage.Product) storage.Product {
	product := bproduct
	product.VKMetadata.Images = maps.Clone(bproduct.VKMetadata.Images)

	if !product.ExportVK {
		if product.VKId != "" {
			newProduct, err := c.vkStorage.RemoveProduct(ctx, product)
			if err != nil {
				logger.Log.WithFields(logrus.Fields{
					"error":     err,
					"productID": product.ID,
					"VKID":      product.VKId,
				}).Logf(logrus.ErrorLevel, "Ошибка при удалении товара %s в VK", product.Name)

				c.addStatisticRow("VK", product.ID, fmt.Sprintf("Ошибка при удалении товара: %s", err), statistic.STATUS_ERROR)
			} else {
				c.addStatisticRow("VK", product.ID, "Успешно удалили товар", statistic.STATUS_NORMAL)
			}

			return newProduct
		}

		logger.Log.WithFields(logrus.Fields{
			"productID": product.ID,
		}).Logf(logrus.InfoLevel, "Пропускаем товар %s - не шлем в VK", product.Name)

		c.addStatisticRow("VK", product.ID, "Не отправляем товар в VK", statistic.STATUS_NORMAL)

		return product
	}

	if product.VKId != "" {
		newProduct, err := c.vkStorage.EditProduct(ctx, product, 0)
		if err != nil {
			logger.Log.WithFields(logrus.Fields{
				"error":   err,
				"product": product.ID,
				"VKID":    product.VKId,
			}).Logf(logrus.ErrorLevel, "Ошибка при изменении товара %s в VK", product.Name)

			c.addStatisticRow("VK", product.ID, fmt.Sprintf("Ошибка при изменении товара: %s", err), statistic.STATUS_ERROR)

			return newProduct
		}

		c.addStatisticRow("VK", product.ID, "Успешно обновили товар", statistic.STATUS_NORMAL)

		time.Sleep(time.Millisecond * 400)

		addToAlbum, err := c.vkStorage.AddProductToAlbum(ctx, newProduct)
		if err != nil {
			c.addStatisticRow("VK", product.ID, err.Error(), statistic.STATUS_ERROR)
		}

		if addToAlbum {
			c.addStatisticRow("VK", product.ID, fmt.Sprintf("Успешно добавили товар в подборку %s", product.PathName), statistic.STATUS_NORMAL)
		}

		return newProduct
	}

	newProduct, err := c.vkStorage.CreateProduct(ctx, product)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Logf(logrus.ErrorLevel, "Ошибка при создании товара %s в VK", product.Name)

		c.addStatisticRow("VK", product.ID, fmt.Sprintf("Ошибка при создании товара: %s", err), statistic.STATUS_ERROR)

		return newProduct
	}

	c.addStatisticRow("VK", product.ID, "Успешно создали товар", statistic.STATUS_NORMAL)

	time.Sleep(time.Millisecond * 400)

	addToAlbum, err := c.vkStorage.AddProductToAlbum(ctx, newProduct)
	if err != nil {
		c.addStatisticRow("VK", product.ID, err.Error(), statistic.STATUS_ERROR)
	}

	if addToAlbum {
		c.addStatisticRow("VK", product.ID, fmt.Sprintf("Успешно добавили товар в подборку %s", product.PathName), statistic.STATUS_NORMAL)
	}

	return newProduct
}

// convertProductsToAvito готовит массив Товаров из МойСклад к виду, требуемуму Avito.
func (c *Controller) convertProductsToAvito(source map[string]storage.Product) []avito.Product {
	products := maps.Clone(source)
	for i, p := range products {
		if p.ExportAvito == false || p.ImagesResponse.Meta.Size == 0 || p.Price == 0 || p.Quantity == 0 {
			delete(products, i)
		}
	}

	result := make([]avito.Product, 0, len(products))

	for _, p := range products {
		if len(p.Images) == 0 {
			logger.Log.Logf(logrus.ErrorLevel, "У товара '%s' не смогли загрузить картинки, убираем его из выгрузки", p.Name)
			continue
		}

		p.Description = strings.Join([]string{p.Article, p.Description, config.Config.ProductDescriptionAdd}, "\n")

		product := avito.Product{
			ID:          p.ID,
			Title:       p.Name,
			Description: avito.ProductDescription{Text: p.Description},
			AvitoId:     p.AvitoId,
			Price:       p.Price,
			VideoURL:    p.VideoURL,
			Address:     "Свердловская обл., Екатеринбург, ул. Хохрякова, 74",
			Category:    "Часы и украшения",
			GoodsType:   "Другое",
			AdType:      "Продаю своё",
			Condition:   "Новое",
		}

		for _, img := range p.Images {
			image := avito.Image{
				Url: img.Url,
			}

			product.Images.Image = append(product.Images.Image, image)
		}

		result = append(result, product)
	}

	return result
}

func (c *Controller) createAvitoAutoloadFile() {
	products := c.convertProductsToAvito(c.storage.Products)

	if err := avito.CreateAutoloadFile(products); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Log(logrus.ErrorLevel, "Ошибка при сохранении товаров в файл выгрузки Avito")
	}
}

func (c *Controller) updateAllProductsFromVK(ctx context.Context) {
	if err := c.vkStorage.GetProductList(ctx); err != nil {
		log.Fatal(err)
	}

	for _, vkItem := range c.vkStorage.Products {
		for _, product := range c.storage.Products {
			if product.Article != vkItem.SKU {
				continue
			}

			if product.VKId == "" {
				fmt.Println(product)
			}

			product.VKId = strconv.Itoa(vkItem.ID)
			c.updateProductFromVKToMoySklad(ctx, product)

			logger.Log.WithFields(logrus.Fields{
				"productId": product.ID,
			}).Log(logrus.InfoLevel, "Обновили товар в Мойсклад "+product.Name)

			break
		}
	}
}

func (c *Controller) updateProductFromVKToMoySklad(ctx context.Context, product storage.Product) {
	if err := c.storage.UpdateAttributesProduct(ctx, product); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":     err,
			"productId": product.ID,
		}).Log(logrus.ErrorLevel, "Ошибка при обновлении VK-аттрибутов товара в МойСклад")

		return
	}
}

func (c *Controller) addStatisticRow(category, id, message string, status statistic.Status) {
	c.statistics[category][id] = append(
		c.statistics[category][id],
		*statistic.NewStatisticaRow(message, status))
}
