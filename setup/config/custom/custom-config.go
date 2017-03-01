package custom

type Config struct {
	VERSION            string
	MAILINGLIST_TYPE   string
	SENDGRID_APIKEY    string
	SENDGRID_FROM_NAME string
	SENDGRID_FROM_ADDR string

	MAILCHIMP_APIKEY      string
	MAILCHIMP_APIENDPOINT string
	MAILCHIMP_LIST_ID     string
	MAILCHIMP_GROUP_ID    int

	CONSTANT_CONTACT_API_ENDPOINT string
	CONSTANT_CONTACT_KEY          string
	CONSTANT_CONTACT_TOKEN        string
	CONSTANT_CONTACT_LIST_ID      string

	GCS_BUCKET_AVATAR string

	MAX_READ_SIZE int64

	OAUTH_ALLOWED_SCHEMAS []string

	//https://developers.google.com/identity/protocols/OAuth2InstalledApp#choosingredirecturi
	OAUTH_GOOGLE_CLIENTID       string
	OAUTH_GOOGLE_CLIENTSECRET   string
	OAUTH_FACEBOOK_CLIENTID     string
	OAUTH_FACEBOOK_CLIENTSECRET string
	OAUTH_USERID_PREFIX         string

	SUSPEND_AUTH          bool
	SUSPEND_EMAIL         bool
	TASKQUEUE_MAILINGLIST string
	TASKQUEUE_REGISTER    string
	EMAIL_TARGET_HOSTNAME string
	API_HOSTNAME          string
	COOKIE_SECURE         bool
	COOKIE_DOMAIN         string

	JWT_COOKIE_NAME                string
	JWT_COOKIE_SID_NAME            string
	JWT_HEADER_SID_NAME            string
	REQUEST_SOURCE_APPENGINE_APPID string
}
