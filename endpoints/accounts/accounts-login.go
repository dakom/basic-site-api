package accounts

import (
	"errors"
	"strconv"
	"strings"

	"github.com/dakom/basic-site-api/lib/auth"
	"github.com/dakom/basic-site-api/lib/datastore"
	"github.com/dakom/basic-site-api/lib/pages"
	"github.com/dakom/basic-site-api/lib/utils/cipher"
	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"
)

func GotLoginServiceRequest(rData *pages.RequestData) {

	//
	username := strings.ToLower(strings.TrimSpace(rData.HttpRequest.FormValue("uname")))
	password := rData.HttpRequest.FormValue("pw")
	audience := auth.JWT_AUDIENCE_APP //for web environments, we might want to allow setting this to cookie...

	userRecord, _, jwtString, err := DoLogin(rData, username, password, audience, LOOKUP_TYPE_USERNAME)

	if err != nil {

		if userRecord == nil {

			rData.SetJsonErrorCodeResponse(err.Error()) //nousername
		} else {
			rData.SetJsonErrorCodeWithDataResponse(err.Error(), pages.JsonMapGeneric{
				"uid":   userRecord.GetKeyIntAsString(),
				"email": userRecord.GetData().Email,
				"fname": userRecord.GetData().FirstName,
				"lname": userRecord.GetData().LastName,
				"avid":  strconv.FormatInt(userRecord.GetData().AvatarId, 10),
			})
		}

		return
	}

	rData.SetJsonSuccessResponse(pages.JsonMapGeneric{"jwt": jwtString})
}

func DoLogin(rData *pages.RequestData, username string, password string, audience string, lookupType int64) (*datastore.UserRecord, *datastore.JwtRecord, string, error) {

	if len(username) < 1 {
		return nil, nil, "", errors.New(statuscodes.MISSING_USERNAME)
	}

	if lookupType != LOOKUP_TYPE_OAUTH && strings.HasPrefix(username, rData.SiteConfig.OAUTH_USERID_PREFIX) {
		return nil, nil, "", errors.New(statuscodes.NOUSERNAME)
	}

	userRecord, err := GetUserRecordViaUsername(rData.Ctx, username)
	if err != nil {
		rData.LogError(err.Error())
		return nil, nil, "", errors.New(statuscodes.TECHNICAL)
	}

	if userRecord == nil {
		return nil, nil, "", errors.New(statuscodes.NOUSERNAME)

	}

	if !userRecord.GetData().IsActive {
		return userRecord, nil, "", errors.New(statuscodes.NOT_ACTIVATED)
	}

	if lookupType != LOOKUP_TYPE_OAUTH {
		if len(password) < 1 {
			return userRecord, nil, "", errors.New(statuscodes.MISSING_PASSWORD)
		}

		if !cipher.ComparePWHash(password, userRecord.GetData().Password) {
			return userRecord, nil, "", errors.New(statuscodes.WRONG_PASSWORD)
		}
	}

	jwtRecord, jwtString, err := auth.GetNewLoginJWT(rData, userRecord, audience)

	if err != nil {
		return userRecord, nil, "", errors.New(statuscodes.TECHNICAL)
	}

	if audience == auth.JWT_AUDIENCE_COOKIE {
		auth.SetJWTCookie(rData, jwtString, jwtRecord.GetData().SessionId, int(auth.GetFinalDurationByAudience(jwtRecord.GetData().Audience)))
	}

	return userRecord, jwtRecord, jwtString, nil
}
