package pagenames

//these are only used when needed for tasks and emails,
//or hardcoded in pageconfigs to avoid needing to edit in two places for every page
const ACCOUNT_LOGOUT_SERVICE string = "account/logout"
const ACCOUNT_LOGIN_SERVICE string = "account/login"
const ACCOUNT_ACTIVATE_SERVICE string = "account/activate"
const ACCOUNT_ACTIVATE_SEND_TOKEN string = "account/activate-send-token"
const ACCOUNT_AVATAR_PULL_WEBHOOK string = "webhooks/account/avatar-pull"
const MAILINGLIST_SUBSCRIBE_WEBHOOK string = "webhooks/account/mailinglist-subscribe"
const MAILINGLIST_UPDATE_EMAIL_WEBHOOK string = "webhooks/account/mailinglist-update-email"
const MAILINGLIST_UPDATE_NAME_WEBHOOK string = "webhooks/account/mailinglist-update-name"

const INTERNAL_OAUTH_RESPONSE string = "account/oauth-response"

const INTERNAL_INDEX string = "index"
const INTERNAL_STATUS_TEMPLATE string = "status"
const INTERNAL_ACCOUNT_PASSWORD_RESET_FORM string = "account-password-reset-form"
const INTERNAL_ADMIN_VIEWMEDIA string = "admin-viewmedia"
const INTERNAL_ADMIN_EDITMEDIA string = "admin-editmedia"

const APP_PAGE_ACCOUNT_ACTION_ACTIVATE string = "account-action/activate"
const APP_PAGE_ACCOUNT_ACTION_EMAIL_CHANGE string = "account-action/email-change"
const APP_PAGE_ACCOUNT_ACTION_PASSWORD_RESET string = "account-action/password-reset"
