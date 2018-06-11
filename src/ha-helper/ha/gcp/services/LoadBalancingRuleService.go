package services

import (
	"encoding/json"
	"ha-helper/ha/common/beans"
	"ha-helper/ha/common/constants"
	commoninterfaces "ha-helper/ha/common/interfaces"
	gcpbeans "ha-helper/ha/gcp/beans"
	gcputils "ha-helper/ha/gcp/utils"
	"log"
	"strings"
)

type LoadBalancingRuleService struct {
	svc commoninterfaces.IServiceClient
}

func (lbRuleService *LoadBalancingRuleService) Initialize(params ...interface{}) {

	lbRuleService.svc = params[0].(commoninterfaces.IServiceClient)

}

func (lbRuleService *LoadBalancingRuleService) GetLBRule(loadBalancingRuleName string, regionName string) (*gcpbeans.LoadBalancingRule, bool) {

	var forwardingRuleAPIUrl, responseStr, responseCode string
	var loadBalancingRuleResult *gcpbeans.LoadBalancingRule = &gcpbeans.LoadBalancingRule{}
	var returnValue bool
	var params beans.IaaSDescriptors = lbRuleService.svc.GetIaaSDescriptors()

	forwardingRuleAPIUrl = params.ManagementURL + "/compute/v1/projects/" + params.ProjectId + "/regions/" + regionName + "/forwardingRules/" + loadBalancingRuleName
	/*
		REST API format for getting the details of load balancing rule of a given load balancer is as follows.

		GET https://www.googleapis.com/compute/v1/projects/{project}/regions/{region}/forwardingRules/{resourceId}

	*/
	responseStr, responseCode, returnValue = lbRuleService.svc.InvokeAPI("GET", forwardingRuleAPIUrl, lbRuleService.svc.GetCommonRequestHeaders(), nil)
	if returnValue == true {
		err := json.Unmarshal([]byte(responseStr), loadBalancingRuleResult)
		if err != nil {
			log.Println("Error occurred while unmarshalling load balancing rule details ", err.Error())
			return nil, false
		}
		log.Println("Load balancing rule with name : ", loadBalancingRuleResult.Name, "and id ", loadBalancingRuleResult.ID, "retrieved successfully.")
		return loadBalancingRuleResult, true
	} else {
		if strings.Compare(responseCode, constants.HTTP_STATUS_NOT_FOUND) != 0 {
			// if GetLBRule() is called before the rule is created, 404 status will be returned. If the error is something other than 404,
			// then something wrong might've happened. Lets log it explicitly for later analysis.
			log.Println("Error occurred during LoadBalancingRuleService GetLBRule call : HTTP Status: ", responseCode, " error: ", responseStr)
			return nil, false
		} else {
			// 404. It could be the case that a load balancing rule is yet to be created. Lets return true.
			return nil, true
		}
	}

}

func (lbRuleService *LoadBalancingRuleService) CreateLBRule(createLBRuleInput gcpbeans.CreateLBRuleInput, regionName string) bool {

	var createLBRuleAPIUrl, responseStr, responseCode string
	var returnValue bool
	var currentOperation *gcpbeans.Operation = &gcpbeans.Operation{}
	var params beans.IaaSDescriptors = lbRuleService.svc.GetIaaSDescriptors()

	createLBRuleAPIUrl = params.ManagementURL + "/compute/v1/projects/" + params.ProjectId + "/regions/" + regionName + "/forwardingRules"
	/*
		The REST API URL format is as follows.
		POST https://www.googleapis.com/compute/v1/projects/{project}/regions/{region}/forwardingRules
	*/
	responseStr, responseCode, returnValue = lbRuleService.svc.InvokeAPI("POST", createLBRuleAPIUrl, lbRuleService.svc.GetCommonRequestHeaders(), createLBRuleInput)
	if returnValue == true {
		err := json.Unmarshal([]byte(responseStr), currentOperation)
		if err != nil {
			log.Println("Error occurred while unmarshalling operation details while creating load balancing rule", err.Error())
			return false
		}
		log.Println("Load balancing rule creation :", createLBRuleInput.Name, " initiated successfully.")
		// let's check if the operation status is successful.
		return lbRuleService.IsProvisioningSuccessful(*currentOperation)
	} else {
		log.Println("Error occurred during load balancing rule creation call : HTTP Status: ", responseCode, " error: ", responseStr)
		return false
	}

}

func (lbRuleService *LoadBalancingRuleService) IsProvisioningSuccessful(operation gcpbeans.Operation) bool {
	return gcputils.IsResourceProvisioningSuccessful(operation, lbRuleService.svc)
}
