package accounts

import (
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/dakom/basic-site-api/lib/auth/auth_roles"
	"github.com/dakom/basic-site-api/lib/datastore"
	"github.com/dakom/basic-site-api/lib/pages"
	"github.com/dakom/basic-site-api/lib/utils/cipher"
	"github.com/dakom/basic-site-api/lib/utils/text"
	"github.com/dakom/basic-site-api/setup/config/static/pagenames"
	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"
	"golang.org/x/net/context"

	gaeds "google.golang.org/appengine/datastore"
	"google.golang.org/appengine/taskqueue"
)

type RegisterInfo struct {
	Terms      bool
	Newsletter bool

	Username     string
	EmailAddress string
	FirstName    string
	LastName     string
	Password     string
	ParentId     int64

	AppId   string
	AppPort string

	AvatarUrl  string
	LookupType int64 //only used if subaccounts are allowed
}

func GotRegisterServiceRequest(rData *pages.RequestData) {

	var terms bool
	var newsletter bool

	if rData.HttpRequest.FormValue("terms") == "true" {
		terms = true
	}
	if rData.HttpRequest.FormValue("newsletter") == "true" {
		newsletter = true
	}

	info := &RegisterInfo{
		Terms:        terms,
		Newsletter:   newsletter,
		Username:     strings.ToLower(strings.TrimSpace(rData.HttpRequest.FormValue("uname"))),
		EmailAddress: strings.ToLower(strings.TrimSpace(rData.HttpRequest.FormValue("uname"))),
		FirstName:    strings.TrimSpace(rData.HttpRequest.FormValue("fname")),
		LastName:     strings.TrimSpace(rData.HttpRequest.FormValue("lname")),
		Password:     strings.TrimSpace(rData.HttpRequest.FormValue("pw")),
		LookupType:   LOOKUP_TYPE_USERNAME,
		AppId:        strings.TrimSpace(rData.HttpRequest.FormValue("appId")),
		AppPort:      strings.TrimSpace(rData.HttpRequest.FormValue("appPort")),
	}
	if err := DoRegister(rData, info); err != nil {
		rData.SetJsonErrorCodeResponse(err.Error())
		return
	}
	rData.SetJsonSuccessCodeResponse(statuscodes.CHECK_EMAIL)
}

