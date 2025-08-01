package email

import (
	"fmt"
	"github.com/KirillKhitev/carat_export/internal/config"
	"github.com/KirillKhitev/carat_export/internal/logger"
	"github.com/KirillKhitev/carat_export/internal/statistic"
	"github.com/KirillKhitev/carat_export/internal/storage"
	em "github.com/jordan-wright/email"
	"net/smtp"
)

func Notify(list map[string]storage.Product, statistics map[string]map[string][]statistic.StatisticRow) {
	emailInstance := em.NewEmail()
	emailInstance.From = config.Config.EmailFrom
	emailInstance.To = []string{config.Config.EmailToAddr}
	emailInstance.Bcc = []string{}
	emailInstance.Cc = []string{}
	emailInstance.Subject = config.Config.EmailSubject
	emailInstance.HTML = prepareEmailBody(list, statistics)

	addr := fmt.Sprintf("%s:%s", config.Config.EmailSmtp, config.Config.EmailSmtpPort)
	auth := smtp.PlainAuth("", config.Config.EmailToAddr, config.Config.EmailAppPassword, config.Config.EmailSmtp)
	err := emailInstance.Send(addr, auth)

	if err != nil {
		logger.Log.Errorf("email notify failed: %s", err)
		return
	}

	logger.Log.Info("email notify success")
}

func prepareEmailBody(list map[string]storage.Product, statistics map[string]map[string][]statistic.StatisticRow) []byte {
	result := `<html>
	<head>
		<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
		<title>Hello Gophers!</title>
	</head>
	<body>
		<table width=100% cellpadding=10 cellspacing=10>`

	for _, product := range list {
		bColor := "green"
		for _, row := range statistics["VK"][product.ID] {
			if row.Status == statistic.STATUS_ERROR {
				bColor = "red"
				break
			}
		}

		result += `<tr>`
		result += fmt.Sprintf(`<td style="border: 2px solid %s"`, bColor)
		result += `<h4 style="font-size:16px;">` + product.Name + `</h4>`
		result += fmt.Sprintf(`<p>ID: <b>%s</b></p>`, product.ID)
		result += fmt.Sprintf(`<p>Артикул: <b>%s</b></p>`, product.Article)
		result += fmt.Sprintf(`<p>Цена: <b>%d</b></p>`, product.Price)
		result += fmt.Sprintf(`<p>Доступно: <b>%.0f</b></p>`, product.Quantity)
		result += fmt.Sprintf(`<p>Описание: %s</p>`, product.Description)
		result += fmt.Sprintf(`<p>Подборка: <b>%s</b></p>`, product.PathName)
		result += fmt.Sprintf(`<p>Выгружать на Avito: <b>%s</b></p>`, prepareBoolValue(product.ExportAvito))
		result += fmt.Sprintf(`<p>Avito ID: <b>%s</b></p>`, product.AvitoId)
		result += fmt.Sprintf(`<p>Выгружать в VK: <b>%s</b></p>`, prepareBoolValue(product.ExportVK))
		result += fmt.Sprintf(`<p>VK ID: <b>%s</b></p>`, product.VKId)

		result += `<div style="margin-bottom: 10px; margin-top: 10px;"><b>Выгрузка в VK:</b></div>`

		for _, row := range statistics["VK"][product.ID] {
			bColor = "black"
			if row.Status == statistic.STATUS_ERROR {
				bColor = "red"
			}

			result += fmt.Sprintf(`<p style="margin-bottom: 5px; margin-left: 25px; border: 2px solid %s; padding: 5px;">%s</p>`, bColor, row.Message)
		}

		result += `</td>`
		result += `</tr>`
	}

	result += `</table>
	</body>
</html>`

	return []byte(result)
}

func prepareBoolValue(v bool) string {
	if v {
		return "Да"
	}

	return "Нет"
}
