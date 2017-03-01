// email
package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/dakom/basic-site-api/lib/pages"

	"github.com/dakom/basic-site-api/lib/datastore"

	"google.golang.org/appengine/urlfetch"
)

type MailchimpAPIError struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
	Name   string `json:"name"`
	Err    string `json:"error"`
}

type MailchimpSuccessResponse struct {
	Email       string `json:"email"`
	EmailId     string `json:"euid"`
	ListEmailId string `json:"leid"`
}

type MergeVarsSubscribe struct {
	FirstName string      `json:"FNAME"`
	LastName  string      `json:"LNAME"`
	NewEmail  string      `json:"new-email"`
	Groupings []Groupings `json:"groupings"`
}

type MergeVarsUpdateEmail struct {
	NewEmail string `json:"new-email"`
}

type MergeVarsUpdateName struct {
	FirstName string `json:"FNAME"`
	LastName  string `json:"LNAME"`
}

type Groupings struct {
	ID     int      `json:"id"`
	Groups []string `json:"groups"`
}

func mailchimpSubscribe(rData *pages.RequestData, userRecord *datastore.UserRecord) (*MailchimpSuccessResponse, error) {

	interestGroups := []string{"ViaRegistration"}

	if userRecord.GetData().UserMailinglistData.HasMarketingNewsletter {
		interestGroups = append(interestGroups, "Marketing")
	}

	mergeVars := MergeVarsSubscribe{
		NewEmail:  userRecord.GetData().Email,
		FirstName: userRecord.GetData().FirstName,
		LastName:  userRecord.GetData().LastName,
		Groupings: []Groupings{Groupings{ID: rData.SiteConfig.MAILCHIMP_GROUP_ID, Groups: interestGroups}},
	}

	jsonObject := map[string]interface{}{"apikey": rData.SiteConfig.MAILCHIMP_APIKEY, "id": rData.SiteConfig.MAILCHIMP_LIST_ID, "email": map[string]string{"email": userRecord.GetData().Email}, "merge_vars": mergeVars, "update_existing": true, "replace_interests": false, "double_optin": false}

	return mailchimpApiCall(rData, "lists/subscribe.json", jsonObject)
}

func mailchimpUpdateEmail(rData *pages.RequestData, userRecord *datastore.UserRecord) (*MailchimpSuccessResponse, error) {
	var resp *MailchimpSuccessResponse
	var err error

	mergeVars := MergeVarsUpdateEmail{
		NewEmail: userRecord.GetData().Email,
	}

	jsonObject := map[string]interface{}{"apikey": rData.SiteConfig.MAILCHIMP_APIKEY, "id": rData.SiteConfig.MAILCHIMP_LIST_ID, "email": map[string]string{"leid": userRecord.GetData().UserMailinglistData.ListEmailId}, "merge_vars": mergeVars, "email_type": "html", "replace_interests": false}

	resp, err = mailchimpApiCall(rData, "lists/update-member.json", jsonObject)

	if err != nil && err.Error() == "List_AlreadySubscribed" {
		resp, err = mailchimpUnsubscribe(rData, userRecord.GetData().Email)
		if err == nil {
			resp, err = mailchimpApiCall(rData, "lists/update-member.json", jsonObject)
		} else {

			resp, err = mailchimpSubscribe(rData, userRecord)

			if err != nil {
				rData.LogError("unable to update email, tried everything but ultimately got error: %v", err)
				return nil, nil
			}
		}
	}

	return resp, err

}

func mailchimpUpdateName(rData *pages.RequestData, userRecord *datastore.UserRecord) (*MailchimpSuccessResponse, error) {

	mergeVars := MergeVarsUpdateName{
		FirstName: userRecord.GetData().FirstName,
		LastName:  userRecord.GetData().LastName,
	}

	jsonObject := map[string]interface{}{"apikey": rData.SiteConfig.MAILCHIMP_APIKEY, "id": rData.SiteConfig.MAILCHIMP_LIST_ID, "email": map[string]string{"leid": userRecord.GetData().UserMailinglistData.ListEmailId}, "merge_vars": mergeVars, "email_type": "html", "replace_interests": false}

	return mailchimpApiCall(rData, "lists/update-member.json", jsonObject)
}

func mailchimpUnsubscribe(rData *pages.RequestData, emailAddress string) (*MailchimpSuccessResponse, error) {

	jsonObject := map[string]interface{}{"apikey": rData.SiteConfig.MAILCHIMP_APIKEY, "id": rData.SiteConfig.MAILCHIMP_LIST_ID, "email": map[string]string{"email": emailAddress}, "delete_member": false, "send_goodbye": false, "send_notify": false}

	return mailchimpApiCall(rData, "lists/unsubscribe.json", jsonObject)
}

func mailchimpApiCall(rData *pages.RequestData, apiName string, jsonObject map[string]interface{}) (*MailchimpSuccessResponse, error) {
	var mailchimpSuccessResponse MailchimpSuccessResponse

	client := urlfetch.Client(rData.Ctx)

	jsonData, err := json.Marshal(jsonObject)
	if err != nil {
		return &mailchimpSuccessResponse, err
	}

	jsonBuffer := bytes.NewBuffer(jsonData)

	if err != nil {
		return &mailchimpSuccessResponse, err
	}

	httpResponse, err := client.Post(rData.SiteConfig.MAILCHIMP_APIENDPOINT+apiName, "application/json", jsonBuffer)
	if err != nil {
		return &mailchimpSuccessResponse, err
	}

	defer httpResponse.Body.Close()
	body, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return &mailchimpSuccessResponse, err
	}

	var mailchimpAPIError MailchimpAPIError

	json.Unmarshal(body, &mailchimpAPIError)
	if mailchimpAPIError.Err != "" || mailchimpAPIError.Code != 0 {
		return &mailchimpSuccessResponse, fmt.Errorf(mailchimpAPIError.Name) //fmt.Errorf("Error: %v %v %v %v", mailchimpAPIError.Status, mailchimpAPIError.Code, mailchimpAPIError.Name, mailchimpAPIError.Err)
	}

	json.Unmarshal(body, &mailchimpSuccessResponse)

	if mailchimpSuccessResponse.EmailId == "" || mailchimpSuccessResponse.ListEmailId == "" {
		return &mailchimpSuccessResponse, fmt.Errorf("NO LIST ID!")
	}

	return &mailchimpSuccessResponse, nil
}
