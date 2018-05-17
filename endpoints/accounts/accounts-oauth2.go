
/*

Oauth flow is like this:

1. FROM CLIENT (app or browser) - OauthRequest()
		Gets and creates a jwt with:
			a. destination schema (vetted) - allows being either url or app destination when route finishes
			b. destination (vetted) - allows consistency of easily checking destination irregardless of schema differences
		 	c. provider name (vetted) - required for knowing how to get user data
			d. request (vetted) - required for knowing how to get user data
		Creates the third-party consent page url with the jwt as state param
		Returns that url
2. IN BROWSER - User opens that url and completes third-party oauth page (note - "state" is there, and that's fine since the user must know it, threat vector is malicious interceptor only)
3. IN BROWSER - Page is redirected to OauthResponse() from third party, given both "state" and "code" and whatever else provider provides
4. SERVERSIDE/REDIRECT - OauthResponse()
	authenticates the state (i.e. jwt signing)
	issues oauth request with code to get relevant data, and stores as Response field in state
	redirects to destination schema + destination with original jwt (not re-generated one, serves as a sanity check that token is still valid when used)
5. AT CLIENT - loads destination (in browser, via schema, etc.)
	Grabs jwt
	Calls OauthAction() with jwt
6. OauthAction()
	Protected by Jwt auth checking (scope)
	Loads info from db (requires db check)
	Uses State/Response and whatever request params to issue action
	Returns result
*/

package accounts

import (
	"errors"

	"github.com/dakom/basic-site-api/lib/auth/jwt_scopes"
	"github.com/dakom/basic-site-api/lib/datastore"
	"github.com/dakom/basic-site-api/lib/utils/slice"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/dakom/basic-site-api/lib/auth"
	"github.com/dakom/basic-site-api/lib/pages"
	"github.com/dakom/basic-site-api/setup/config/static/pagenames"
	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/google"
	goauth2 "google.golang.org/api/oauth2/v2"
)

var OAUTH_ALLOWED_REQUESTS = []string{"userinfo"}
var OAUTH_ALLOWED_PROVIDERS = []string{"google", "facebook", "clever"}
var OAUTH_ALLOWED_DESTINATIONS = []string{"oauth-action/login", "oauth-action/register"}

type StateInfo struct {
	Destination string `json:"dest"`
	Request     string `json:"request"`
	RequestMeta string `json:"requestMeta,omitempty" datastore:",noindex"`
	Scheme      string `json:"scheme"`
	Provider    string `json:"provider"`
	Response    string `json:"response,omitempty" datastore:",noindex"`
}

type RegisterRequestMeta struct {
	Audience   string `json:"aud"`
	Terms      bool   `json:"terms"`
	Newsletter bool   `json:"newsletter"`
	AppId      string `json:"appId"`
	AppName    string `json:"appName"`
	AppScheme  string `json:"appScheme"`
	AppPort    string `json:"appPort"`
}

type LoginRequestMeta struct {
	Audience  string `json:"aud"`
	AppId     string `json:"appId"`
	AppName   string `json:"appName"`
	AppScheme string `json:"appScheme"`
	AppPort   string `json:"appPort"`
}

func (s *StateInfo) ResponseUrl(hostname string) string {
	return hostname + pagenames.INTERNAL_OAUTH_RESPONSE + "/"
}

func (s *StateInfo) DestinationUrl(rData *pages.RequestData, jwtRecord *datastore.JwtRecord) (string, error) {

	//take the jwt, strip the extra stuff, and resign (gets around long uri limit)
	jwtData := jwtRecord.GetData()

	var newRecord datastore.JwtRecord

	newRecord.SetData(&datastore.JwtData{
		SelfId:       jwtData.SelfId, //this probably isn't necessary since it'll be automatically filled with newRecord's GetData() in SignJwt, but doesn't hurt
		Audience:     jwtData.Audience,
		UserId:       jwtData.UserId,
		UserType:     jwtData.UserType,
		ExpiresAt:    jwtData.ExpiresAt,
		IssuedAt:     jwtData.IssuedAt,
		Scopes:       jwtData.Scopes,
		SessionId:    jwtData.SessionId,
		FinalExpires: jwtData.FinalExpires,
		Subject:      jwtData.Subject,
		//ommitting: Extra        string `json:"extra,omitempty" datastore:",noindex"`
	})

	newJwtString, err := auth.SignJwt(rData.Ctx, &newRecord)
	if err != nil {
		return "", err
	}

	return s.Scheme + s.Destination + "/" + newJwtString, nil
}

