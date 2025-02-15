/*
Copyright 2019 The Cloud-Barista Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package mcir is to manage multi-cloud infra resource
package mcir

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"

	"github.com/cloud-barista/cb-spider/interface/api"
	"github.com/cloud-barista/cb-tumblebug/src/core/common"

	validator "github.com/go-playground/validator/v10"
)

// SpiderImageReqInfoWrapper struct is ...
type SpiderImageReqInfoWrapper struct { // Spider
	ConnectionName string
	ReqInfo        SpiderImageInfo
}

// SpiderImageInfo struct is ...
type SpiderImageInfo struct { // Spider
	// Fields for request
	Name string

	// Fields for response
	IId          common.IID // {NameId, SystemId}
	GuestOS      string     // Windows7, Ubuntu etc.
	Status       string     // available, unavailable
	KeyValueList []common.KeyValue
}

// TbImageReq struct is for image create request
type TbImageReq struct {
	Name           string `json:"name" validate:"required"`
	ConnectionName string `json:"connectionName" validate:"required"`
	CspImageId     string `json:"cspImageId" validate:"required"`
	Description    string `json:"description"`
}

// TbImageReqStructLevelValidation func is for Validation
func TbImageReqStructLevelValidation(sl validator.StructLevel) {

	u := sl.Current().Interface().(TbImageReq)

	err := common.CheckString(u.Name)
	if err != nil {
		// ReportError(field interface{}, fieldName, structFieldName, tag, param string)
		sl.ReportError(u.Name, "name", "Name", "NotObeyingNamingConvention", "")
	}
}

// TbImageInfo struct is for image object
type TbImageInfo struct {
	Namespace            string            `json:"namespace,omitempty"` // required to save in RDB
	Id                   string            `json:"id,omitempty"`
	Name                 string            `json:"name,omitempty"`
	ConnectionName       string            `json:"connectionName,omitempty"`
	CspImageId           string            `json:"cspImageId,omitempty"`
	CspImageName         string            `json:"cspImageName,omitempty"`
	Description          string            `json:"description,omitempty"`
	CreationDate         string            `json:"creationDate,omitempty"`
	GuestOS              string            `json:"guestOS,omitempty"` // Windows7, Ubuntu etc.
	Status               string            `json:"status,omitempty"`  // available, unavailable
	KeyValueList         []common.KeyValue `json:"keyValueList,omitempty"`
	AssociatedObjectList []string          `json:"associatedObjectList,omitempty"`
	IsAutoGenerated      bool              `json:"isAutoGenerated,omitempty"`
}

// ConvertSpiderImageToTumblebugImage accepts an Spider image object, converts to and returns an TB image object
func ConvertSpiderImageToTumblebugImage(spiderImage SpiderImageInfo) (TbImageInfo, error) {
	if spiderImage.IId.NameId == "" {
		err := fmt.Errorf("ConvertSpiderImageToTumblebugImage failed; spiderImage.IId.NameId == \"\" ")
		emptyTumblebugImage := TbImageInfo{}
		return emptyTumblebugImage, err
	}

	tumblebugImage := TbImageInfo{}
	//tumblebugImage.Id = spiderImage.IId.NameId

	spiderKeyValueListName := common.LookupKeyValueList(spiderImage.KeyValueList, "Name")
	if len(spiderKeyValueListName) > 0 {
		tumblebugImage.Name = spiderKeyValueListName
	} else {
		tumblebugImage.Name = spiderImage.IId.NameId
	}

	tumblebugImage.CspImageId = spiderImage.IId.NameId
	tumblebugImage.CspImageName = common.LookupKeyValueList(spiderImage.KeyValueList, "Name")
	tumblebugImage.Description = common.LookupKeyValueList(spiderImage.KeyValueList, "Description")
	tumblebugImage.CreationDate = common.LookupKeyValueList(spiderImage.KeyValueList, "CreationDate")
	tumblebugImage.GuestOS = spiderImage.GuestOS
	tumblebugImage.Status = spiderImage.Status
	tumblebugImage.KeyValueList = spiderImage.KeyValueList

	return tumblebugImage, nil
}

// RegisterImageWithId accepts image creation request, creates and returns an TB image object
func RegisterImageWithId(nsId string, u *TbImageReq) (TbImageInfo, error) {

	resourceType := common.StrImage

	err := common.CheckString(nsId)
	if err != nil {
		temp := TbImageInfo{}
		common.CBLog.Error(err)
		return temp, err
	}

	// returns InvalidValidationError for bad validation input, nil or ValidationErrors ( []FieldError )
	err = validate.Struct(u)
	if err != nil {

		// this check is only needed when your code could produce
		// an invalid value for validation such as interface with nil
		// value most including myself do not usually have code like this.
		if _, ok := err.(*validator.InvalidValidationError); ok {
			fmt.Println(err)
			temp := TbImageInfo{}
			return temp, err
		}

		// for _, err := range err.(validator.ValidationErrors) {

		// 	fmt.Println(err.Namespace()) // can differ when a custom TagNameFunc is registered or
		// 	fmt.Println(err.Field())     // by passing alt name to ReportError like below
		// 	fmt.Println(err.StructNamespace())
		// 	fmt.Println(err.StructField())
		// 	fmt.Println(err.Tag())
		// 	fmt.Println(err.ActualTag())
		// 	fmt.Println(err.Kind())
		// 	fmt.Println(err.Type())
		// 	fmt.Println(err.Value())
		// 	fmt.Println(err.Param())
		// 	fmt.Println()
		// }

		temp := TbImageInfo{}
		return temp, err
	}

	check, err := CheckResource(nsId, resourceType, u.Name)

	if check {
		temp := TbImageInfo{}
		err := fmt.Errorf("The image " + u.Name + " already exists.")
		return temp, err
	}

	if err != nil {
		temp := TbImageInfo{}
		err := fmt.Errorf("Failed to check the existence of the image " + u.Name + ".")
		return temp, err
	}

	res, err := LookupImage(u.ConnectionName, u.CspImageId)
	if err != nil {
		common.CBLog.Error(err)
		//err := fmt.Errorf("an error occurred while lookup image via CB-Spider")
		emptyImageInfoObj := TbImageInfo{}
		return emptyImageInfoObj, err
	}

	content, err := ConvertSpiderImageToTumblebugImage(res)
	if err != nil {
		common.CBLog.Error(err)
		//err := fmt.Errorf("an error occurred while converting Spider image info to Tumblebug image info.")
		emptyImageInfoObj := TbImageInfo{}
		return emptyImageInfoObj, err
	}
	content.Namespace = nsId
	content.ConnectionName = u.ConnectionName
	content.Id = u.Name
	content.Name = u.Name
	content.AssociatedObjectList = []string{}

	// cb-store
	fmt.Println("=========================== PUT registerImage")
	Key := common.GenResourceKey(nsId, resourceType, content.Id)
	Val, _ := json.Marshal(content)
	err = common.CBStore.Put(Key, string(Val))
	if err != nil {
		common.CBLog.Error(err)
		return content, err
	}
	keyValue, err := common.CBStore.Get(Key)
	if err != nil {
		fmt.Println("In RegisterImageWithId(); CBStore.Get() returned error.")
	}
	fmt.Println("<" + keyValue.Key + "> \n" + keyValue.Value)
	fmt.Println("===========================")

	// "INSERT INTO `image`(`namespace`, `id`, ...) VALUES ('nsId', 'content.Id', ...);
	_, err = common.ORM.Insert(&content)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Data inserted successfully..")
	}

	return content, nil
}

// RegisterImageWithInfo accepts image creation request, creates and returns an TB image object
func RegisterImageWithInfo(nsId string, content *TbImageInfo) (TbImageInfo, error) {

	resourceType := common.StrImage

	err := common.CheckString(nsId)
	if err != nil {
		temp := TbImageInfo{}
		common.CBLog.Error(err)
		return temp, err
	}
	err = common.CheckString(content.Name)
	if err != nil {
		temp := TbImageInfo{}
		common.CBLog.Error(err)
		return temp, err
	}
	check, err := CheckResource(nsId, resourceType, content.Name)

	if check {
		temp := TbImageInfo{}
		err := fmt.Errorf("The image " + content.Name + " already exists.")
		return temp, err
	}

	if err != nil {
		temp := TbImageInfo{}
		err := fmt.Errorf("Failed to check the existence of the image " + content.Name + ".")
		return temp, err
	}

	content.Namespace = nsId
	//content.Id = common.GenUid()
	content.Id = content.Name
	content.AssociatedObjectList = []string{}

	fmt.Println("=========================== PUT registerImage")
	Key := common.GenResourceKey(nsId, resourceType, content.Id)
	Val, _ := json.Marshal(content)
	err = common.CBStore.Put(Key, string(Val))
	if err != nil {
		common.CBLog.Error(err)
		return *content, err
	}
	keyValue, err := common.CBStore.Get(Key)
	if err != nil {
		fmt.Println("In RegisterImageWithInfo(); CBStore.Get() returned error.")
	}
	fmt.Println("<" + keyValue.Key + "> \n" + keyValue.Value)
	fmt.Println("===========================")

	// "INSERT INTO `image`(`namespace`, `id`, ...) VALUES ('nsId', 'content.Id', ...);
	_, err = common.ORM.Insert(content)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Data inserted successfully..")
	}

	return *content, nil
}

// SpiderImageList is struct for Spider Image List
type SpiderImageList struct {
	Image []SpiderImageInfo `json:"image"`
}

// LookupImageList accepts Spider conn config,
// lookups and returns the list of all images in the region of conn config
// in the form of the list of Spider image objects
func LookupImageList(connConfig string) (SpiderImageList, error) {

	if connConfig == "" {
		content := SpiderImageList{}
		err := fmt.Errorf("LookupImage() called with empty connConfig.")
		common.CBLog.Error(err)
		return content, err
	}

	if os.Getenv("SPIDER_CALL_METHOD") == "REST" {

		url := common.SpiderRestUrl + "/vmimage"

		// Create Req body
		tempReq := common.SpiderConnectionName{}
		tempReq.ConnectionName = connConfig

		client := resty.New().SetCloseConnection(true)
		client.SetAllowGetMethodPayload(true)

		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(tempReq).
			SetResult(&SpiderImageList{}). // or SetResult(AuthSuccess{}).
			//SetError(&AuthError{}).       // or SetError(AuthError{}).
			Get(url)

		if err != nil {
			common.CBLog.Error(err)
			content := SpiderImageList{}
			err := fmt.Errorf("an error occurred while requesting to CB-Spider")
			return content, err
		}

		fmt.Println(string(resp.Body()))

		fmt.Println("HTTP Status code: " + strconv.Itoa(resp.StatusCode()))
		switch {
		case resp.StatusCode() >= 400 || resp.StatusCode() < 200:
			err := fmt.Errorf(string(resp.Body()))
			common.CBLog.Error(err)
			content := SpiderImageList{}
			return content, err
		}

		temp := resp.Result().(*SpiderImageList)
		return *temp, nil

	} else {

		// Set CCM gRPC API
		ccm := api.NewCloudResourceHandler()
		err := ccm.SetConfigPath(os.Getenv("CBTUMBLEBUG_ROOT") + "/conf/grpc_conf.yaml")
		if err != nil {
			common.CBLog.Error("ccm failed to set config : ", err)
			return SpiderImageList{}, err
		}
		err = ccm.Open()
		if err != nil {
			common.CBLog.Error("ccm api open failed : ", err)
			return SpiderImageList{}, err
		}
		defer ccm.Close()

		result, err := ccm.ListImageByParam(connConfig)
		if err != nil {
			common.CBLog.Error(err)
			return SpiderImageList{}, err
		}

		temp := SpiderImageList{}
		err = json.Unmarshal([]byte(result), &temp)
		if err != nil {
			common.CBLog.Error(err)
			return SpiderImageList{}, err
		}
		return temp, nil

	}
}

// LookupImage accepts Spider conn config and CSP image ID, lookups and returns the Spider image object
func LookupImage(connConfig string, imageId string) (SpiderImageInfo, error) {

	if connConfig == "" {
		content := SpiderImageInfo{}
		err := fmt.Errorf("LookupImage() called with empty connConfig.")
		common.CBLog.Error(err)
		return content, err
	} else if imageId == "" {
		content := SpiderImageInfo{}
		err := fmt.Errorf("LookupImage() called with empty imageId.")
		common.CBLog.Error(err)
		return content, err
	}

	if os.Getenv("SPIDER_CALL_METHOD") == "REST" {

		url := common.SpiderRestUrl + "/vmimage/" + url.QueryEscape(imageId)

		// Create Req body
		tempReq := common.SpiderConnectionName{}
		tempReq.ConnectionName = connConfig

		client := resty.New().SetCloseConnection(true)
		client.SetAllowGetMethodPayload(true)

		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(tempReq).
			SetResult(&SpiderImageInfo{}). // or SetResult(AuthSuccess{}).
			//SetError(&AuthError{}).       // or SetError(AuthError{}).
			Get(url)

		if err != nil {
			common.CBLog.Error(err)
			content := SpiderImageInfo{}
			err := fmt.Errorf("an error occurred while requesting to CB-Spider")
			return content, err
		}

		fmt.Println(string(resp.Body()))

		fmt.Println("HTTP Status code: " + strconv.Itoa(resp.StatusCode()))
		switch {
		case resp.StatusCode() >= 400 || resp.StatusCode() < 200:
			err := fmt.Errorf(string(resp.Body()))
			common.CBLog.Error(err)
			content := SpiderImageInfo{}
			return content, err
		}

		temp := resp.Result().(*SpiderImageInfo)
		return *temp, nil

	} else {

		// Set CCM gRPC API
		ccm := api.NewCloudResourceHandler()
		err := ccm.SetConfigPath(os.Getenv("CBTUMBLEBUG_ROOT") + "/conf/grpc_conf.yaml")
		if err != nil {
			common.CBLog.Error("ccm failed to set config : ", err)
			return SpiderImageInfo{}, err
		}
		err = ccm.Open()
		if err != nil {
			common.CBLog.Error("ccm api open failed : ", err)
			return SpiderImageInfo{}, err
		}
		defer ccm.Close()

		result, err := ccm.GetImageByParam(connConfig, imageId)
		if err != nil {
			common.CBLog.Error(err)
			return SpiderImageInfo{}, err
		}

		temp := SpiderImageInfo{}
		err2 := json.Unmarshal([]byte(result), &temp)
		if err2 != nil {
			//fmt.Errorf("an error occurred while unmarshaling: " + err2.Error())
			common.CBLog.Error(err2)
		}
		return temp, nil

	}
}

func RefineImageName(imageName string) string {
	out := strings.ToLower(imageName)
	out = strings.ReplaceAll(out, ".", "-")
	out = strings.ReplaceAll(out, "_", "-")
	out = strings.ReplaceAll(out, ":", "-")
	out = strings.ReplaceAll(out, "/", "-")

	return out
}

// FetchImagesForAllConnConfigs gets all conn configs from Spider, lookups all images for each region of conn config, and saves into TB image objects
func FetchImagesForConnConfig(connConfig string, nsId string) (imageCount uint, err error) {
	fmt.Println("FetchImagesForConnConfig(" + connConfig + ")")

	spiderImageList, err := LookupImageList(connConfig)
	if err != nil {
		common.CBLog.Error(err)
		return 0, err
	}

	for _, spiderImage := range spiderImageList.Image {
		tumblebugImage, err := ConvertSpiderImageToTumblebugImage(spiderImage)
		if err != nil {
			common.CBLog.Error(err)
			return 0, err
		}

		tumblebugImageId := connConfig + "-" + RefineImageName(tumblebugImage.Name)
		//fmt.Println("tumblebugImageId: " + tumblebugImageId) // for debug

		check, err := CheckResource(nsId, common.StrImage, tumblebugImageId)
		if check {
			common.CBLog.Infoln("The image " + tumblebugImageId + " already exists in TB; continue")
			continue
		} else if err != nil {
			common.CBLog.Infoln("Cannot check the existence of " + tumblebugImageId + " in TB; continue")
			continue
		} else {
			tumblebugImage.Name = tumblebugImageId
			tumblebugImage.ConnectionName = connConfig

			_, err := RegisterImageWithInfo(nsId, &tumblebugImage)
			if err != nil {
				common.CBLog.Error(err)
				return 0, err
			}
			imageCount++
		}
	}
	return imageCount, nil
}

// FetchImagesForAllConnConfigs gets all conn configs from Spider, lookups all images for each region of conn config, and saves into TB image objects
func FetchImagesForAllConnConfigs(nsId string) (connConfigCount uint, imageCount uint, err error) {

	err = common.CheckString(nsId)
	if err != nil {
		common.CBLog.Error(err)
		return 0, 0, err
	}

	connConfigs, err := common.GetConnConfigList()
	if err != nil {
		common.CBLog.Error(err)
		return 0, 0, err
	}

	for _, connConfig := range connConfigs.Connectionconfig {
		temp, _ := FetchImagesForConnConfig(connConfig.ConfigName, nsId)
		imageCount += temp
		connConfigCount++
	}
	return connConfigCount, imageCount, nil
}

// SearchImage accepts arbitrary number of keywords, and returns the list of matched TB image objects
func SearchImage(nsId string, keywords ...string) ([]TbImageInfo, error) {

	err := common.CheckString(nsId)
	if err != nil {
		common.CBLog.Error(err)
		return nil, err
	}

	tempList := []TbImageInfo{}

	//sqlQuery := "SELECT * FROM `image` WHERE `namespace`='" + nsId + "'"
	sqlQuery := common.ORM.Where("Namespace = ?", nsId)

	for _, keyword := range keywords {
		keyword = RefineImageName(keyword)
		//sqlQuery += " AND `name` LIKE '%" + keyword + "%'"
		sqlQuery = sqlQuery.And("Name LIKE ?", "%"+keyword+"%")
	}

	err = sqlQuery.Find(&tempList)
	if err != nil {
		common.CBLog.Error(err)
		return tempList, err
	}
	return tempList, nil
}

// UpdateImage accepts to-be TB image objects,
// updates and returns the updated TB image objects
func UpdateImage(nsId string, imageId string, fieldsToUpdate TbImageInfo) (TbImageInfo, error) {
	resourceType := common.StrImage

	err := common.CheckString(nsId)
	if err != nil {
		temp := TbImageInfo{}
		common.CBLog.Error(err)
		return temp, err
	}

	if len(fieldsToUpdate.Namespace) > 0 {
		temp := TbImageInfo{}
		err := fmt.Errorf("You should not specify 'namespace' in the JSON request body.")
		common.CBLog.Error(err)
		return temp, err
	}

	if len(fieldsToUpdate.Id) > 0 {
		temp := TbImageInfo{}
		err := fmt.Errorf("You should not specify 'id' in the JSON request body.")
		common.CBLog.Error(err)
		return temp, err
	}

	check, err := CheckResource(nsId, resourceType, imageId)

	if err != nil {
		temp := TbImageInfo{}
		common.CBLog.Error(err)
		return temp, err
	}

	if !check {
		temp := TbImageInfo{}
		err := fmt.Errorf("The image " + imageId + " does not exist.")
		return temp, err
	}

	tempInterface, err := GetResource(nsId, resourceType, imageId)
	if err != nil {
		temp := TbImageInfo{}
		err := fmt.Errorf("Failed to get the image " + imageId + ".")
		return temp, err
	}
	asIsImage := TbImageInfo{}
	err = common.CopySrcToDest(&tempInterface, &asIsImage)
	if err != nil {
		temp := TbImageInfo{}
		err := fmt.Errorf("Failed to CopySrcToDest() " + imageId + ".")
		return temp, err
	}

	// Update specified fields only
	toBeImage := asIsImage
	toBeImageJSON, _ := json.Marshal(fieldsToUpdate)
	err = json.Unmarshal(toBeImageJSON, &toBeImage)

	// cb-store
	fmt.Println("=========================== PUT UpdateImage")
	Key := common.GenResourceKey(nsId, resourceType, toBeImage.Id)
	Val, _ := json.Marshal(toBeImage)
	err = common.CBStore.Put(Key, string(Val))
	if err != nil {
		temp := TbImageInfo{}
		common.CBLog.Error(err)
		return temp, err
	}
	keyValue, err := common.CBStore.Get(Key)
	if err != nil {
		common.CBLog.Error(err)
		err = fmt.Errorf("In UpdateImage(); CBStore.Get() returned an error.")
		common.CBLog.Error(err)
		// return nil, err
	}

	fmt.Println("<" + keyValue.Key + "> \n" + keyValue.Value)
	fmt.Println("===========================")

	// "UPDATE `image` SET `id`='" + imageId + "', ... WHERE `namespace`='" + nsId + "' AND `id`='" + imageId + "';"
	_, err = common.ORM.Update(&toBeImage, &TbImageInfo{Namespace: nsId, Id: imageId})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("SQL data updated successfully..")
	}

	return toBeImage, nil
}
