package cfnetworkingaction

import (
	"code.cloudfoundry.org/cfnetworking-cli-api/cfnetworking/cfnetv1"
	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/actor/v3action"
)

type Policy struct {
	SourceName         string
	SourceType         string
	DestinationName    string
	DestinationStartIP string
	DestinationEndIP   string
	DestinationType    string
	Protocol           string
	StartPort          int
	EndPort            int
}

func (actor Actor) AddNetworkPolicy(spaceGUID, srcAppName, srcType, destAppName, destIPStart, destIPEnd, protocol string, startPort, endPort int) (Warnings, error) {
	var allWarnings Warnings

	srcApp, warnings, err := actor.V3Actor.GetApplicationByNameAndSpace(srcAppName, spaceGUID)
	allWarnings = append(allWarnings, Warnings(warnings)...)
	if err != nil {
		return allWarnings, err
	}

	policyRequest := cfnetv1.PolicyList{}
	if destAppName != "" {
		var destApp v3action.Application
		destApp, warnings, err = actor.V3Actor.GetApplicationByNameAndSpace(destAppName, spaceGUID)
		allWarnings = append(allWarnings, Warnings(warnings)...)
		if err != nil {
			return allWarnings, err
		}
		policyRequest.Policies = []cfnetv1.Policy{
			{
				Source: cfnetv1.PolicySource{
					ID: srcApp.GUID,
				},
				Destination: cfnetv1.PolicyDestination{
					ID:       destApp.GUID,
					Protocol: cfnetv1.PolicyProtocol(protocol),
					Ports: cfnetv1.Ports{
						Start: startPort,
						End:   endPort,
					},
				},
			},
		}
	} else {
		policyRequest.EgressPolicies = []cfnetv1.EgressPolicy{
			{
				Source: cfnetv1.EgressPolicySource{
					ID:   srcApp.GUID,
					Type: srcType,
				},
				Destination: cfnetv1.EgressPolicyDestination{
					IPs: []cfnetv1.IP{
						{
							Start: destIPStart,
							End:   destIPEnd,
						},
					},
					Protocol: cfnetv1.PolicyProtocol(protocol),
					Ports: []cfnetv1.Ports{
						{
							Start: startPort,
							End:   endPort,
						},
					},
				},
			},
		}
	}

	err = actor.NetworkingClient.CreatePolicies(policyRequest)

	return allWarnings, err
}

func (actor Actor) NetworkPoliciesBySpace(spaceGUID string) ([]Policy, Warnings, error) {
	var allWarnings Warnings

	applications, warnings, err := actor.V3Actor.GetApplicationsBySpace(spaceGUID)
	allWarnings = append(allWarnings, Warnings(warnings)...)
	if err != nil {
		return []Policy{}, allWarnings, err
	}

	policyList, err := actor.NetworkingClient.ListPolicies()
	if err != nil {
		return []Policy{}, allWarnings, err
	}

	appNameByGuid := map[string]string{}
	for _, app := range applications {
		appNameByGuid[app.GUID] = app.Name
	}

	var policies []Policy
	emptyPolicy := Policy{}
	for _, v1Policy := range policyList.Policies {
		policy := actor.transformPolicy(appNameByGuid, v1Policy)
		if policy != emptyPolicy {
			policies = append(policies, policy)
		}
	}

	for _, v1Policy := range policyList.EgressPolicies {
		policy := actor.transformEgressPolicy(appNameByGuid, v1Policy)
		if policy != emptyPolicy {
			policies = append(policies, policy)
		}
	}

	return policies, allWarnings, nil
}