func (s *StateInfo) ErrorUrl(statusCode string) string {
	return s.Scheme + "status/" + statusCode
}

type OAuthUserInfo struct {
	Id        string `json:"uid,omitempty" datastore:",noindex"`
	Email     string `json:"email,omitempty" datastore:",noindex"`
	FirstName string `json:"fname,omitempty" datastore:",noindex"`
	LastName  string `json:"lname,omitempty" datastore:",noindex"`
	AvatarURL string `json:"aurl,omitempty" datastore:",noindex"`
}

type FacebookPictureData struct {
	Url string `json:"url"`
}
type FacebookPicture struct {
	Data FacebookPictureData `json:"data"`
}
type FacebookUser struct {
	Id            string          `json:"id"`
	Email         string          `json:"email"`
	FirstName     string          `json:"first_name"`
	LastName      string          `json:"last_name"`
	AvatarPicture FacebookPicture `json:"picture"`
}

//really just service request - after callbacks and all that

func OauthRequest(rData *pages.RequestData) {

	state := StateInfo{
		Destination: rData.HttpRequest.FormValue("dest"),
		Scheme:      rData.HttpRequest.FormValue("scheme"),
		Provider:    rData.HttpRequest.FormValue("provider"),
		Request:     rData.HttpRequest.FormValue("request"),
		RequestMeta: rData.HttpRequest.FormValue("meta"),
	}

	if !slice.StringInSlice(state.Destination, OAUTH_ALLOWED_DESTINATIONS) || !slice.StringInSlice(state.Scheme, rData.SiteConfig.OAUTH_ALLOWED_SCHEMES) || !slice.StringInSlice(state.Provider, OAUTH_ALLOWED_PROVIDERS) || !slice.StringInSlice(state.Request, OAUTH_ALLOWED_REQUESTS) {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	stateBytes, err := json.Marshal(state)
	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	_, stateJwtString, err := auth.GetNewSystemsOobJWT(rData, auth.SYSTEM_ID_OAUTH, jwt_scopes.OAUTH_STATE, string(stateBytes))
	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	endpoint := getEndpointConfig(rData, &state)
	if endpoint == nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	authUrl := endpoint.AuthCodeURL(stateJwtString, oauth2.ApprovalForce, oauth2.AccessTypeOnline)

	authUrlFormatted := fmt.Sprintf("%s", authUrl)
	rData.LogInfo(authUrlFormatted)

	rData.SetJsonSuccessResponse(pages.JsonMapGeneric{
		"url": authUrlFormatted,
	})
}

func OauthResponse(rData *pages.RequestData) {

	stateJwtString := rData.HttpRequest.FormValue("state")
	code := rData.HttpRequest.FormValue("code")

	stateJwtRecord, state, err := getStateAndRecordFromJwtString(rData, stateJwtString)

	if code != "" && err == nil {
		var response string

		if state.Request == "userinfo" {
			userInfo := getUserInfoFromRequest(rData, state, code)
			if userInfo != nil {
				if infoBytes, err := json.Marshal(userInfo); err == nil {
					response = string(infoBytes)
				}
			}
		}

		if response != "" {
			state.Response = response
			if stateBytes, err := json.Marshal(state); err == nil {
				stateJwtRecord.GetData().Extra = string(stateBytes)
				if err := datastore.Save(rData.Ctx, stateJwtRecord); err == nil {

					dst, err := state.DestinationUrl(rData, stateJwtRecord)

					if err == nil {
						//SUCCESS!
						rData.HttpRedirectDestination = dst
						return
					}

				}
			}
		}
	}

	if state == nil || state.Scheme == "" {
		state = &StateInfo{
			Scheme: rData.SiteConfig.OAUTH_ALLOWED_SCHEMES[len(rData.SiteConfig.OAUTH_ALLOWED_SCHEMES)-1],
		}
	}

	rData.HttpRedirectDestination = state.ErrorUrl(statuscodes.AUTH)
}

func OauthAction(rData *pages.RequestData) {
	rData.LogInfo("JWT: %v", rData.JwtString)
	
	response := pages.JsonMapGeneric{}

	state, err := getStateFromJwtRecord(rData.JwtRecord)
	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.AUTH)
		return
	}

	if state.Request == "userinfo" {
		var userInfo OAuthUserInfo
		if err := json.Unmarshal([]byte(state.Response), &userInfo); err != nil {
			rData.SetJsonErrorCodeResponse(statuscodes.AUTH)
			return
		}

		actionType := rData.HttpRequest.FormValue("action")

		if actionType == "login" {
			var requestMeta LoginRequestMeta

			if state.RequestMeta != "" {
				if err := json.Unmarshal([]byte(state.RequestMeta), &requestMeta); err != nil {
					rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
					return
				}
				response["meta"] = requestMeta
			}

                        _, userRecordInfo, _, jwtString, err := DoLogin(rData, userInfo.Id, "", requestMeta.Audience, LOOKUP_TYPE_OAUTH)

			if err != nil {
				response["code"] = err.Error()
				rData.SetJsonErrorResponse(response)
				return
			}

			response["jwt"] = jwtString
                        response["userInfo"] = userRecordInfo
			rData.SetJsonSuccessResponse(response)
			return

		} else if actionType == "register" {
			var requestMeta RegisterRequestMeta
			if err := json.Unmarshal([]byte(state.RequestMeta), &requestMeta); err != nil {
				rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
				return
			}

			response["meta"] = requestMeta

			registerInfo := &RegisterInfo{
				Terms:        requestMeta.Terms,
				Newsletter:   requestMeta.Newsletter,
				Username:     userInfo.Id,
				EmailAddress: userInfo.Email,
				FirstName:    userInfo.FirstName,
				LastName:     userInfo.LastName,
				LookupType:   LOOKUP_TYPE_OAUTH,
				AvatarUrl:    userInfo.AvatarURL,
				AppId:        requestMeta.AppId,
				AppPort:      requestMeta.AppPort,
			}

			//theoretically, we could lookup the user's email address and allow syncing multiple accounts based on that email...
			//but decided not to, a separate service could be created to connect/disconnect accounts, and that would be better
			//so the structure is there to separate lookup from actual record, and multiple lookups can be used to reference a single record
			//and it was given lots of thought... but not actually used atm
			if err := DoRegister(rData, registerInfo); err != nil {
				response["code"] = err.Error()
				rData.SetJsonErrorResponse(response)
				return
			}

			//At this point, registration is done and we now log the user in
			//maybe we should just return and let the client issue a login with the same token?
			//would be a bit cleaner to keep the logic in one place...

			userRecord, err := GetUserRecordViaUsername(rData.Ctx, userInfo.Id)
			if err != nil {
				userRecord = nil
			}
			jwtRecord, jwtString, err := auth.GetNewLoginJWT(rData, userRecord, requestMeta.Audience)

			if err != nil {
				response["code"] = statuscodes.TECHNICAL
				rData.SetJsonErrorResponse(response)
				return
			}

			//not this because it will destroy cookie-based login token too! rData.DeleteJwtWhenFinished = true
			auth.DestroyToken(rData)

			response["jwt"] = jwtString
                        response["userInfo"] = GetUserInfoFromRecord(userRecord)
			rData.SetJsonSuccessResponse(response)

			if requestMeta.Audience == auth.JWT_AUDIENCE_COOKIE {
				auth.SetJWTCookie(rData, jwtString, jwtRecord.GetData().SessionId, int(auth.GetFinalDurationByAudience(jwtRecord.GetData().Audience)))
			}
		}
	}

}

