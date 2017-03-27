package auth

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/dakom/basic-site-api/lib/auth/jwt_scopes"
	"github.com/dakom/basic-site-api/lib/datastore"
	"github.com/dakom/basic-site-api/lib/pages"
	"github.com/dakom/basic-site-api/lib/utils/text"
	"github.com/dakom/basic-site-api/setup/config/static/pagenames"
	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"

	"github.com/dgrijalva/jwt-go"
)

const (
	//descripes how the userid is used
	JWT_USERTYPE_USER_RECORD string = "usr"
	JWT_USERTYPE_SYSTEM_ID   string = "sys"

	JWT_DURATION_LONG  int64 = 604800
	JWT_DURATION_SHORT int64 = 3600
	JWT_DURATION_NEVER int64 = -1

	REQUEST_SOURCE_APPENGINE_TASK string = "appengine-task"

	JWT_AUDIENCE_COOKIE string = "cookie" //will vet cookie / header, does not necessarily vet against db
	JWT_AUDIENCE_APP    string = "app"    //for app usual usage, does not necessarily vet against db
	JWT_AUDIENCE_OOB    string = "oob"    //for passing around via email, page embeds, etc. - always vets against db
)

const (
	_ = iota
	SYSTEM_ID_OAUTH
)

func ValidateUserType(rData *pages.RequestData, jwtRecord *datastore.JwtRecord) (bool, interface{}) {

	if jwtRecord != nil {

		if jwtRecord.GetData().UserType == JWT_USERTYPE_USER_RECORD {
			var userRecord datastore.UserRecord

			if err := datastore.LoadFromKey(rData.Ctx, &userRecord, jwtRecord.GetData().UserId); err == nil {

				return true, &userRecord
			}
		} else if jwtRecord.GetData().UserType == JWT_USERTYPE_SYSTEM_ID {
			if jwtRecord.GetData().UserId > 0 {
				return true, jwtRecord.GetData().UserId
			}
		}
	}

	return false, nil
}

//returns isValid and hasRefreshed
//hasRefreshed is returned instead of mixed in here since the response type might be html, cookies, etc.

