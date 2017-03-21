package datastore

import (
	"fmt"
	"strconv"

	"appengine"

	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"

	"golang.org/x/net/context"
	gaeds "google.golang.org/appengine/datastore"
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
	dsi.SetKey(GetKeyFromVal(c, dsi.GetType(), keyVal, ancestorKeyVal))
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

func CheckMultiGetResults(multiError error) ([]int, error) {
	var failedIndexes []int

	if me, ok := multiError.(appengine.MultiError); ok {
		for idx, merr := range me {
			//if merr is nil, the index did not contain an error
			if merr != nil {
				failedIndexes = append(failedIndexes, idx)
			}
		}
	} else if me != nil {
		return nil, me
	}

	return nil, nil
}

func GetMultiKeysFromInts(c context.Context, kind string, keyVals []int64, commonAncestorKey interface{}) []*gaeds.Key {
	keys := make([]*gaeds.Key, len(keyVals))

	for idx, keyVal := range keyVals {
		keys[idx] = GetKeyFromVal(c, kind, keyVal, commonAncestorKey)
	}

	return keys
}

func GetKeyFromVal(c context.Context, kind string, keyVal interface{}, ancestorKeyVal interface{}) *gaeds.Key {
	if keyVal == nil {
		return nil
	}

	ancestorKey := GetKeyFromVal(c, kind, ancestorKeyVal, nil)

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
