package webhooks

import (
	"image"
	"io"
	"strconv"

	"github.com/dakom/basic-site-api/endpoints/accounts"
	"github.com/dakom/basic-site-api/lib/datastore"
	"github.com/dakom/basic-site-api/lib/email"
	"github.com/dakom/basic-site-api/lib/pages"

	"google.golang.org/appengine/urlfetch"
)

func AvatarPull(rData *pages.RequestData) {

	var userRecord datastore.UserRecord

	intID, err := strconv.ParseInt(rData.HttpRequest.FormValue("uid"), 10, 64)
	if err == nil {
		err = datastore.LoadFromKey(rData.Ctx, &userRecord, intID)
	}

	if err != nil {
		rData.SetHttpStatusResponse(400, err.Error())
		return
	}

	aurl := rData.HttpRequest.FormValue("aurl")
	if aurl == "" {
		return
	}

	client := urlfetch.Client(rData.Ctx)
	resp, err := client.Get(aurl)
	if err != nil {
		rData.SetHttpStatusResponse(400, err.Error())
		return
	}
	defer resp.Body.Close()

	//Read in full image
	srcImage, _, err := image.Decode(io.LimitReader(resp.Body, rData.SiteConfig.MAX_READ_SIZE))
	if err != nil {
		rData.SetHttpStatusResponse(400, err.Error())
		return
	}

	accounts.UpdateAvatar(rData, srcImage, &userRecord)

}

func MailingListSubscribe(rData *pages.RequestData) {
	var userRecord datastore.UserRecord

	if intID, err := strconv.ParseInt(rData.HttpRequest.FormValue("uid"), 10, 64); err != nil {
		rData.SetHttpStatusResponse(400, err.Error())
		return
	} else if err := datastore.LoadFromKey(rData.Ctx, &userRecord, intID); err != nil {
		rData.SetHttpStatusResponse(400, err.Error())
		return
	}

	if err := email.MailingListSubscribe(rData, &userRecord); err != nil {

		rData.SetHttpStatusResponse(400, err.Error())
		return
	}

	if rData.HttpRequest.FormValue("aurl") != "" {
		AvatarPull(rData)
	}
}

func MailingListUpdateEmail(rData *pages.RequestData) {
	MailingListUpdate(rData, "email")
}
func MailingListUpdateName(rData *pages.RequestData) {
	MailingListUpdate(rData, "name")
}

func MailingListUpdate(rData *pages.RequestData, updateType string) {
	var userRecord datastore.UserRecord
	if intID, err := strconv.ParseInt(rData.HttpRequest.FormValue("uid"), 10, 64); err != nil {
		rData.SetHttpStatusResponse(400, err.Error())
		return
	} else if err := datastore.LoadFromKey(rData.Ctx, &userRecord, intID); err != nil {
		rData.SetHttpStatusResponse(400, err.Error())
		return
	}

	if err := email.MailingListUpdate(rData, &userRecord, updateType); err != nil {

		rData.SetHttpStatusResponse(400, err.Error())
		return
	}
}