func DoRegister(rData *pages.RequestData, info *RegisterInfo) error {
	var parentRecord *datastore.UserRecord

	if !info.Terms {
		return errors.New(statuscodes.TERMS)
	}

	if info.LookupType == LOOKUP_TYPE_OAUTH { //oauth requires having correct prefix
		if !strings.HasPrefix(info.Username, rData.SiteConfig.OAUTH_USERID_PREFIX) {
			return errors.New(statuscodes.INVALID_USERNAME)
		}
	} else { //non-oauth may not have that prefix
		if strings.HasPrefix(info.Username, rData.SiteConfig.OAUTH_USERID_PREFIX) {
			return errors.New(statuscodes.INVALID_USERNAME)
		}
		if info.ParentId != 0 { //subaccount, username must validate with normal alphanumeric
			re := regexp.MustCompile("^[a-zA-Z0-9_]*$")
			if !re.MatchString(info.Username) {
				return errors.New(statuscodes.INVALID_USERNAME)
			}

			//must also have passed valid parent
			var err error
			parentRecord, err = GetUserRecordViaKey(rData.Ctx, info.ParentId)
			if err != nil {
				return errors.New(statuscodes.TECHNICAL)
			} else if parentRecord == nil {
				return errors.New(statuscodes.MISSINGINFO)
			}
		} else { //regular account - mustvalidate with email
			if info.EmailAddress == "" {
				return errors.New(statuscodes.MISSINGINFO)
			}
			if !govalidator.IsEmail(info.EmailAddress) {
				return errors.New(statuscodes.INVALID_EMAIL)
			}
		}
	}

	if info.Password == "" {
		if newPassword, err := text.RandomHexString(12); err == nil {
			info.Password = newPassword
		}
	}

	if info.Username == "" || info.FirstName == "" || info.LastName == "" || info.Password == "" {
		return errors.New(statuscodes.MISSINGINFO)
	}

	if len(info.Password) < 6 || len(info.Password) > 32 {
		return errors.New(statuscodes.INVALID_PASSWORD)
	}

	existingUserRecord, err := GetUserRecordViaUsername(rData.Ctx, info.Username)
	if err != nil {

		return errors.New(statuscodes.TECHNICAL)
	}
	if existingUserRecord != nil {
		return errors.New(statuscodes.USERNAME_EXISTS)
	}

	passwordHash, err := cipher.NewPWHash(info.Password, nil)

	if err != nil {

		return errors.New(statuscodes.TECHNICAL)
	}

	opts := gaeds.TransactionOptions{
		XG: true,
	}

	err = gaeds.RunInTransaction(rData.Ctx, func(c context.Context) error {
		var userRecord datastore.UserRecord
		userRecord.GetData().UsernameHistory = []string{info.Username}
		userRecord.GetData().Email = info.EmailAddress
		userRecord.GetData().FirstName = info.FirstName
		userRecord.GetData().LastName = info.LastName
		userRecord.GetData().Password = passwordHash
		userRecord.GetData().AddedDate = time.Now()
		userRecord.GetData().Roles = auth_roles.USER

		/*Conceptually subaccounts could have been created as actual children of master accounts
		  However that would require knowing the parentid in conjunction with the username
		  And/or storing it with the lookup record ... and across the pipeline.
		  It didn't seem to add much value that way and this is easier
		*/
		if parentRecord != nil {
			userRecord.GetData().ParentId = info.ParentId
		}
		if info.LookupType == LOOKUP_TYPE_OAUTH || parentRecord != nil {
			userRecord.GetData().IsActive = true
		}
		if info.Newsletter {
			userRecord.GetData().UserMailinglistData.HasMarketingNewsletter = true
		}

		if err := datastore.SaveToAutoKey(rData.Ctx, &userRecord); err != nil {
			return err
		}

		var lookupRecord datastore.UsernameLookupRecord

		lookupRecord.GetData().UserId = userRecord.GetKey().IntID()

		err = datastore.SaveToKey(rData.Ctx, &lookupRecord, info.Username)
		if err != nil {
			return err
		}

		if parentRecord != nil {
			parentRecord.GetData().SubAccountIds = append(parentRecord.GetData().SubAccountIds, userRecord.GetKey().IntID())
			if err := datastore.Save(rData.Ctx, parentRecord); err != nil {
				return err
			}
		}

		if !userRecord.GetData().IsActive {
			params := url.Values{}
			params.Set("uname", info.Username)
			if info.AppId != "" {
				params.Set("appId", info.AppId)
			}
			if info.AppPort != "" {
				params.Set("appPort", info.AppPort)
			}
			activationTask := taskqueue.NewPOSTTask("/"+pagenames.ACCOUNT_ACTIVATE_SEND_TOKEN, params)
			_, err = taskqueue.Add(rData.Ctx, activationTask, rData.SiteConfig.TASKQUEUE_REGISTER)
			if err != nil {

				return err
			}
		}

		if info.EmailAddress != "" {
			//note... theoretically this could easily be a bad/spam address... but on the other hand they might just not complete the activation process
			//makes sense to grab it and then change it if we get in trouble with spam
			//similarly since we have the flag in datastore, we could write a utility that culls the list before sending, even with tuning (i.e. cull 70% of non-activated addresses before sending a blast)
			params := url.Values{}
			params.Set("uid", strconv.FormatInt(userRecord.GetKey().IntID(), 10))
			if info.AppId != "" {
				params.Set("appId", info.AppId)
			}
			if info.AppPort != "" {
				params.Set("appPort", info.AppPort)
			}

			if info.AvatarUrl != "" {
				//in this case, set the url in the subscribe webhook to avoid a race condition
				params.Set("aurl", info.AvatarUrl)
			}
			mailingListTask := taskqueue.NewPOSTTask("/"+pagenames.MAILINGLIST_SUBSCRIBE_WEBHOOK, params)
			_, err = taskqueue.Add(rData.Ctx, mailingListTask, rData.SiteConfig.TASKQUEUE_MAILINGLIST)
			if err != nil {
				return err
			}
		} else if info.AvatarUrl != "" {
			params := url.Values{}
			params.Set("uid", strconv.FormatInt(userRecord.GetKey().IntID(), 10))
			params.Set("aurl", info.AvatarUrl)
			if info.AppId != "" {
				params.Set("appId", info.AppId)
			}
			if info.AppPort != "" {
				params.Set("appPort", info.AppPort)
			}

			avatarTask := taskqueue.NewPOSTTask("/"+pagenames.ACCOUNT_AVATAR_PULL_WEBHOOK, params)
			_, err = taskqueue.Add(rData.Ctx, avatarTask, rData.SiteConfig.TASKQUEUE_REGISTER)
			if err != nil {
				rData.LogInfo("AVATAR ERROR %v", err.Error())
				return err
			}
		}

		userRecord.AddToSearch(rData.Ctx)

		return nil
	}, &opts)

	if err != nil {

		return errors.New(statuscodes.TECHNICAL)
	}

	return nil
}
