package datastore

import (
	"strconv"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine/log"

	"github.com/lionelbarrow/braintree-go"
	gaesr "google.golang.org/appengine/search"
)

const USER_TYPE = "User"

type UserData struct {
	Email           string
	UsernameHistory []string
	FirstName       string
	LastName        string
	Password        string
	IsActive        bool
	AvatarId        int64
	Roles           int64
	ParentId        int64
	SubAccountIds   []int64
	Credits         []UserCredits
	MediaAccessIds  []int64
	ContactUserIds  []int64
	AddedDate       time.Time

	UserMailinglistData
}

//these sub-structsjust to make it easier to manage
type UserMailinglistData struct {
	EmailId                string
	ListEmailId            string
	HasMarketingNewsletter bool
}

type UserCredits struct {
	TransactionHistoryId int64 //how the credits were purchased
	AmountRemaining      int64
	CostOfSingleCredit   braintree.Decimal
}

//boilerplate

type UserRecord struct {
	DsRecord
	data *UserData
}

//Satisfy the interface....
func (dsr *UserRecord) GetRawData() interface{} {
	return dsr.GetData()
}

func (dsr *UserRecord) GetType() string {
	return USER_TYPE
}

func (dsr *UserRecord) GetData() *UserData {
	if dsr.data == nil {
		dsr.SetData(&UserData{})
	}
	return dsr.data
}

func (dsr *UserRecord) SetData(newData *UserData) {
	dsr.data = newData
}

/* More fun stuff.... */

func (dsr *UserRecord) GetFullName() string {
	name := dsr.GetData().FirstName

	if name != "" {
		name += " "
	}

	name += dsr.GetData().LastName

	return name
}

func (dsr *UserRecord) GetFullNameShortened() string {
	name := dsr.GetData().FirstName

	if dsr.GetData().LastName == "" {
		return name
	}
	if name != "" {
		name += " "
	}

	name += dsr.GetData().LastName[:1] + "."

	return name
}

func (dsr *UserRecord) GetRemainingCredits() int64 {
	var total int64
	for _, userCredits := range dsr.GetData().Credits {
		total += userCredits.AmountRemaining
	}

	return total
}

func (dsr *UserRecord) AddToSearch(c context.Context) error {
	userID := strconv.FormatInt(dsr.GetKey().IntID(), 10)
	index, err := gaesr.Open(UserSearchType)

	if err != nil {
		log.Errorf(c, "SEARCH_ADD (index open) User ID: %v, %v", userID, err)
		return err
	}

	userInfo := UserSearchData{
		Email:     dsr.GetData().Email,
		FirstName: dsr.GetData().FirstName,
		LastName:  dsr.GetData().LastName,
	}

	_, err = index.Put(c, userID, &userInfo)

	if err != nil {
		log.Errorf(c, "SEARCH_ADD (data entry) User ID: %v, %v", userID, err)
		return err
	}

	return nil
}

/* Username lookup */
const USER_NAME_LOOKUP_TYPE = "UsernameLookup"

type UsernameLookupData struct {
	UserId int64
}

type UsernameLookupRecord struct {
	DsRecord
	data *UsernameLookupData
}

func (dsr *UsernameLookupRecord) GetRawData() interface{} {
	return dsr.GetData()
}
func (dsr *UsernameLookupRecord) GetType() string {
	return USER_NAME_LOOKUP_TYPE
}

func (dsr *UsernameLookupRecord) GetData() *UsernameLookupData {
	if dsr.data == nil {
		dsr.SetData(&UsernameLookupData{})
	}
	return dsr.data
}

func (dsr *UsernameLookupRecord) SetData(newData *UsernameLookupData) {
	dsr.data = newData
}
