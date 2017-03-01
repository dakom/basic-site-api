// email
package email

import (
	"net/http"

	"github.com/dakom/basic-site-api/lib/pages"

	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/dakom/basic-site-api/lib/datastore"
	"google.golang.org/appengine/urlfetch"
)

type Message struct {
	Subject string
	Body    string
}

func Send(rData *pages.RequestData, toName string, toAddress string, msg *Message) error {
	if rData.SiteConfig.SUSPEND_EMAIL == true {
		return nil
	}
	from := mail.NewEmail(rData.SiteConfig.SENDGRID_FROM_NAME, rData.SiteConfig.SENDGRID_FROM_ADDR)
	to := mail.NewEmail(toName, toAddress)
	content := mail.NewContent("text/html", msg.Body)
	m := mail.NewV3MailInit(from, msg.Subject, to, content)

	request := sendgrid.GetRequest(rData.SiteConfig.SENDGRID_APIKEY, "/v3/mail/send", "https://api.sendgrid.com")
	client := rest.Client{&http.Client{
		Transport: &urlfetch.Transport{Context: rData.Ctx},
	}}

	request.Method = "POST"
	request.Body = mail.GetRequestBody(m)
	_, err := client.API(request)

	return err
}

func MailingListSubscribe(rData *pages.RequestData, userRecord *datastore.UserRecord) error {
	if rData.SiteConfig.SUSPEND_EMAIL == true {
		return nil
	}

	if rData.SiteConfig.MAILINGLIST_TYPE == "MAILCHIMP" {
		if response, err := mailchimpSubscribe(rData, userRecord); err != nil {
			return err
		} else {
			return processMailchimpResponse(rData, response, userRecord)
		}
	} else if rData.SiteConfig.MAILINGLIST_TYPE == "CONSTANTCONTACT" {
		if contactID, err := constantContactSubscribe(rData, userRecord); err != nil {
			return err
		} else {
			userRecord.GetData().UserMailinglistData.EmailId = contactID
			return datastore.Save(rData.Ctx, userRecord)
		}
	}

	return nil
}

func MailingListUpdate(rData *pages.RequestData, userRecord *datastore.UserRecord, updateType string) error {
	//update in search db, errors aren't critical but should be investigated by backend
	if err := userRecord.AddToSearch(rData.Ctx); err != nil {
		rData.LogError("%v", err)
	}

	if rData.SiteConfig.SUSPEND_EMAIL == true {
		return nil
	}

	if rData.SiteConfig.MAILINGLIST_TYPE == "MAILCHIMP" {
		if updateType == "email" {
			if response, err := mailchimpUpdateEmail(rData, userRecord); err != nil {
				return err
			} else {
				return processMailchimpResponse(rData, response, userRecord)
			}
		} else if updateType == "name" {
			if response, err := mailchimpUpdateName(rData, userRecord); err != nil {
				return err
			} else {
				return processMailchimpResponse(rData, response, userRecord)
			}
		}

	} else if rData.SiteConfig.MAILINGLIST_TYPE == "CONSTANTCONTACT" {
		if updateType == "email" {
			return constantContactUpdateEmail(rData, userRecord)
		} else if updateType == "name" {
			return constantContactUpdateName(rData, userRecord)
		}
	}

	return nil
}

func processMailchimpResponse(rData *pages.RequestData, response *MailchimpSuccessResponse, userRecord *datastore.UserRecord) error {
	userRecord.GetData().UserMailinglistData.EmailId = response.EmailId
	userRecord.GetData().UserMailinglistData.ListEmailId = response.ListEmailId

	return datastore.Save(rData.Ctx, userRecord)
}
