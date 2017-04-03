package accounts

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/dakom/basic-site-api/lib/datastore"
	"github.com/dakom/basic-site-api/lib/pages"
	"github.com/dakom/basic-site-api/setup/config/static/pagenames"
	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"

	"google.golang.org/appengine/taskqueue"
)

func GotNameChangeServiceRequest(rData *pages.RequestData) {

	if rData.HttpRequest.FormValue("ntype") == "dname" {
		//if we're only changing the display name - just do it and get out early
		//other cases involve changing third party data (mailing list etc.)

		dName := strings.TrimSpace(rData.HttpRequest.FormValue("name"))
		if rData.SiteConfig.DisplayNameValidator != nil && !rData.SiteConfig.DisplayNameValidator(dName) {
			rData.SetJsonErrorCodeResponse(statuscodes.INVALID_DISPLAYNAME)
			return
		}

		rData.UserRecord.GetData().DisplayName = dName
		if err := datastore.Save(rData.Ctx, rData.UserRecord); err != nil {
			rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
			return
		}

		rData.SetJsonSuccessCodeResponse(statuscodes.NAME_CHANGED)
	} else {
		var fname string
		var lname string

		missingInfo := true

		if rData.HttpRequest.FormValue("ntype") == "fname" {
			fname = strings.TrimSpace(rData.HttpRequest.FormValue("name"))
		} else if rData.HttpRequest.FormValue("ntype") == "lname" {
			lname = strings.TrimSpace(rData.HttpRequest.FormValue("name"))
		} else {
			fname = strings.TrimSpace(rData.HttpRequest.FormValue("fname"))
			lname = strings.TrimSpace(rData.HttpRequest.FormValue("lname"))
		}

		if len(fname) > 1 {
			rData.UserRecord.GetData().FirstName = fname
			missingInfo = false
		}
		if len(lname) > 1 {
			rData.UserRecord.GetData().LastName = lname
			missingInfo = false
		}

		if missingInfo {
			rData.SetJsonErrorCodeResponse(statuscodes.MISSINGINFO)
			return
		}

		err := datastore.Save(rData.Ctx, rData.UserRecord)
		if err != nil {
			rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
			return
		}

		params := url.Values{}
		params.Set("uid", strconv.FormatInt(rData.UserRecord.GetKey().IntID(), 10))
		params.Set("fname", rData.UserRecord.GetData().FirstName)
		params.Set("lname", rData.UserRecord.GetData().LastName)

		mailingListTask := taskqueue.NewPOSTTask("/"+pagenames.MAILINGLIST_UPDATE_NAME_WEBHOOK, params)
		_, err = taskqueue.Add(rData.Ctx, mailingListTask, rData.SiteConfig.TASKQUEUE_MAILINGLIST)

		if err != nil {
			rData.LogError("%v", err)
		}

		rData.SetJsonSuccessCodeResponse(statuscodes.NAME_CHANGED)
	}
}
