package accounts

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/dakom/basic-site-api/lib/auth"
	"github.com/dakom/basic-site-api/lib/auth/jwt_scopes"
	"github.com/dakom/basic-site-api/lib/datastore"
	"github.com/dakom/basic-site-api/lib/email"
	"github.com/dakom/basic-site-api/lib/pages"
	"github.com/dakom/basic-site-api/setup/config/static/pagenames"
	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"
	"golang.org/x/net/context"
	"google.golang.org/appengine/taskqueue"
)

func GotActivateRequest(rData *pages.RequestData) {

	if rData.UserRecord.GetData().IsActive {
		rData.SetJsonErrorCodeResponse(statuscodes.ACTIVATION_EXISTS)
		return
	} else {

		err := ActivateUser(rData.Ctx, rData.UserRecord)
		if err != nil {
			rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
			return
		}

		params := url.Values{}
		params.Set("uid", strconv.FormatInt(rData.UserRecord.GetKey().IntID(), 10))

		mailingListTask := taskqueue.NewPOSTTask("/"+pagenames.MAILINGLIST_SUBSCRIBE_WEBHOOK, params)
		_, err = taskqueue.Add(rData.Ctx, mailingListTask, rData.SiteConfig.TASKQUEUE_MAILINGLIST)

		if err != nil {
			rData.LogError("%v", err)
		}

		rData.SetJsonSuccessCodeResponse(statuscodes.ACTIVATION_COMPLETED)
		rData.DeleteJwtWhenFinished = true
	}

}

func SendActivateTokenRequest(rData *pages.RequestData) {

	username := strings.ToLower(strings.TrimSpace(rData.HttpRequest.FormValue("uname")))
	if len(username) < 1 {
		rData.SetJsonErrorCodeResponse(statuscodes.MISSING_USERNAME)
		return
	}

	userRecord, err := GetUserRecordViaUsername(rData.Ctx, username)
	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}
	if userRecord == nil {
		rData.SetJsonErrorCodeResponse(statuscodes.NOUSERNAME)
		return
	}

	if userRecord.GetData().IsActive {
		rData.SetJsonErrorCodeResponse(statuscodes.ALREADY_ACTIVE)
		return
	}

	_, jwtString, err := auth.GetNewUserOobJWT(rData, userRecord, jwt_scopes.OOB_USER_ACTIVATE, nil)

	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	url := rData.SiteConfig.EMAIL_TARGET_HOSTNAME + pagenames.APP_PAGE_ACCOUNT_ACTION_ACTIVATE + "/" + jwtString

	emailMessage := email.GetEmailActivationMessage(rData.HttpRequest.FormValue("locale"), url)
	err = email.Send(rData, userRecord.GetFullName(), userRecord.GetData().Email, emailMessage)

	if err != nil {
		rData.LogError(err.Error())
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	rData.SetJsonSuccessCodeResponse(statuscodes.CHECK_EMAIL)
}

func ActivateUser(c context.Context, userRecord *datastore.UserRecord) error {
	userRecord.GetData().IsActive = true

	err := datastore.Save(c, userRecord)
	if err != nil {
		return err
	}
	//add to search db, errors aren't critical but should be investigated by backend
	userRecord.AddToSearch(c)
	return nil
}