func ValidatePageRequest(rData *pages.RequestData) (bool, bool) {
	var isValid, dbIsValid, hasRefreshed, isExpired, validatedUserType bool
	var dbRecord *datastore.JwtRecord

	rData.JwtString = getJwtStringFromRequest(rData)

	rData.JwtRecord, isExpired = GetJwtFromString(rData, rData.JwtString, false) //for the case of validating a page request, only check db below based on scope logic etc

	rData.UserRecord = nil

	//even if the jwt will ultimately be invalid, let's set the user info if it's available
	if ok, iface := ValidateUserType(rData, rData.JwtRecord); ok {
		validatedUserType = true
		if rData.JwtRecord.GetData().UserType == JWT_USERTYPE_USER_RECORD {
			rData.UserRecord = iface.(*datastore.UserRecord)
		}
	}

	//check against Roles - only one must be fulfilled
	if rData.PageConfig.Roles != 0 {
		if rData.UserRecord == nil {
			goto fail
		}

		if (rData.PageConfig.Roles & uint64(rData.UserRecord.GetData().Roles)) == 0 {

			goto fail
		}

	}

	//check for internal auth stuff

	if rData.PageConfig.RequestSource == REQUEST_SOURCE_APPENGINE_TASK {
		if len(rData.HttpRequest.Header.Get("X-AppEngine-QueueName")) == 0 {

			goto fail
		}
	}

	if rData.PageConfig.RequestSource == rData.SiteConfig.REQUEST_SOURCE_APPENGINE_APPID {
		if rData.HttpRequest.Header.Get("X-Appengine-Inbound-Appid") != rData.SiteConfig.REQUEST_SOURCE_APPENGINE_APPID {
			goto fail
		}
	}

	//token exists and is signed properly... but maybe it's expired and needs a refresh or requires additional check against db
	//failure here resets jwtMap to nil, i.e. as though no valid one were ever supplied
	if rData.JwtRecord != nil {

		if isExpired {

			dbRecord, dbIsValid = GetJwtFromDb(rData, rData.JwtRecord.GetKey())
			if !dbIsValid {

				rData.JwtRecord = nil
			} else {
				rData.JwtRecord = dbRecord
				rData.JwtRecord.GetData().ExpiresAt = time.Now().Add(time.Duration(JWT_DURATION_SHORT) * time.Second).Unix()

				if updatedJwtString, err := SignJwt(rData.Ctx, rData.JwtRecord); err != nil {
					rData.JwtRecord = nil
				} else {
					// no need to save the new one to the db, unless it's near Final Expires time (see below)
					rData.JwtString = updatedJwtString
					hasRefreshed = true
				}
			}
		} else if rData.PageConfig.RequiresDBScopeCheck || rData.JwtRecord.GetData().Audience == JWT_AUDIENCE_OOB {
			//only needs to check if jwt is not expired since an expired jwt validates against the database already
			dbRecord, dbIsValid = GetJwtFromDb(rData, rData.JwtRecord.GetKey())
			if !dbIsValid {

				rData.JwtRecord = nil
			} else {

				rData.JwtRecord = dbRecord
			}

		}
	}

	//validate scopes!
	if rData.PageConfig.Scopes != 0 && !rData.SiteConfig.SUSPEND_AUTH {

		if rData.JwtRecord == nil {
			goto fail
		}
		//non existant user where page scope requires one is not okay.
		if !validatedUserType {

			goto fail
		}
		//inactive user for anything other than activate request is not ok
		if rData.UserRecord != nil && !rData.UserRecord.GetData().IsActive && rData.PageConfig.PageName != pagenames.ACCOUNT_ACTIVATE_SERVICE {
			goto fail
		}

		if rData.SiteConfig.SKIP_CSRF_CHECK != true && rData.PageConfig.SkipCsrfCheck != true && rData.JwtRecord.GetData().Audience == JWT_AUDIENCE_COOKIE && (rData.JwtRecord.GetData().SessionId == "" || rData.HttpRequest.Header.Get(rData.SiteConfig.JWT_HEADER_SID_NAME) == "" || rData.JwtRecord.GetData().SessionId != rData.HttpRequest.Header.Get(rData.SiteConfig.JWT_HEADER_SID_NAME)) {
			//when jwt is from web view, it must validate against the session id set in the *header*, to prevent against csrf attacks
			//exception is if scope is only PAGE_READ
			rData.LogInfo("FAIL HERE! %s %s: %s", rData.JwtRecord.GetData().SessionId, rData.SiteConfig.JWT_HEADER_SID_NAME, rData.HttpRequest.Header.Get(rData.SiteConfig.JWT_HEADER_SID_NAME))
			goto fail
		}

		if rData.PageConfig.AcceptAnyScope {
			//accept scope if it's anywhere in the config and jwt
			if (rData.PageConfig.Scopes & uint64(rData.JwtRecord.GetData().Scopes)) == 0 {

				goto fail
			}

		} else {
			//all page scopes must be satisfied!
			if (rData.PageConfig.Scopes & uint64(rData.JwtRecord.GetData().Scopes)) != rData.PageConfig.Scopes {

				goto fail
			}
		}

	}

	//user info is ok if it was needed- validate page scope for request!!!

	//all is validated... if we grabbed the rData.JwtRecord at some point (check against db), might as well update long expirey here
	//f it's too close (i.e. time remaining is less than half of original duration)... lets sessions last longer while active
	if rData.JwtRecord != nil {
		durationByAudience := GetFinalDurationByAudience(rData.JwtRecord.GetData().Audience)

		if durationByAudience != JWT_DURATION_NEVER {
			finalExpireDiff := (rData.JwtRecord.GetData().FinalExpires - time.Now().Unix())
			if finalExpireDiff < durationByAudience/2 {
				rData.JwtRecord.GetData().FinalExpires = time.Now().Add(time.Duration(durationByAudience) * time.Second).Unix()
				datastore.Save(rData.Ctx, rData.JwtRecord) //we don't care about errors here
			}
		}
	}

	//success
	isValid = true
	goto complete

fail:

	rData.JwtRecord = nil
	rData.JwtString = ""
	rData.UserRecord = nil
	isValid = false

complete:
	return isValid, hasRefreshed
}

func SignJwt(ctx context.Context, jwtRecord *datastore.JwtRecord) (string, error) {
	if jwtRecord.GetData().Audience == JWT_AUDIENCE_COOKIE && jwtRecord.GetData().SessionId == "" {
		return "", fmt.Errorf(statuscodes.MISSINGINFO)
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod("AppEngine"), jwtRecord.GetData())

	jwtString, err := token.SignedString(ctx)

	if err != nil {
		return "", err
	}

	return jwtString, nil
}

