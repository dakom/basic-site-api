package init

import (
	"net/http"
	"strings"

	"github.com/dakom/basic-site-api/lib/auth"
	"github.com/dakom/basic-site-api/lib/auth/jwt_scopes"
	"github.com/dakom/basic-site-api/lib/pages"
	"github.com/dakom/basic-site-api/setup/config/custom"
	"github.com/dakom/basic-site-api/setup/config/extendable/pageconfig"
	"github.com/dakom/basic-site-api/setup/config/static/pagenames"
	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"

	"google.golang.org/appengine"
)

func Start(extraPageConfigs map[string]*pages.PageConfig, siteConfig *custom.Config) {
	pageConfigs := pageconfig.GetPageConfigs(extraPageConfigs)

	http.HandleFunc("/", wrapRequest(pageConfigs, siteConfig))
}

func wrapRequest(pageConfigs map[string]*pages.PageConfig, siteConfig *custom.Config) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		gotPageRequest(w, r, pageConfigs, siteConfig)
	}
}

func cullAllowedHeaders(requestedHeaderString string, allowedHeaders []string) []string {
	var result []string
	requestedHeaders := strings.Split(requestedHeaderString, ", ")
	for _, requestedHeader := range requestedHeaders {
		requestedHeader = strings.TrimSpace(requestedHeader)
		for _, allowedHeader := range allowedHeaders {
			if strings.ToLower(requestedHeader) == strings.ToLower(allowedHeader) {
				result = append(result, requestedHeader)
				break
			}
		}
	}

	return result
}

