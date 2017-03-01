package accounts

import (
	"strconv"
	"strings"

	"github.com/dakom/basic-site-api/setup/config/static/pagenames"
	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"
	"github.com/dakom/basic-site-api/lib/auth"
	"github.com/dakom/basic-site-api/lib/auth/jwt_scopes"
	"github.com/dakom/basic-site-api/lib/datastore"
	"github.com/dakom/basic-site-api/lib/email"
	"github.com/dakom/basic-site-api/lib/pages"
	"github.com/dakom/basic-site-api/lib/utils/cipher"
)

func GotChangePasswordTokenRequestBySession(rData *pages.RequestData) {
	sendChangePasswordToken(rData, rData.UserRecord)
}

func ForgotPasswordByUsername(rData *pages.RequestData) {
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

	if !userRecord.GetData().IsActive {
		rData.SetJsonErrorCodeWithDataResponse(statuscodes.NOT_ACTIVATED, map[string]interface{}{
			"uid":   rData.UserRecord.GetKeyIntAsString(),
			"email": rData.UserRecord.GetData().Email,
			"fname": rData.UserRecord.GetData().FirstName,
			"lname": rData.UserRecord.GetData().LastName,
			"avid":  strconv.FormatInt(rData.UserRecord.GetData().AvatarId, 10),
		})

		return
	}

	sendChangePasswordToken(rData, userRecord)
}

func sendChangePasswordToken(rData *pages.RequestData, userRecord *datastore.UserRecord) {
	var parentRecord *datastore.UserRecord
	emailAddress := userRecord.GetData().Email

	_, jwtString, err := auth.GetNewUserOobJWT(rData, userRecord, jwt_scopes.OOB_USER_PASSWORD_CHANGE|jwt_scopes.ACCOUNT_READ, nil)

	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	if userRecord.GetData().ParentId != 0 {

		parentRecord, err = GetUserRecordViaKey(rData.Ctx, userRecord.GetData().ParentId)
		if err != nil {
			rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
			return
		} else if parentRecord == nil {
			rData.SetJsonErrorCodeResponse(statuscodes.MISSINGINFO)
			return
		}

		emailAddress = parentRecord.GetData().Email

	}

	url := rData.SiteConfig.EMAIL_TARGET_HOSTNAME + pagenames.APP_PAGE_ACCOUNT_ACTION_PASSWORD_RESET + "/" + jwtString
	//check if userRecord.IsChild() and then get record of parent to actually get email address....

	emailMessage := email.GetEmailChangePasswordMessage(rData.HttpRequest.FormValue("locale"), url)

	err = email.Send(rData, userRecord.GetFullName(), emailAddress, emailMessage)

	if err != nil {
		rData.LogError(err.Error())
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	rData.SetJsonSuccessCodeResponse(statuscodes.CHECK_EMAIL)
}

func GotChangePasswordActionRequest(rData *pages.RequestData) {

	password := rData.HttpRequest.FormValue("pw")
	if len(password) < 6 || len(password) > 32 {
		rData.SetJsonErrorCodeResponse(statuscodes.INVALID_PASSWORD)
		return
	}

	passwordHash, err := cipher.NewPWHash(password, nil)
	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	rData.UserRecord.GetData().Password = passwordHash

	err = datastore.Save(rData.Ctx, rData.UserRecord)
	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	rData.SetJsonSuccessResponse(nil)
	rData.DeleteJwtWhenFinished = true
}