func (actor Actor) NetworkPoliciesBySpaceAndAppName(spaceGUID string, srcAppName string) ([]Policy, Warnings, error) {
	var allWarnings Warnings
	var appGUID string

	applications, warnings, err := actor.V3Actor.GetApplicationsBySpace(spaceGUID)
	allWarnings = append(allWarnings, Warnings(warnings)...)
	if err != nil {
		return []Policy{}, allWarnings, err
	}

	appNameByGuid := map[string]string{}
	for _, app := range applications {
		appNameByGuid[app.GUID] = app.Name
	}

	var v1Policies []cfnetv1.Policy

	srcApp, warnings, err := actor.V3Actor.GetApplicationByNameAndSpace(srcAppName, spaceGUID)
	allWarnings = append(allWarnings, Warnings(warnings)...)
	if err != nil {
		return []Policy{}, allWarnings, err
	}

	appGUID = srcApp.GUID
	policyList, err := actor.NetworkingClient.ListPolicies(appGUID)
	if err != nil {
		return []Policy{}, allWarnings, err
	}
	v1Policies = policyList.Policies
	var policies []Policy
	emptyPolicy := Policy{}
	for _, v1Policy := range v1Policies {
		if v1Policy.Source.ID == appGUID {
			policy := actor.transformPolicy(appNameByGuid, v1Policy)
			if policy != emptyPolicy {
				policies = append(policies, policy)
			}
		}
	}

	return policies, allWarnings, nil
}

func (actor Actor) RemoveNetworkPolicy(spaceGUID, srcAppName, destAppName, protocol string, startPort, endPort int) (Warnings, error) {
	var allWarnings Warnings

	srcApp, warnings, err := actor.V3Actor.GetApplicationByNameAndSpace(srcAppName, spaceGUID)
	allWarnings = append(allWarnings, Warnings(warnings)...)
	if err != nil {
		return allWarnings, err
	}

	destApp, warnings, err := actor.V3Actor.GetApplicationByNameAndSpace(destAppName, spaceGUID)
	allWarnings = append(allWarnings, Warnings(warnings)...)
	if err != nil {
		return allWarnings, err
	}

	policyToRemove := cfnetv1.Policy{
		Source: cfnetv1.PolicySource{
			ID: srcApp.GUID,
		},
		Destination: cfnetv1.PolicyDestination{
			ID:       destApp.GUID,
			Protocol: cfnetv1.PolicyProtocol(protocol),
			Ports: cfnetv1.Ports{
				Start: startPort,
				End:   endPort,
			},
		},
	}

	v1Policies, err := actor.NetworkingClient.ListPolicies(srcApp.GUID)
	if err != nil {
		return allWarnings, err
	}

	for _, v1Policy := range v1Policies.Policies {
		if v1Policy == policyToRemove {
			return allWarnings, actor.NetworkingClient.RemovePolicies([]cfnetv1.Policy{policyToRemove})
		}
	}

	return allWarnings, actionerror.PolicyDoesNotExistError{}
}

func (actor Actor) GetSpaceByNameAndOrganization(spaceName string, orgGUID string) (v3action.Space, Warnings, error) {
	space, warnings, err := actor.V3Actor.GetSpaceByNameAndOrganization(spaceName, orgGUID)
	return space, Warnings(warnings), err
}

func (Actor) transformPolicy(appNameByGuid map[string]string, v1Policy cfnetv1.Policy) Policy {
	srcName, srcOk := appNameByGuid[v1Policy.Source.ID]
	dstName, dstOk := appNameByGuid[v1Policy.Destination.ID]
	if srcOk && dstOk {
		return Policy{
			SourceName:      srcName,
			SourceType:      "app",
			DestinationName: dstName,
			DestinationType: "app",
			Protocol:        string(v1Policy.Destination.Protocol),
			StartPort:       v1Policy.Destination.Ports.Start,
			EndPort:         v1Policy.Destination.Ports.End,
		}
	}
	return Policy{}
}

func (Actor) transformEgressPolicy(appNameByGuid map[string]string, v1Policy cfnetv1.EgressPolicy) Policy {
	srcName, srcOk := appNameByGuid[v1Policy.Source.ID]
	if srcOk {
		var startIP, endIP string
		if len(v1Policy.Destination.IPs) > 0 {
			startIP = v1Policy.Destination.IPs[0].Start
			endIP = v1Policy.Destination.IPs[0].End
		}
		var startPort, endPort int
		if len(v1Policy.Destination.Ports) > 0 {
			startPort = v1Policy.Destination.Ports[0].Start
			endPort = v1Policy.Destination.Ports[0].End
		}
		return Policy{
			SourceName:         srcName,
			SourceType:         v1Policy.Source.Type,
			DestinationStartIP: startIP,
			DestinationEndIP:   endIP,
			DestinationType:    "ip",
			Protocol:           string(v1Policy.Destination.Protocol),
			StartPort:          startPort,
			EndPort:            endPort,
		}
	}
	return Policy{}
}