func SetJWTCookie(rData *pages.RequestData, jwtString string, sid string, duration int) {
	if rData.HttpWriter != nil {
		rData.LogInfo("SETTING COOKIE! %s", rData.SiteConfig.COOKIE_DOMAIN)
		http.SetCookie(rData.HttpWriter, &http.Cookie{Name: rData.SiteConfig.JWT_COOKIE_NAME, Value: jwtString, MaxAge: duration, HttpOnly: rData.SiteConfig.COOKIE_SECURE, Secure: rData.SiteConfig.COOKIE_SECURE, Path: "/", Domain: rData.SiteConfig.COOKIE_DOMAIN})
	}
}
func UnsetJWTCookie(rData *pages.RequestData) {
	if rData.HttpWriter != nil {
		rData.LogInfo("DELETING COOKIE! %s", rData.SiteConfig.COOKIE_DOMAIN)
		http.SetCookie(rData.HttpWriter, &http.Cookie{Name: rData.SiteConfig.JWT_COOKIE_NAME, Value: "", MaxAge: -1, Secure: rData.SiteConfig.COOKIE_SECURE, Path: "/", Domain: rData.SiteConfig.COOKIE_DOMAIN})
	}

}

func GetNewLoginJWT(rData *pages.RequestData, userRecord *datastore.UserRecord, audience string) (*datastore.JwtRecord, string, error) {
	var scopes int64
	var sid string
	var err error

	if audience != JWT_AUDIENCE_APP && audience != JWT_AUDIENCE_COOKIE {
		return nil, "", fmt.Errorf(statuscodes.MISSINGINFO)
	}

	if userRecord.GetData().ParentId == 0 {
		scopes = jwt_scopes.ACCOUNT_FULL_MASTER
	} else {
		scopes = jwt_scopes.ACCOUNT_FULL_SUB
	}

	if audience == JWT_AUDIENCE_COOKIE {
		sid, err = text.RandomHexString(12)
		if err != nil {
			return nil, "", err
		}
	}

	if scopes == 0 {
		return nil, "", fmt.Errorf(statuscodes.MISSINGINFO)
	}
	return makeNewJwtFromInfo(rData, userRecord.GetKey().IntID(), JWT_USERTYPE_USER_RECORD, scopes, audience, sid, "", "")
}

func GetNewUserOobJWT(rData *pages.RequestData, userRecord *datastore.UserRecord, scopes int64, extraMap map[string]interface{}) (*datastore.JwtRecord, string, error) {

	if userRecord == nil {
		return nil, "", fmt.Errorf(statuscodes.MISSINGINFO)
	}

	var extra string

	if extraMap != nil {
		if extraString, err := text.MakeJsonString(extraMap); err != nil {
			return nil, "", fmt.Errorf(statuscodes.TECHNICAL)
		} else {
			extra = extraString
		}
	}

	return makeNewJwtFromInfo(rData, userRecord.GetKey().IntID(), JWT_USERTYPE_USER_RECORD, scopes, JWT_AUDIENCE_OOB, "", "", extra)
}

func GetNewSystemsOobJWT(rData *pages.RequestData, systemId int64, scopes int64, extra string) (*datastore.JwtRecord, string, error) {

	if systemId <= 0 {
		return nil, "", fmt.Errorf(statuscodes.MISSINGINFO)
	}

	return makeNewJwtFromInfo(rData, systemId, JWT_USERTYPE_SYSTEM_ID, scopes, JWT_AUDIENCE_OOB, "", "", extra)
}

func DestroyToken(rData *pages.RequestData) error {

	rData.HttpRequest.Header.Set("Authorization", "")
	UnsetJWTCookie(rData)

	if rData.JwtRecord != nil {
		return datastore.Delete(rData.Ctx, rData.JwtRecord)
	}

	return nil

}

func GetFinalDurationByAudience(audience string) int64 {
	switch audience {
	case JWT_AUDIENCE_APP:
		return JWT_DURATION_LONG
		//return JWT_DURATION_NEVER
	case JWT_AUDIENCE_COOKIE:
		return JWT_DURATION_SHORT
	default: //oob and system
		return JWT_DURATION_SHORT

	}
}

