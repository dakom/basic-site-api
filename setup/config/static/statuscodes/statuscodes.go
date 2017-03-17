package statuscodes

import "errors"

const NONE string = "NONE"

//error
const TECHNICAL string = "TECHNICAL"
const API_AUTH string = "API_AUTH"
const AUTH string = "AUTH"
const AUTH_OOB string = "AUTH_OOB"
const NOT_ACTIVATED string = "NOT_ACTIVATED"
const MISSINGINFO string = "MISSINGINFO"
const MISSING_USERNAME string = "MISSING_USERNAME"
const INVALID_USERNAME string = "INVALID_USERNAME"
const WRONG_PASSWORD string = "WRONG_PASSWORD"
const PASSWORDS_NOMATCH string = "PASSWORDS_NOMATCH"
const MISSING_PASSWORD string = "MISSING_PASSWORD"
const USER_EXISTS string = "USER_EXISTS"
const ALREADY_ACTIVE string = "ALREADY_ACTIVE"
const NOEMAIL string = "NOEMAIL"
const NOUSERNAME string = "NOUSERNAME"
const NOUSERNAME_OAUTH string = "NOUSERNAME_OAUTH"
const NODATA string = "NODATA"
const TERMS string = "TERMS"
const INVALID_EMAIL string = "INVALID_EMAIL"
const INVALID_PASSWORD string = "INVALID_PASSWORD"
const INVALID_FIRSTNAME string = "INVALID_FIRSTNAME"
const INVALID_LASTNAME string = "INVALID_LASTNAME"
const USERNAME_EXISTS string = "USERNAME_EXISTS"
const EMAIL_EXISTS string = "EMAIL_EXISTS"
const CONTACT_EXISTS string = "CONTACT_EXISTS"
const EXPIRED string = "EXPIRED"
const CHECK_EMAIL string = "CHECK_EMAIL"
const RECORD_LENGTH_MISMATCH string = "RECORD_LENGTH_MISMATCH"

//success
const ACTIVATION_COMPLETED string = "ACTIVATION_COMPLETED"
const LOGIN_COMPLETED string = "LOGIN_COMPLETED"
const EMAIL_CHANGED string = "EMAIL_CHANGED"
const NAME_CHANGED string = "NAME_CHANGED"
const PASSWORD_CHANGED string = "PASSWORD_CHANGED"
const ACTIVATION_EXISTS string = "ACTIVATION_EXISTS"
const AVATAR_CHANGED string = "AVATAR_CHANGED"
const LOGOUT_SUCCESS string = "LOGOUT_SUCCESS"

func Error(code string) error {
	return errors.New(code)
}
