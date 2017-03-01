package datastore

import (
	"errors"
	"fmt"
	"strconv"

	"appengine"

	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"

	"golang.org/x/net/context"
	gaeds "google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

//Generic Interface (covers all specific record types)
type DsInterface interface {
	//Set these in each specific datastore type
	GetRawData() interface{}
	GetType() string

	//These are set in the base DsRecord struct, no need to re-create per type
	SetKey(*gaeds.Key)
	GetKey() *gaeds.Key
}

//Generic Base structure (reduces repetition on all specific record types)
type DsRecord struct {
	Key *gaeds.Key
}

func (dsr *DsRecord) SetKey(k *gaeds.Key) {
	dsr.Key = k
}
func (dsr *DsRecord) GetKey() *gaeds.Key {
	return dsr.Key
}
func (dsr *DsRecord) GetKeyIntAsString() string {
	if dsr.Key == nil {
		return ""
	}
	return strconv.FormatInt(dsr.Key.IntID(), 10)
}

//Functions to operate on the interface
func SetKey(c context.Context, dsi DsInterface, keyVal interface{}) {
	SetAncestorKey(c, dsi, keyVal, nil)
}

func SetAncestorKey(c context.Context, dsi DsInterface, keyVal interface{}, ancestorKeyVal interface{}) {
	dsi.SetKey(getKeyFromVal(c, dsi.GetType(), keyVal, ancestorKeyVal))
}

func LoadFromKey(c context.Context, dsi DsInterface, keyVal interface{}) error {
	SetKey(c, dsi, keyVal)
	return Load(c, dsi)
}

func LoadFromKeyStringAsInt(c context.Context, dsi DsInterface, keyVal string) error {
	keyInt, err := strconv.ParseInt(keyVal, 10, 64)
	if err != nil {
		return err
	}
	SetKey(c, dsi, keyInt)
	return Load(c, dsi)
}

func LoadFromAncestorKey(c context.Context, dsi DsInterface, keyVal interface{}, ancestorKeyVal interface{}) error {
	SetAncestorKey(c, dsi, keyVal, ancestorKeyVal)
	return Load(c, dsi)
}

func Load(c context.Context, dsi DsInterface) error {
	if dsi.GetKey() == nil {
		return fmt.Errorf(statuscodes.MISSINGINFO)
	}

	return gaeds.Get(c, dsi.GetKey(), dsi.GetRawData())
}

func SaveToKey(c context.Context, dsi DsInterface, keyVal interface{}) error {
	SetKey(c, dsi, keyVal)
	return Save(c, dsi)
}

func SaveToAncestorKey(c context.Context, dsi DsInterface, keyVal interface{}, ancestorKeyVal interface{}) error {
	SetAncestorKey(c, dsi, keyVal, ancestorKeyVal)
	return Save(c, dsi)
}

func SaveToKeyStringAsInt(c context.Context, dsi DsInterface, keyVal string) error {
	keyInt, err := strconv.ParseInt(keyVal, 10, 64)
	if err != nil {
		return err
	}
	SetKey(c, dsi, keyInt)
	return Save(c, dsi)
}

func Save(c context.Context, dsi DsInterface) error {
	if dsi.GetKey() == nil {
		return fmt.Errorf(statuscodes.MISSINGINFO)
	}

	_, err := gaeds.Put(c, dsi.GetKey(), dsi.GetRawData())
	return (err)
}

func SaveToAutoKey(c context.Context, dsi DsInterface) error {
	return SaveToAutoAncestorKey(c, dsi, nil)
}

//func SaveToAutoAncestorKey(c context.Context, dsi DsInterface, parentKey *gaeds.Key) error {
func SaveToAutoAncestorKey(c context.Context, dsi DsInterface, parentKey *gaeds.Key) error {

	incompleteKey := gaeds.NewIncompleteKey(c, dsi.GetType(), parentKey)
	newKey, err := gaeds.Put(c, incompleteKey, dsi.GetRawData())

	if err != nil {
		return err
	}

	SetKey(c, dsi, newKey)
	return nil

	//Really necessary?
	/*
		_, err = gaeds.Put(c, dsi.GetKey(), dsi.GetRawData())
		return (err)
	*/
}

func Delete(c context.Context, dsi DsInterface) error {
	if dsi.GetKey() == nil {
		return fmt.Errorf(statuscodes.MISSINGINFO)
	}

	return gaeds.Delete(c, dsi.GetKey())
}

func GetMultiKeys(c context.Context, kind string, keyVals []interface{}, ancestorKeyVals []interface{}) []*gaeds.Key {
	keys := make([]*gaeds.Key, len(keyVals))

	for idx, keyVal := range keyVals {
		var ancestorKeyVal interface{}
		if ancestorKeyVals != nil {
			if idx < len(ancestorKeyVals) {
				ancestorKeyVal = ancestorKeyVals[idx]
			}
		}
		keys[idx] = getKeyFromVal(c, kind, keyVal, ancestorKeyVal)
	}

	return keys
}

func GetMultiKeysInt(c context.Context, kind string, keyVals []int64, ancestorKeyVals []int64) []*gaeds.Key {
	keys := make([]*gaeds.Key, len(keyVals))

	for idx, keyVal := range keyVals {
		var ancestorKeyVal interface{}
		if ancestorKeyVals != nil {
			if idx < len(ancestorKeyVals) {
				ancestorKeyVal = ancestorKeyVals[idx]
			}
		}
		keys[idx] = getKeyFromVal(c, kind, keyVal, ancestorKeyVal)
	}

	return keys
}

func GetMultiDataSimpleInt(c context.Context, kind string, keyVals []int64) ([]interface{}, error) {
	return GetMultiDataInt(c, kind, keyVals, nil, true)
}

func GetMultiDataSameAncestorInt(c context.Context, kind string, keyVals []int64, ancestorKeyVal int64, checkLength bool) ([]interface{}, error) {
	keyLen := len(keyVals)

	ancestorKeyList := make([]int64, keyLen)

	for i := 0; i < keyLen; i++ {
		ancestorKeyList[i] = ancestorKeyVal
	}

	return GetMultiDataInt(c, kind, keyVals, ancestorKeyList, checkLength)
}

func GetMultiDataInt(c context.Context, kind string, keyVals []int64, ancestorKeyVals []int64, checkLength bool) ([]interface{}, error) {
	keys := GetMultiKeysInt(c, kind, keyVals, ancestorKeyVals)

	var tempDatas interface{}
	var targetRecords []interface{}
	var err error

	if kind == USER_TYPE {
		tempDatas = make([]*UserData, len(keys))
	}

	multiError := gaeds.GetMulti(c, keys, tempDatas)

	validKeys := make([]bool, len(keys))
	for idx, _ := range validKeys {
		validKeys[idx] = true
	}

	if multiError != nil {
		if me, ok := multiError.(appengine.MultiError); ok {
			for idx, merr := range me {
				if merr != nil {
					validKeys[idx] = false
				} else {

					log.Errorf(c, "Errorin getMulti for ID %v: %v", keys[idx].IntID(), merr)
				}
			}
		}
	}

	for idx, isValid := range validKeys {
		if isValid {
			if kind == USER_TYPE {
				targetRecord := UserRecord{
					DsRecord: DsRecord{
						Key: keys[idx],
					},
				}
				targetRecord.SetData(tempDatas.([]*UserData)[idx])
				targetRecords = append(targetRecords, &targetRecord)
			}

		}
	}

	if checkLength == true {
		if len(targetRecords) != len(keys) {
			err = errors.New(statuscodes.RECORD_LENGTH_MISMATCH)
		}
	}

	return targetRecords, err
}

func getKeyFromVal(c context.Context, kind string, keyVal interface{}, ancestorKeyVal interface{}) *gaeds.Key {
	if keyVal == nil {
		return nil
	}

	ancestorKey := getKeyFromVal(c, kind, ancestorKeyVal, nil)

	switch keyVal := keyVal.(type) {
	case *gaeds.Key:
		return keyVal
	case int:
		if keyVal == 0 {
			return nil
		} else {
			return gaeds.NewKey(c, kind, "", int64(keyVal), ancestorKey)
		}
	case int32:
		if keyVal == 0 {
			return nil
		} else {
			return gaeds.NewKey(c, kind, "", int64(keyVal), ancestorKey)
		}
	case int64:
		if keyVal == 0 {
			return nil
		} else {
			return gaeds.NewKey(c, kind, "", keyVal, ancestorKey)
		}

	case string:
		if keyVal == "" {
			return nil
		} else {
			return gaeds.NewKey(c, kind, keyVal, 0, ancestorKey)
		}
	}

	return nil
}