func getStateFromJwtRecord(stateJwtRecord *datastore.JwtRecord) (*StateInfo, error) {
	var state StateInfo

	err := json.Unmarshal([]byte(stateJwtRecord.GetData().Extra), &state)
	return &state, err
}

func getStateAndRecordFromJwtString(rData *pages.RequestData, stateJwtString string) (*datastore.JwtRecord, *StateInfo, error) {

	stateJwtRecord, isExpired := auth.GetJwtFromString(rData, stateJwtString, true)

	if stateJwtRecord != nil && !isExpired {
		if state, err := getStateFromJwtRecord(stateJwtRecord); err == nil {
			return stateJwtRecord, state, nil
		}
	}

	return stateJwtRecord, nil, errors.New(statuscodes.AUTH)
}

func getUserInfoFromRequest(rData *pages.RequestData, state *StateInfo, code string) *OAuthUserInfo {

	endpointConfig := getEndpointConfig(rData, state)
	if endpointConfig == nil {
		return nil
	}

	tok, err := endpointConfig.Exchange(rData.Ctx, code)
	if err != nil {
		rData.LogInfo("ERROR!!! %v", err)

		return nil
	}

	client := endpointConfig.Client(rData.Ctx, tok)

	var userInfo *OAuthUserInfo

	if state.Provider == "google" {
		googleUserInfo, err := getInfo_Google(rData, client)
		if err == nil {
			userInfo = &OAuthUserInfo{
				Id:        googleUserInfo.Id,
				Email:     googleUserInfo.Email,
				FirstName: googleUserInfo.GivenName,
				LastName:  googleUserInfo.FamilyName,
				AvatarURL: googleUserInfo.Picture,
			}
		}
	} else if state.Provider == "facebook" {
		facebookUserInfo, err := getInfo_Facebook(rData, client, tok.AccessToken)
		if err == nil {
			userInfo = &OAuthUserInfo{
				Id:        facebookUserInfo.Id,
				Email:     facebookUserInfo.Email,
				FirstName: facebookUserInfo.FirstName,
				LastName:  facebookUserInfo.LastName,
				AvatarURL: facebookUserInfo.AvatarPicture.Data.Url,
			}
		}
	}

	if userInfo != nil {
		userInfo.Id = rData.SiteConfig.OAUTH_USERID_PREFIX + "-" + state.Provider + "-" + userInfo.Id
	}

	return userInfo
}