func gotPageRequest(w http.ResponseWriter, r *http.Request, pageConfigs map[string]*pages.PageConfig, siteConfig *custom.Config) {
	var ok, isAuthorized, jwtWasRefreshed bool

	//add CORS

	pageName := strings.Trim(r.URL.Path, "/")

	rData := &pages.RequestData{
		Ctx:                    appengine.NewContext(r),
		SiteConfig:             siteConfig,
		HttpWriter:             w,
		HttpRequest:            r,
		HttpStatusResponseCode: 200,
	}

	/*
		for k, v := range rData.HttpRequest.Header {
			rData.LogInfo("%v = %v", k, v)
		}

	*/

	//CORS - headers (cull the list for allowed headers)
	accessControlAllowedHeaders := cullAllowedHeaders(r.Header.Get("Access-Control-Request-Headers"), siteConfig.CORS_ALLOWED_HEADERS)
	if len(accessControlAllowedHeaders) > 0 {
		rData.HttpWriter.Header().Add("Access-Control-Allow-Headers", strings.Join(accessControlAllowedHeaders, ", "))
	}

	//CORS - methods (cull the list for allowed headers... we don't distinguish per header type for now)
	accessControlAllowedMethods := cullAllowedHeaders(r.Header.Get("Access-Control-Request-Method"), siteConfig.CORS_ALLOWED_METHODS)
	if len(accessControlAllowedMethods) > 0 {
		rData.HttpWriter.Header().Add("Access-Control-Allow-Methods", strings.Join(accessControlAllowedMethods, ", "))
	}

	//CORS - origin (only one at a time- first come first serve)
	accessControlRequestOrigin := r.Header.Get("Origin")
	for _, allowedOrigin := range siteConfig.CORS_ALLOWED_ORIGINS {
		if strings.ToLower(accessControlRequestOrigin) == strings.ToLower(allowedOrigin) {
			rData.HttpWriter.Header().Add("Access-Control-Allow-Origin", allowedOrigin)
			rData.HttpWriter.Header().Add("Access-Control-Allow-Credentials", "true")
			break
		}
	}

	//CORS - preflight options, exit early
	if strings.ToLower(r.Method) == "options" {
		rData.HttpWriter.WriteHeader(200)
		return
	}

	if rData.PageConfig, ok = pageConfigs[pageName]; !ok {
		rData.PageConfig = nil

		for configName, configData := range pageConfigs {
			if configName != "" {
				if strings.HasPrefix(pageName, configName) {

					rData.PageConfig = configData
					rData.ExtraUrlParams = strings.Split(pageName[len(configName):], "/")
					rData.PageConfig.PageName = configName
					break
				}
			}
		}

	} else {
		rData.PageConfig.PageName = pageName
	}

	if rData.PageConfig == nil {
		rData.SetJsonErrorCodeResponse("o_O")
		rData.OutputJsonString()
		return
	}

	//deal with special case scenarios of login/logout
	//wipe the existing login, and skip jwt check for target page
	if rData.PageConfig.PageName == pagenames.ACCOUNT_LOGOUT_SERVICE || rData.PageConfig.PageName == pagenames.ACCOUNT_LOGIN_SERVICE {
		if err := auth.DestroyToken(rData); err != nil {
			rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
			rData.OutputJsonString()
			return
		}
		if rData.PageConfig.PageName == pagenames.ACCOUNT_LOGOUT_SERVICE {
			rData.SetJsonSuccessCodeResponse(statuscodes.LOGOUT_SUCCESS)
			rData.OutputJsonString()
			return
		}

		isAuthorized = true
	} else {
		//authorization checks
		isAuthorized, jwtWasRefreshed = auth.ValidatePageRequest(rData)
	}

	//alright, let's check the authorization!
	if !isAuthorized {
		statusCode := statuscodes.AUTH

		switch rData.PageConfig.Scopes {
		case jwt_scopes.TRANSACTION_PAYMENT, jwt_scopes.OOB_USER_PASSWORD_CHANGE, jwt_scopes.OOB_USER_EMAIL_CHANGE, jwt_scopes.OOB_USER_ACTIVATE:
			statusCode = statuscodes.AUTH_OOB
		}

		if rData.PageConfig.HandlerType == pages.HANDLER_TYPE_JSON {

			rData.SetJsonErrorCodeResponse(statusCode)
			rData.OutputJsonString()
		} else {
			rData.SetHttpStatusResponse(401, statusCode)
			rData.OutputHttpResponse()
		}
		return
	}

	//from here on in we are definately authorized!

	//if audience is web, set it right away (cookies must come before all!)
	if jwtWasRefreshed && rData.JwtRecord.GetData().Audience == auth.JWT_AUDIENCE_COOKIE {
		auth.SetJWTCookie(rData, rData.JwtString, rData.JwtRecord.GetData().SessionId, int(auth.GetFinalDurationByAudience(rData.JwtRecord.GetData().Audience)))
	}

	//Everything is authorized! Let's go for it...

	//for these, we need to set the header first since output is just logged on the fly
	//note that those pages are usually just admin/debugging type pages- generally everything else is templates or set json
	if rData.PageConfig.HandlerType == pages.HANDLER_TYPE_HTML_STRINGS {
		rData.SetContentType("text/html; charset=utf-8")
	}
	if rData.PageConfig.Handler != nil {
		rData.PageConfig.Handler(rData)
	}

	if rData.DeleteJwtWhenFinished {
		if err := auth.DestroyToken(rData); err != nil {
			rData.LogError(err.Error()) //non-critical, but log for investigation
		}
	}

	if rData.PageConfig.HandlerType == pages.HANDLER_TYPE_HTML_STRINGS {
		//do nothing, for html it's templates and things
	} else if rData.PageConfig.HandlerType == pages.HANDLER_TYPE_HTTP_STATUS {

		rData.OutputHttpResponse()
	} else if rData.PageConfig.HandlerType == pages.HANDLER_TYPE_JSON {
		//json must mix in after processing
		if jwtWasRefreshed {
			if rData.JsonResponse == nil {
				rData.JsonResponse = make(pages.JsonMapGeneric)
			}
			rData.JsonResponse.SetJwt(rData.JwtString)
		}
		rData.OutputJsonString()
	} else if rData.PageConfig.HandlerType == pages.HANDLER_TYPE_HTTP_REDIRECT {
		if rData.HttpRedirectIsPermanent {
			http.Redirect(rData.HttpWriter, rData.HttpRequest, rData.HttpRedirectDestination, http.StatusMovedPermanently)
		} else {
			http.Redirect(rData.HttpWriter, rData.HttpRequest, rData.HttpRedirectDestination, http.StatusTemporaryRedirect)
		}

		return
	}

}
