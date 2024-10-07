package controller

import (
	"context"
	"github.com/KirillKhitev/carat_export/internal/avito"
	"github.com/KirillKhitev/carat_export/internal/config"
	"github.com/KirillKhitev/carat_export/internal/logger"
	"github.com/KirillKhitev/carat_export/internal/storage"
	"github.com/sirupsen/logrus"
	"strings"
	"sync"
	"time"
)

type Controller struct {
	storage              *storage.MoySklad
	wgImageWorkers       *sync.WaitGroup
	stopImageWorkersChan chan struct{}
	productIdsChan       chan string
}

func NewController() *Controller {
	return &Controller{
		storage:              storage.NewMoySklad(),
		wgImageWorkers:       &sync.WaitGroup{},
		stopImageWorkersChan: make(chan struct{}),
		productIdsChan:       make(chan string),
	}
}

func (c *Controller) Start(ctx context.Context) {
	c.startProductsProcess(ctx)
}

func (c *Controller) Close() error {
	close(c.stopImageWorkersChan)
	c.wgImageWorkers.Wait()

	logger.Log.Log(logrus.InfoLevel, "Все ImageWorkers остановлены")

	logger.Log.Logln(logrus.InfoLevel, "Контроллер остановлен")

	return nil
}

func (c *Controller) Clear() {
	c.storage.Clear()
	c.stopImageWorkersChan = make(chan struct{})
}

func (c *Controller) startProductsProcess(ctx context.Context) {
	ticker := time.NewTicker(time.Second * time.Duration(config.Config.MoySkladInterval))
	defer ticker.Stop()

	for {
		<-ticker.C

		logger.Log.Restart()
		logger.Log.Logln(logrus.InfoLevel, "Начинаем выгрузку")

		c.startImageWorkers(ctx)

		if err := c.storage.GetProductsList(ctx); err != nil {
			logger.Log.WithFields(logrus.Fields{
				"error": err,
			}).Logln(logrus.ErrorLevel, "Ошибка при получении списка товаров")

			continue
		}

		for id, _ := range c.storage.Products {
			c.productIdsChan <- id
		}

		close(c.stopImageWorkersChan)

		c.wgImageWorkers.Wait()

		products := c.convertProductsToAvito(c.storage.Products)

		if err := avito.CreateAutoloadFile(products); err != nil {
			logger.Log.WithFields(logrus.Fields{
				"error": err,
			}).Log(logrus.ErrorLevel, "Ошибка при сохранении товаров в файл выгрузки Avito")
		}

		c.Clear()

		logger.Log.Logln(logrus.InfoLevel, "Закончили выгрузку")
	}
}

func (c *Controller) startImageWorkers(ctx context.Context) {
	for w := 1; w <= config.Config.ImageWorkers; w++ {
		c.wgImageWorkers.Add(1)
		go c.imageWorker(ctx, w)
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

// convertProductsToAvito готовит массив Товаров из МойСклад к виду, требуемуму Avito.
func (c *Controller) convertProductsToAvito(products map[string]storage.Product) []avito.Product {
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
			Category:    "Коллекционирование",
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
