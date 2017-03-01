// email
package email

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/dakom/basic-site-api/lib/pages"

	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"
	"github.com/dakom/basic-site-api/lib/datastore"

	"google.golang.org/appengine/urlfetch"
)

func constantContactSubscribe(rData *pages.RequestData, userRecord *datastore.UserRecord) (string, error) {
	var contactID string
	var ok bool

	contactInfo := constantContactContactInfoGet(rData, userRecord.GetData().Email, true)

	if contactInfo == nil {
		emailAddresses := []map[string]interface{}{map[string]interface{}{
			"email_address":  userRecord.GetData().Email,
			"confirm_status": "NO_CONFIRMATION_REQUIRED",
			"status":         "ACTIVE",
		}}

		lists := []map[string]interface{}{map[string]interface{}{
			"id": rData.SiteConfig.CONSTANT_CONTACT_LIST_ID,
		}}

		contactInfo = map[string]interface{}{
			"email_addresses": emailAddresses,
			"first_name":      userRecord.GetData().FirstName,
			"last_name":       userRecord.GetData().LastName,
			"lists":           lists,
		}

		jsonData, err := json.Marshal(contactInfo)
		if err != nil {
			return contactID, err
		}

		buffer := bytes.NewBuffer(jsonData)

		_, userInfo, err := constantContactApiCall(rData, "POST", "contacts", "action_by=ACTION_BY_OWNER", buffer)

		if err != nil {
			return contactID, err
		}

		contactID, ok = userInfo["id"].(string)
		if !ok {

			return contactID, statuscodes.Error(statuscodes.TECHNICAL)
		}

	} else {
		contactID, ok = contactInfo["id"].(string)

		if !ok {
			return contactID, statuscodes.Error(statuscodes.TECHNICAL)
		}

		lists, ok := contactInfo["lists"].([]interface{})

		if !ok {

			return contactID, statuscodes.Error(statuscodes.TECHNICAL)
		}

		isInList := false

		for _, listItemInterface := range lists {
			listItem, ok := listItemInterface.(map[string]interface{})
			if !ok {
				return contactID, statuscodes.Error(statuscodes.TECHNICAL)
			}
			if listItem["id"] == rData.SiteConfig.CONSTANT_CONTACT_LIST_ID {
				isInList = true
				break
			}
		}

		if !isInList {
			lists = append(lists,
				map[string]interface{}{
					"id": rData.SiteConfig.CONSTANT_CONTACT_LIST_ID,
				})

			contactInfo["lists"] = lists

			err := constantContactContactInfoUpdate(rData, contactID, contactInfo)

			if err != nil {
				return contactID, err
			}

		}

	}

	return contactID, nil
}

func constantContactUpdateEmail(rData *pages.RequestData, userRecord *datastore.UserRecord) error {
	contactInfo := constantContactContactInfoGet(rData, userRecord.GetData().UserMailinglistData.EmailId, false)

	if contactInfo == nil {
		return statuscodes.Error("NO CONTACT!")
	}

	contactInfo["email_addresses"] = []map[string]interface{}{map[string]interface{}{
		"email_address":  userRecord.GetData().Email,
		"confirm_status": "NO_CONFIRMATION_REQUIRED",
		"status":         "ACTIVE",
	}}

	err := constantContactContactInfoUpdate(rData, userRecord.GetData().UserMailinglistData.EmailId, contactInfo)

	if err != nil {
		return err
	}

	return nil
}

func constantContactUpdateName(rData *pages.RequestData, userRecord *datastore.UserRecord) error {
	contactInfo := constantContactContactInfoGet(rData, userRecord.GetData().UserMailinglistData.EmailId, false)

	if contactInfo == nil {
		return statuscodes.Error("NO CONTACT!")
	}

	contactInfo["first_name"] = userRecord.GetData().FirstName
	contactInfo["last_name"] = userRecord.GetData().LastName

	err := constantContactContactInfoUpdate(rData, userRecord.GetData().UserMailinglistData.EmailId, contactInfo)

	if err != nil {
		return err
	}

	return nil

}

func constantContactContactInfoUpdate(rData *pages.RequestData, contactID string, contactInfo map[string]interface{}) error {
	jsonData, err := json.Marshal(contactInfo)
	if err != nil {
		return err
	}

	buffer := bytes.NewBuffer(jsonData)

	_, _, err = constantContactApiCall(rData, "PUT", "contacts/"+contactID, "action_by=ACTION_BY_OWNER", buffer)

	return err
}

func constantContactContactInfoGet(rData *pages.RequestData, identifier string, isEmail bool) map[string]interface{} {
	var args string
	var apiName string
	var userInfo map[string]interface{}
	var ok bool
	var results []interface{}

	if isEmail {
		apiName = "contacts"
		args = "email=" + identifier
	} else {
		apiName = "contacts/" + identifier
	}

	_, jsonMap, err := constantContactApiCall(rData, "GET", apiName, args, nil)

	if err != nil {
		return nil
	}

	if isEmail {
		results, ok = jsonMap["results"].([]interface{})

		if !ok {
			return nil
		}

		if len(results) == 0 {
			return nil
		}

		userInfo, ok = results[0].(map[string]interface{})
		if !ok {
			return nil
		}
	} else {
		userInfo = jsonMap
	}

	_, ok = userInfo["id"].(string)
	if !ok {
		return nil
	}

	return userInfo
}

func constantContactApiCall(rData *pages.RequestData, requestType string, apiName string, extraParams string, requestData *bytes.Buffer) (int, map[string]interface{}, error) {
	var httpRequest *http.Request
	var err error

	client := urlfetch.Client(rData.Ctx)

	params := "?api_key=" + rData.SiteConfig.CONSTANT_CONTACT_KEY
	if extraParams != "" {
		params += "&" + extraParams
	}

	url := rData.SiteConfig.CONSTANT_CONTACT_API_ENDPOINT + apiName + params

	if requestData == nil {
		httpRequest, err = http.NewRequest(requestType, url, nil)
	} else {
		httpRequest, err = http.NewRequest(requestType, url, requestData)
	}

	if err != nil {
		return 0, nil, err
	}

	httpRequest.Header.Add("Authorization", "Bearer "+rData.SiteConfig.CONSTANT_CONTACT_TOKEN)
	httpRequest.Header.Add("Content-Type", "application/json")

	httpResponse, err := client.Do(httpRequest)

	if err != nil {
		return 0, nil, err
	}

	defer httpResponse.Body.Close()
	body, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return httpResponse.StatusCode, nil, err
	}

	if httpResponse.StatusCode != 200 && httpResponse.StatusCode != 201 {
		return httpResponse.StatusCode, nil, statuscodes.Error(httpResponse.Status)
	}

	var jsonResponse interface{}
	err = json.Unmarshal(body, &jsonResponse)

	if err != nil {
		return httpResponse.StatusCode, nil, err
	}

	jsonMap := jsonResponse.(map[string]interface{})

	return httpResponse.StatusCode, jsonMap, nil
}