func getEndpointConfig(rData *pages.RequestData, state *StateInfo) *oauth2.Config {

	if state.Provider == "google" {
		config := &oauth2.Config{
			ClientID:     rData.SiteConfig.OAUTH_GOOGLE_CLIENTID,
			ClientSecret: rData.SiteConfig.OAUTH_GOOGLE_CLIENTSECRET,
			RedirectURL:  state.ResponseUrl(rData.SiteConfig.API_HOSTNAME),
			Scopes:       []string{goauth2.PlusLoginScope, goauth2.PlusMeScope, goauth2.UserinfoEmailScope, goauth2.UserinfoProfileScope},
			Endpoint:     google.Endpoint,
		}

		return config
	} else if state.Provider == "facebook" {
		return &oauth2.Config{
			ClientID:     rData.SiteConfig.OAUTH_FACEBOOK_CLIENTID,
			ClientSecret: rData.SiteConfig.OAUTH_FACEBOOK_CLIENTSECRET,
			RedirectURL:  state.ResponseUrl(rData.SiteConfig.API_HOSTNAME),
			Scopes:       []string{"email", "user_about_me", "public_profile"},
			Endpoint:     facebook.Endpoint,
		}
	}
	return nil

}

func getInfo_Google(rData *pages.RequestData, client *http.Client) (*goauth2.Userinfoplus, error) {

	service, err := goauth2.New(client)
	if err != nil {
		return nil, err

	}
	uService := goauth2.NewUserinfoService(service)
	return uService.Get().Do()

	//fmt.Fprintf(w, "Username: %s %s<br>ID: %s<br>Email: %s<br>Picture: <img src='%s' /><br>", gouser.GivenName, gouser.FamilyName, gouser.Id, gouser.Email, gouser.Picture)
}

func getInfo_Facebook(rData *pages.RequestData, client *http.Client, accessToken string) (*FacebookUser, error) {

	response, err := client.Get(fmt.Sprintf("https://graph.facebook.com/me?access_token=%s&fields=email,first_name,last_name,picture.type(large)", accessToken))

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	str, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var facebookUser FacebookUser

	err = json.Unmarshal([]byte(str), &facebookUser)
	if err != nil {
		return nil, err
	}

	return &facebookUser, nil

	//fmt.Fprintf(w, "Username: %s<br>ID: %s<br>Birthday: %s<br>Email: %s<br>", fauser.Name, fauser.Id, fauser.Birthday, fauser.Email)
	//img := fmt.Sprintf("https://graph.facebook.com/%s/picture?width=180&height=180", fauser.Id)
}
