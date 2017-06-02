package datastore

import (
	"github.com/dakom/basic-site-api/lib/utils/text"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/net/context"
)

const JWT_TYPE = "JWTLookup"

type JwtData struct {
	//SelfId is something we typically want to avoid with datastore records since the key is always available to the record
	//However, in this case we want it in the data too since jti (as string, not int64 btw) is part of the spec
	//The alternative would be to mix in the record's key ad-hoc and we'd lose the benefit of automatic json marshalling/unmarshalling
	//Since GetData() is a function, it automatically sets SelfId when the data is requested :)
	//So it's a tiny bit of redundency to gain clearer code and significantly better workflow

	SelfId       string `json:"jti,omitempty" datastore:",noindex"`
	Audience     string `json:"aud,omitempty" datastore:",noindex"`
	UserId       int64  `json:"uid,omitempty" datastore:",noindex"`
	UserType     string `json:"ut,omitempty" datastore:",noindex"`
	ExpiresAt    int64  `json:"exp,omitempty"`
	IssuedAt     int64  `json:"iat,omitempty" datastore:",noindex"`
	Scopes       int64  `json:"scopes,omitempty" datastore:",noindex"`
	SessionId    string `json:"sid,omitempty" datastore:",noindex"`
	FinalExpires int64  `json:"fexp,omitempty"`
	Subject      string `json:"sub,omitempty" datastore:",noindex"`
	Extra        string `json:"extra,omitempty" datastore:",noindex"`
}

type JwtRecord struct {
	DsRecord
	data *JwtData
}

func (dsr *JwtRecord) GetRawData() interface{} {
	return dsr.GetData()
}
func (dsr *JwtRecord) GetType() string {
	return JWT_TYPE
}

func (dsr *JwtRecord) GetData() *JwtData {
	if dsr.data == nil {
		dsr.SetData(&JwtData{})
	}

	if dsr.data.SelfId == "" {
		dsr.data.SelfId = dsr.GetKeyIntAsString()
	}

	return dsr.data
}

func (dsr *JwtRecord) SetData(newData *JwtData) {
	dsr.data = newData
}

func (dsr *JwtRecord) SetExtraMap(c context.Context, extraMap map[string]interface{}) error {
	if extraString, err := text.MakeJsonString(extraMap); err != nil {
		return err
	} else {
		dsr.GetData().Extra = extraString
		if err := Save(c, dsr); err != nil {
			return err
		}
	}

	return nil
}

//Make JwtData satisfy claims interface

func (data *JwtData) Valid() error {
	//inherent jwt checking only cares about signing and expirey here
	standardClaims := jwt.StandardClaims{
		ExpiresAt: data.ExpiresAt,
	}

	//return fmt.Errorf("FOO")

	return standardClaims.Valid()
}
