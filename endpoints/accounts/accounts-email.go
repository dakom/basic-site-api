package accounts

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/context"

	gaeds "google.golang.org/appengine/datastore"
	"google.golang.org/appengine/taskqueue"

	"github.com/asaskevich/govalidator"

	"github.com/dakom/basic-site-api/lib/auth"
	"github.com/dakom/basic-site-api/lib/auth/jwt_scopes"
	"github.com/dakom/basic-site-api/lib/datastore"
	"github.com/dakom/basic-site-api/lib/email"
	"github.com/dakom/basic-site-api/lib/pages"
	"github.com/dakom/basic-site-api/setup/config/static/pagenames"
	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"
)

func GotEmailChangeTokenRequest(rData *pages.RequestData) {

	emailAddress := strings.ToLower(strings.TrimSpace(rData.HttpRequest.FormValue("email")))

	if len(emailAddress) < 1 {
		rData.SetJsonErrorCodeResponse(statuscodes.MISSINGINFO)
		return
	}

	if !govalidator.IsEmail(emailAddress) || strings.HasPrefix(emailAddress, rData.SiteConfig.OAUTH_USERID_PREFIX) {
		rData.SetJsonErrorCodeResponse(statuscodes.INVALID_EMAIL)
		return
	}

	existingUserRecord, err := GetUserRecordViaUsername(rData.Ctx, emailAddress)
	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	if existingUserRecord != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.USERNAME_EXISTS)
		return
	}

	_, jwtString, err := auth.GetNewUserOobJWT(rData, rData.UserRecord, jwt_scopes.OOB_USER_EMAIL_CHANGE, map[string]interface{}{"email": emailAddress})

	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	url := rData.SiteConfig.EMAIL_TARGET_HOSTNAME + pagenames.APP_PAGE_ACCOUNT_ACTION_EMAIL_CHANGE + "/" + jwtString + appUrlParamsFromRequest(rData)

	emailMessage := email.GetEmailChangeEmailAddressMessage(rData.HttpRequest.FormValue("locale"), url)
	err = email.Send(rData, rData.UserRecord.GetFullName(), emailAddress, emailMessage)

	if err != nil {
		rData.LogError(err.Error())
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	rData.SetJsonSuccessCodeResponse(statuscodes.CHECK_EMAIL)

}

func GotEmailChangeActionRequest(rData *pages.RequestData) {
	var emailAddress string
	var emailData map[string]string

	if rData.JwtRecord == nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	if err := json.Unmarshal([]byte(rData.JwtRecord.GetData().Extra), &emailData); err == nil {

		if tmpEmailAddress, ok := emailData["email"]; ok {
			emailAddress = tmpEmailAddress
		}
	} else {
		rData.LogInfo(err.Error())
	}

	rData.LogInfo("EMAIL ADDRESS: %v", emailData)

	if emailAddress == "" {
		rData.SetJsonErrorCodeResponse(statuscodes.MISSINGINFO)
		return
	}

	existingUserRecord, err := GetUserRecordViaUsername(rData.Ctx, emailAddress)
	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	if existingUserRecord != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.USERNAME_EXISTS)
		return

	}

	opts := gaeds.TransactionOptions{
		XG: true,
	}

	err = gaeds.RunInTransaction(rData.Ctx, func(c context.Context) error {
		//delete the old lookup record for this email if it exists
		var lookupRecord datastore.UsernameLookupRecord

		err = datastore.LoadFromKey(rData.Ctx, &lookupRecord, rData.UserRecord.GetData().Email)
		if err == nil {

			err = datastore.Delete(c, &lookupRecord)
			if err != nil {
				return err
			}
		}

		//update user's email address
		rData.UserRecord.GetData().Email = emailAddress

		err = datastore.Save(rData.Ctx, rData.UserRecord)
		if err != nil {
			return err
		}

		//create new lookup record for this email address
		var newLookupRecord datastore.UsernameLookupRecord
		newLookupRecord.GetData().UserId = rData.UserRecord.GetKey().IntID()

		err = datastore.SaveToKey(c, &newLookupRecord, emailAddress)
		if err != nil {
			return err
		}

		params := url.Values{}
		params.Set("uid", strconv.FormatInt(rData.UserRecord.GetKey().IntID(), 10))

		params.Set("locale", rData.HttpRequest.FormValue("locale"))
		mailingListTask := taskqueue.NewPOSTTask("/"+pagenames.MAILINGLIST_UPDATE_EMAIL_WEBHOOK, params)
		_, err = taskqueue.Add(rData.Ctx, mailingListTask, rData.SiteConfig.TASKQUEUE_MAILINGLIST)

		if err != nil {
			rData.LogError("TaskQueue non-critical (mailing list) error %v", err)
		}

		return nil
	}, &opts)

	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	rData.SetJsonSuccessCodeResponse(statuscodes.EMAIL_CHANGED)

	rData.DeleteJwtWhenFinished = true

}
