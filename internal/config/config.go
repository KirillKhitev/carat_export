package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
)

type Params struct {
	MoySkladUrl           string `json:"moy_sklad_url"`
	MoySkladLogin         string `json:"moy_sklad_login"`
	MoySkladPassword      string `json:"moy_sklad_password"`
	MoySkladInterval      int    `json:"moy_sklad_interval"`
	AvitoFilePath         string `json:"avito_filepath"`
	ImagesDir             string `json:"images_dir"`
	ImagesPath            string `json:"images_path"`
	ServerURL             string `json:"server_url"`
	LogLevel              string `json:"log_level"`
	LogDir                string `json:"log_dir"`
	ImageWorkers          int    `json:"image_workers"`
	ProductDescriptionAdd string `json:"product_description_add"`
	NeedDownloadProducts  bool   `json:"need_download_products"`
}

var Config Params = Params{}

const DefaultConfigPath = "config.json"

func (f *Params) Parse() error {
	c := &Params{}

	data, err := os.ReadFile(DefaultConfigPath)

	if err != nil {
		fmt.Println(err)
	} else {
		err = json.Unmarshal(data, c)

		if err != nil {
			return err
		}
	}

	flag.StringVar(&f.MoySkladUrl, "msa", c.MoySkladUrl, "МойСклад URL API")
	flag.StringVar(&f.MoySkladLogin, "msl", c.MoySkladLogin, "МойСклад логин")
	flag.StringVar(&f.MoySkladPassword, "msp", c.MoySkladPassword, "МойСклад пароль")
	flag.IntVar(&f.MoySkladInterval, "msi", c.MoySkladInterval, "МойСклад интервал забора товаров")
	flag.StringVar(&f.AvitoFilePath, "af", c.AvitoFilePath, "Avito путь до файла выгрузки")
	flag.StringVar(&f.ImagesDir, "id", c.ImagesDir, "Путь до папки изображений")
	flag.StringVar(&f.ImagesPath, "ip", c.ImagesPath, "Url изображений")
	flag.StringVar(&f.ServerURL, "iu", c.ServerURL, "Url сервера")
	flag.StringVar(&f.LogLevel, "ll", c.LogLevel, "Уровень логирования")
	flag.StringVar(&f.LogDir, "ld", c.LogDir, "Путь до папки логов")
	flag.StringVar(&f.ProductDescriptionAdd, "da", c.ProductDescriptionAdd, "Дополнительное описание товара")
	flag.IntVar(&f.ImageWorkers, "iw", c.ImageWorkers, "Количество потоков для скачивания изображений")
	flag.BoolVar(&f.NeedDownloadProducts, "nd", c.NeedDownloadProducts, "Начинать ли выгрузку при запуске")
	flag.Parse()

	if envMoySkladUrl := os.Getenv(`MOYSKLAD_URL`); envMoySkladUrl != `` {
		f.MoySkladUrl = envMoySkladUrl
	}

	if envMoySkladLogin := os.Getenv("MOYSKLAD_LOGIN"); envMoySkladLogin != "" {
		f.MoySkladLogin = envMoySkladLogin
	}

	if envMoySkladPassword := os.Getenv("MOYSKLAD_PASSWORD"); envMoySkladPassword != "" {
		f.MoySkladPassword = envMoySkladPassword
	}

	if envMoySkladInterval := os.Getenv("MOYSKLAD_INTERVAL"); envMoySkladInterval != "" {
		if val, err := strconv.Atoi(envMoySkladInterval); err == nil {
			f.MoySkladInterval = val
		} else {
			return fmt.Errorf("wrong value environment MOYSKLAD_INTERVAL: %s", envMoySkladInterval)
		}
	}

	if envAvitoFilePath := os.Getenv("AVITO_FILEPATH"); envAvitoFilePath != "" {
		f.AvitoFilePath = envAvitoFilePath
	}

	if envImagesDir := os.Getenv(`IMAGES_DIR`); envImagesDir != `` {
		f.ImagesDir = envImagesDir
	}

	if envImagesPath := os.Getenv(`IMAGES_PATH`); envImagesPath != `` {
		f.ImagesPath = envImagesPath
	}

	if envServerURL := os.Getenv(`SERVER_URL`); envServerURL != `` {
		f.ServerURL = envServerURL
	}

	if envLogLevel := os.Getenv(`LOG_LEVEL`); envLogLevel != `` {
		f.LogLevel = envLogLevel
	}

	if envLogDir := os.Getenv(`LOG_DIR`); envLogDir != `` {
		f.LogDir = envLogDir
	}

	if envImageWorkers := os.Getenv("IMAGE_WORKERS"); envImageWorkers != "" {
		if val, err := strconv.Atoi(envImageWorkers); err == nil {
			f.ImageWorkers = val
		} else {
			return fmt.Errorf("неверное значение переменной среды IMAGE_WORKERS: %s", envImageWorkers)
		}
	}

	if envProductDescriptionAdd := os.Getenv(`PRODUCT_DESCRIPTION_ADD`); envProductDescriptionAdd != `` {
		f.ProductDescriptionAdd = envProductDescriptionAdd
	}

	if f.MoySkladUrl == "" {
		return fmt.Errorf("Пустой МойСклад URL API")
	}

	if f.MoySkladLogin == "" {
		return fmt.Errorf("Пустой МойСклад Логин")
	}

	if f.MoySkladPassword == "" {
		return fmt.Errorf("Пустой МойСклад Пароль")
	}

	return nil
}

func (f *Params) String() string {
	r, _ := json.Marshal(f)

	return string(r)
}
