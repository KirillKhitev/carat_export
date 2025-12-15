package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
)

type Params struct {
	MoySkladUrl           string  `json:"moy_sklad_url"`
	MoySkladLogin         string  `json:"moy_sklad_login"`
	MoySkladPassword      string  `json:"moy_sklad_password"`
	MoySkladInterval      int     `json:"moy_sklad_interval"`
	AvitoFilePath         string  `json:"avito_filepath"`
	AvitoClickCost        int     `json:"avito_click_cost"`
	AvitoDailyLimit       float64 `json:"avito_daily_limit"`
	ImagesDir             string  `json:"images_dir"`
	ImagesPath            string  `json:"images_path"`
	ServerURL             string  `json:"server_url"`
	LogLevel              string  `json:"log_level"`
	LogDir                string  `json:"log_dir"`
	ImageWorkers          int     `json:"image_workers"`
	ProductDescriptionAdd string  `json:"product_description_add"`
	NeedDownloadProducts  bool    `json:"need_download_products"`
	ImportVKWorkers       int     `json:"vk_workers"`
	MoySkladUpdateVKIDs   bool    `json:"moy_sklad_update_vk_ids"`
	VkToken               string  `json:"vk_access_token"`
	VkGroupID             int     `json:"vk_group_id"`
	VkClientID            int     `json:"vk_client_id"`
	VkCategoryID          int     `json:"vk_category_id"`
	VKRefreshToken        string  `json:"vk_refresh_token"`
	VKDeviceID            string  `json:"vk_device_id"`
	VKServiceKey          string  `json:"vk_service_key"`
	VKSecretKey           string  `json:"vk_secret_key"`
	EmailAppPassword      string  `json:"email_app_password"`
	EmailAppLogin         string  `json:"email_app_login"`
	EmailToAddr           string  `json:"email_to_addr"`
	EmailCcAddr           string  `json:"email_cc_addr"`
	EmailFrom             string  `json:"email_from"`
	EmailSubject          string  `json:"email_subject"`
	EmailSmtp             string  `json:"email_smtp"`
	EmailSmtpPort         string  `json:"email_smtp_port"`
	YMarketCategoryID     int     `json:"ymarket_category_id"`
	YMarketFilepath       string  `json:"ymarket_filepath"`
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
	flag.IntVar(&f.ImportVKWorkers, "vkw", c.ImportVKWorkers, "Количество потоков для импорта в VK")
	flag.BoolVar(&f.MoySkladUpdateVKIDs, "uvi", c.MoySkladUpdateVKIDs, "Обновить VK ID товаров в МойСклад-е")
	flag.StringVar(&f.VkToken, "vkt", c.VkToken, "VK API токен")
	flag.IntVar(&f.VkGroupID, "vgi", c.VkGroupID, "VK Group ID")
	flag.IntVar(&f.VkClientID, "vcli", c.VkClientID, "VK Client ID")
	flag.IntVar(&f.VkCategoryID, "vci", c.VkCategoryID, "VK Category ID")
	flag.StringVar(&f.VKRefreshToken, "vrt", c.VKRefreshToken, "VK Refresh Token")
	flag.StringVar(&f.VKDeviceID, "vdi", c.VKDeviceID, "VK Device ID")
	flag.StringVar(&f.VKServiceKey, "vservk", c.VKServiceKey, "VK Service key")
	flag.StringVar(&f.VKSecretKey, "vseck", c.VKSecretKey, "VK Secret key")
	flag.StringVar(&f.EmailAppPassword, "eap", c.EmailAppPassword, "Email App Password")
	flag.StringVar(&f.EmailFrom, "ef", c.EmailFrom, "Email From")
	flag.StringVar(&f.EmailSubject, "es", c.EmailSubject, "Email Subject")
	flag.StringVar(&f.EmailSmtp, "esm", c.EmailSmtp, "Email SMTP Host")
	flag.StringVar(&f.EmailSmtpPort, "esmp", c.EmailSmtpPort, "Email SMTP Port")
	flag.Parse()

	f.EmailToAddr = c.EmailToAddr
	f.EmailCcAddr = c.EmailCcAddr
	f.EmailAppLogin = c.EmailAppLogin

	f.AvitoClickCost = c.AvitoClickCost
	f.AvitoDailyLimit = c.AvitoDailyLimit
	f.YMarketCategoryID = c.YMarketCategoryID
	f.YMarketFilepath = c.YMarketFilepath

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

	if envImportVKWorkers := os.Getenv("VK_WORKERS"); envImportVKWorkers != "" {
		if val, err := strconv.Atoi(envImportVKWorkers); err == nil {
			f.ImportVKWorkers = val
		} else {
			return fmt.Errorf("неверное значение переменной среды VK_WORKERS: %s", envImportVKWorkers)
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