func GetJwtFromString(rData *pages.RequestData, jwtString string, forceDbCheck bool) (*datastore.JwtRecord, bool) {
	var isExpired bool
	var jwtRecord datastore.JwtRecord

	ctx := rData.Ctx

	//no jwtString at all? get outta here...
	if jwtString == "" {

		return nil, isExpired
	}

	parser := &jwt.Parser{
		UseJSONNumber: true,
		ValidMethods:  []string{"AppEngine"},
	}
	//validate the jwt string
	_, err := parser.ParseWithClaims(jwtString, jwtRecord.GetData(), func(token *jwt.Token) (interface{}, error) {

		return ctx, nil
	})

	//an error occured with validation
	if err != nil {

		if tokenValidationError, ok := err.(*jwt.ValidationError); ok {
			if (tokenValidationError.Errors ^ jwt.ValidationErrorExpired) == 0 {
				//if the *only* error is expirey, then we still want to process the claims so we can re-use them for a refresh token
				isExpired = true
			}
		}

		//otherwise, it's some other error - like bad signing etc and we get outta here now as though there's no jwt at all...
		if !isExpired {
			return nil, isExpired
		}
	}

	// Set the record key... this also serves as a basic sanity check that the claims parsed okay
	// Note that setting the record key doesn't actually touch datastore, it's just a variable - i.e. we still haven't touched the db yet ;)
	// In other words, setting the key here simply lets us pass around the record *as though* we got it from the db
	if jwtId, err := strconv.ParseInt(jwtRecord.GetData().SelfId, 10, 64); err != nil {
		return nil, isExpired
	} else if jwtId == 0 {
		return nil, isExpired
	} else {
		datastore.SetKey(ctx, &jwtRecord, jwtId)
	}

	//if we're not checking the db here, then return what we've got!
	if forceDbCheck == false {
		return &jwtRecord, isExpired
	}

	//otherwise return the db validated record
	if dbRecord, dbIsValid := GetJwtFromDb(rData, jwtRecord.GetKey()); dbIsValid && dbRecord != nil {

		return dbRecord, isExpired
	} else {

		return nil, isExpired
	}
}

func GetJwtFromDb(rData *pages.RequestData, keyVal interface{}) (*datastore.JwtRecord, bool) {
	var jwtRecord datastore.JwtRecord

	err := datastore.LoadFromKey(rData.Ctx, &jwtRecord, keyVal)
	if err != nil {
		return &jwtRecord, false
	}

	if jwtRecord.GetData().FinalExpires == 0 {
		return &jwtRecord, false
	}

	if jwtRecord.GetData().FinalExpires != -1 && jwtRecord.GetData().FinalExpires < time.Now().Unix() {
		return &jwtRecord, false
	}

	return &jwtRecord, true
}

func makeNewJwtFromInfo(rData *pages.RequestData, userID int64, userType string, scopes int64, audience string, sid string, subject string, extra string) (*datastore.JwtRecord, string, error) {

	currentTime := time.Now().Unix()
	expirationTime := time.Now().Add(time.Duration(JWT_DURATION_SHORT) * time.Second).Unix()
	finalExpirationTime := GetFinalDurationByAudience(audience)
	if finalExpirationTime != -1 {
		finalExpirationTime = time.Now().Add(time.Duration(GetFinalDurationByAudience(audience)) * time.Second).Unix()
	}

	var jwtRecord datastore.JwtRecord

	data := &datastore.JwtData{
		Audience:     audience,
		UserId:       userID,
		UserType:     userType,
		ExpiresAt:    expirationTime,
		IssuedAt:     currentTime,
		Scopes:       scopes,
		SessionId:    sid,
		FinalExpires: finalExpirationTime,
		Subject:      subject,
		Extra:        extra,
	}
	jwtRecord.SetData(data)

	if err := datastore.SaveToAutoKey(rData.Ctx, &jwtRecord); err != nil {
		return nil, "", err
	}

	jwtString, err := SignJwt(rData.Ctx, &jwtRecord)
	if err != nil {
		return nil, "", err
	}

	return &jwtRecord, jwtString, nil
}

func getJwtStringFromRequest(rData *pages.RequestData) string {
	var jwtString string

	//Get the jwt from cookie, header, parameter. Cookie last since it's inherent and we want to be able to override
	if authHeader := rData.HttpRequest.Header.Get("Authorization"); authHeader != "" {
		// Should be a bearer token
		if len(authHeader) > 6 && strings.ToUpper(authHeader[0:7]) == "BEARER " {
			jwtString = authHeader[7:]
		}
	}

	if jwtString == "" {
		jwtString = rData.HttpRequest.FormValue("jwt")
	}

	if jwtString == "" {
		if jwtCookie, err := rData.HttpRequest.Cookie(rData.SiteConfig.JWT_COOKIE_NAME); err == nil {
			jwtString = jwtCookie.Value

			rData.LogInfo("GOT FROM COOKIE: %s", jwtString)
		}
	}

	return jwtString
}
