package pageconfig

import (
	"github.com/dakom/basic-site-api/endpoints/accounts"
	account_webhooks "github.com/dakom/basic-site-api/endpoints/accounts/webhooks"
	"github.com/dakom/basic-site-api/endpoints/ping"
	"github.com/dakom/basic-site-api/endpoints/version"
	"github.com/dakom/basic-site-api/setup/config/static/pagenames"

	"github.com/dakom/basic-site-api/lib/auth"
	"github.com/dakom/basic-site-api/lib/auth/jwt_scopes"
	"github.com/dakom/basic-site-api/lib/pages"
)

func GetPageConfigs(extraPageConfigs map[string]*pages.PageConfig) map[string]*pages.PageConfig {
	baseConfigs := map[string]*pages.PageConfig{

		//Generic account stuff - basically the foundation

		"account/logout":                      &pages.PageConfig{HandlerType: pages.HANDLER_TYPE_JSON}, //this request is handled directly in main
		"account/login":                       &pages.PageConfig{Handler: accounts.GotLoginServiceRequest, HandlerType: pages.HANDLER_TYPE_JSON},
		"account/login-token-refresh":         &pages.PageConfig{Handler: accounts.GotRefreshTokenRequest, HandlerType: pages.HANDLER_TYPE_JSON, Scopes: jwt_scopes.ACCOUNT_FULL_ANY, RequiresDBScopeCheck: true},
		pagenames.ACCOUNT_ACTIVATE_SEND_TOKEN: &pages.PageConfig{Handler: accounts.SendActivateTokenRequest, HandlerType: pages.HANDLER_TYPE_JSON},
		pagenames.ACCOUNT_ACTIVATE_SERVICE:    &pages.PageConfig{Handler: accounts.GotActivateRequest, HandlerType: pages.HANDLER_TYPE_JSON, Scopes: jwt_scopes.OOB_USER_ACTIVATE},

		"account/register":               &pages.PageConfig{Handler: accounts.GotRegisterServiceRequest, HandlerType: pages.HANDLER_TYPE_JSON},
		"account/password-forgot-authed": &pages.PageConfig{Handler: accounts.GotChangePasswordTokenRequestBySession, HandlerType: pages.HANDLER_TYPE_JSON, Scopes: jwt_scopes.ACCOUNT_FULL_ANY},
		"account/password-forgot":        &pages.PageConfig{Handler: accounts.ForgotPasswordByUsername, HandlerType: pages.HANDLER_TYPE_JSON},
		"account/password-change-action": &pages.PageConfig{Handler: accounts.GotChangePasswordActionRequest, HandlerType: pages.HANDLER_TYPE_JSON, Scopes: jwt_scopes.OOB_USER_PASSWORD_CHANGE},

		"account/email-send-token": &pages.PageConfig{Handler: accounts.GotEmailChangeTokenRequest, HandlerType: pages.HANDLER_TYPE_JSON, Scopes: jwt_scopes.ACCOUNT_FULL_MASTER},
		"account/email-change":     &pages.PageConfig{Handler: accounts.GotEmailChangeActionRequest, HandlerType: pages.HANDLER_TYPE_JSON, Scopes: jwt_scopes.OOB_USER_EMAIL_CHANGE},

		"account/get-info":           &pages.PageConfig{Handler: accounts.GotSettingsInfoServiceRequest, HandlerType: pages.HANDLER_TYPE_JSON, Scopes: jwt_scopes.ACCOUNT_READ},
		"account/name-change":        &pages.PageConfig{Handler: accounts.GotNameChangeServiceRequest, HandlerType: pages.HANDLER_TYPE_JSON, Scopes: jwt_scopes.ACCOUNT_FULL_MASTER},
		"account/avatar-change-file": &pages.PageConfig{Handler: accounts.GotAvatarFileChangeServiceRequest, HandlerType: pages.HANDLER_TYPE_JSON, Scopes: jwt_scopes.ACCOUNT_FULL_MASTER},
		"account/avatar-change-b64":  &pages.PageConfig{Handler: accounts.GotAvatarBase64ChangeServiceRequest, HandlerType: pages.HANDLER_TYPE_JSON, Scopes: jwt_scopes.ACCOUNT_FULL_MASTER},

		"webhooks/account/avatar-pull":              &pages.PageConfig{Handler: account_webhooks.AvatarPull, HandlerType: pages.HANDLER_TYPE_HTTP_STATUS, RequestSource: auth.REQUEST_SOURCE_APPENGINE_TASK},
		"webhooks/account/mailinglist-subscribe":    &pages.PageConfig{Handler: account_webhooks.MailingListSubscribe, HandlerType: pages.HANDLER_TYPE_HTTP_STATUS, RequestSource: auth.REQUEST_SOURCE_APPENGINE_TASK},
		"webhooks/account/mailinglist-update-email": &pages.PageConfig{Handler: account_webhooks.MailingListUpdateEmail, HandlerType: pages.HANDLER_TYPE_HTTP_STATUS, RequestSource: auth.REQUEST_SOURCE_APPENGINE_TASK},
		"webhooks/account/mailinglist-update-name":  &pages.PageConfig{Handler: account_webhooks.MailingListUpdateName, HandlerType: pages.HANDLER_TYPE_HTTP_STATUS, RequestSource: auth.REQUEST_SOURCE_APPENGINE_TASK},

		//oauth
		"account/oauth-request":           &pages.PageConfig{Handler: accounts.OauthRequest, HandlerType: pages.HANDLER_TYPE_JSON},
		pagenames.INTERNAL_OAUTH_RESPONSE: &pages.PageConfig{Handler: accounts.OauthResponse, HandlerType: pages.HANDLER_TYPE_HTTP_REDIRECT},
		"account/oauth-action":            &pages.PageConfig{Handler: accounts.OauthAction, HandlerType: pages.HANDLER_TYPE_JSON, Scopes: jwt_scopes.OAUTH_STATE},

		//subaccounts
		"account/subaccounts-list":   &pages.PageConfig{Handler: accounts.SubaccountsList, HandlerType: pages.HANDLER_TYPE_JSON, Scopes: jwt_scopes.ACCOUNT_FULL_MASTER},
		"account/subaccounts-create": &pages.PageConfig{Handler: accounts.CreateSubaccountRequest, HandlerType: pages.HANDLER_TYPE_JSON, Scopes: jwt_scopes.ACCOUNT_FULL_MASTER},

		//ping/pong - simple util to test roundtripping
		"ping":    &pages.PageConfig{Handler: ping.GotPongRequest, HandlerType: pages.HANDLER_TYPE_JSON, Scopes: jwt_scopes.ACCOUNT_FULL_ANY},
		"version": &pages.PageConfig{Handler: version.GotVersionRequest, HandlerType: pages.HANDLER_TYPE_JSON},

		/*
		 *
		 *  THE FUN STUFF!!!!!!!!!!!!!!!!!!!!!
		 *
		 */

	}

	//mix in the user page config with base page rData.SiteConfig... this is a map since one url must be handled by one handler/config
	for key, val := range extraPageConfigs {
		baseConfigs[key] = val
	}

	return baseConfigs
}
