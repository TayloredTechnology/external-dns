/*
Copyright 2017 The Kubernetes Authors.

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

package provider

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"testing"

	ibclient "github.com/infobloxopen/infoblox-go-client"
	"github.com/kubernetes-incubator/external-dns/endpoint"
	"github.com/kubernetes-incubator/external-dns/plan"
	"github.com/stretchr/testify/assert"
)

type mockIBConnector struct {
	mockInfobloxZones   *[]ibclient.ZoneAuth
	mockInfobloxObjects *[]ibclient.IBObject
	createdEndpoints    []*endpoint.Endpoint
	deletedEndpoints    []*endpoint.Endpoint
	updatedEndpoints    []*endpoint.Endpoint
}

func (client *mockIBConnector) CreateObject(obj ibclient.IBObject) (ref string, err error) {
	switch obj.ObjectType() {
	case "record:a":
		client.createdEndpoints = append(
			client.createdEndpoints,
			endpoint.NewEndpoint(
				obj.(*ibclient.RecordA).Name,
				obj.(*ibclient.RecordA).Ipv4Addr,
				endpoint.RecordTypeA,
			),
		)
		ref = fmt.Sprintf("%s/%s:%s/default", obj.ObjectType(), base64.StdEncoding.EncodeToString([]byte(obj.(*ibclient.RecordA).Name)), obj.(*ibclient.RecordA).Name)
		obj.(*ibclient.RecordA).Ref = ref
	case "record:cname":
		client.createdEndpoints = append(
			client.createdEndpoints,
			endpoint.NewEndpoint(
				obj.(*ibclient.RecordCNAME).Name,
				obj.(*ibclient.RecordCNAME).Canonical,
				endpoint.RecordTypeCNAME,
			),
		)
		ref = fmt.Sprintf("%s/%s:%s/default", obj.ObjectType(), base64.StdEncoding.EncodeToString([]byte(obj.(*ibclient.RecordCNAME).Name)), obj.(*ibclient.RecordCNAME).Name)
		obj.(*ibclient.RecordCNAME).Ref = ref
	case "record:host":
		for _, i := range obj.(*ibclient.RecordHost).Ipv4Addrs {
			client.createdEndpoints = append(
				client.createdEndpoints,
				endpoint.NewEndpoint(
					obj.(*ibclient.RecordHost).Name,
					i.Ipv4Addr,
					endpoint.RecordTypeA,
				),
			)
		}
		ref = fmt.Sprintf("%s/%s:%s/default", obj.ObjectType(), base64.StdEncoding.EncodeToString([]byte(obj.(*ibclient.RecordHost).Name)), obj.(*ibclient.RecordHost).Name)
		obj.(*ibclient.RecordHost).Ref = ref
	case "record:txt":
		client.createdEndpoints = append(
			client.createdEndpoints,
			endpoint.NewEndpoint(
				obj.(*ibclient.RecordTXT).Name,
				obj.(*ibclient.RecordTXT).Text,
				endpoint.RecordTypeTXT,
			),
		)
		obj.(*ibclient.RecordTXT).Ref = ref
		ref = fmt.Sprintf("%s/%s:%s/default", obj.ObjectType(), base64.StdEncoding.EncodeToString([]byte(obj.(*ibclient.RecordTXT).Name)), obj.(*ibclient.RecordTXT).Name)
	}
	*client.mockInfobloxObjects = append(
		*client.mockInfobloxObjects,
		obj,
	)
	return ref, nil
}

func (client *mockIBConnector) GetObject(obj ibclient.IBObject, ref string, res interface{}) (err error) {
	switch obj.ObjectType() {
	case "record:a":
		var result []ibclient.RecordA
		for _, object := range *client.mockInfobloxObjects {
			if object.ObjectType() == "record:a" {
				if ref != "" &&
					ref != object.(*ibclient.RecordA).Ref {
					continue
				}
				if obj.(*ibclient.RecordA).Name != "" &&
					obj.(*ibclient.RecordA).Name != object.(*ibclient.RecordA).Name {
					continue
				}
				result = append(result, *object.(*ibclient.RecordA))
			}
		}
		*res.(*[]ibclient.RecordA) = result
	case "record:cname":
		var result []ibclient.RecordCNAME
		for _, object := range *client.mockInfobloxObjects {
			if object.ObjectType() == "record:cname" {
				if ref != "" &&
					ref != object.(*ibclient.RecordCNAME).Ref {
					continue
				}
				if obj.(*ibclient.RecordCNAME).Name != "" &&
					obj.(*ibclient.RecordCNAME).Name != object.(*ibclient.RecordCNAME).Name {
					continue
				}
				result = append(result, *object.(*ibclient.RecordCNAME))
			}
		}
		*res.(*[]ibclient.RecordCNAME) = result
	case "record:host":
		var result []ibclient.RecordHost
		for _, object := range *client.mockInfobloxObjects {
			if object.ObjectType() == "record:host" {
				if ref != "" &&
					ref != object.(*ibclient.RecordHost).Ref {
					continue
				}
				if obj.(*ibclient.RecordHost).Name != "" &&
					obj.(*ibclient.RecordHost).Name != object.(*ibclient.RecordHost).Name {
					continue
				}
				result = append(result, *object.(*ibclient.RecordHost))
			}
		}
		*res.(*[]ibclient.RecordHost) = result
	case "record:txt":
		var result []ibclient.RecordTXT
		for _, object := range *client.mockInfobloxObjects {
			if object.ObjectType() == "record:txt" {
				if ref != "" &&
					ref != object.(*ibclient.RecordTXT).Ref {
					continue
				}
				if obj.(*ibclient.RecordTXT).Name != "" &&
					obj.(*ibclient.RecordTXT).Name != object.(*ibclient.RecordTXT).Name {
					continue
				}
				result = append(result, *object.(*ibclient.RecordTXT))
			}
		}
		*res.(*[]ibclient.RecordTXT) = result
	case "zone_auth":
		*res.(*[]ibclient.ZoneAuth) = *client.mockInfobloxZones
	}
	return
}

func (client *mockIBConnector) DeleteObject(ref string) (refRes string, err error) {
	re, _ := regexp.Compile(`([^/]+)/[^:]+:([^/]+)/default`)
	result := re.FindStringSubmatch(ref)

	switch result[1] {
	case "record:a":
		var records []ibclient.RecordA
		obj := ibclient.NewRecordA(
			ibclient.RecordA{
				Name: result[2],
			},
		)
		client.GetObject(obj, ref, &records)
		for _, record := range records {
			client.deletedEndpoints = append(
				client.deletedEndpoints,
				endpoint.NewEndpoint(
					record.Name,
					"",
					endpoint.RecordTypeA,
				),
			)
		}
	case "record:cname":
		var records []ibclient.RecordCNAME
		obj := ibclient.NewRecordCNAME(
			ibclient.RecordCNAME{
				Name: result[2],
			},
		)
		client.GetObject(obj, ref, &records)
		for _, record := range records {
			client.deletedEndpoints = append(
				client.deletedEndpoints,
				endpoint.NewEndpoint(
					record.Name,
					"",
					endpoint.RecordTypeCNAME,
				),
			)
		}
	case "record:host":
		var records []ibclient.RecordHost
		obj := ibclient.NewRecordHost(
			ibclient.RecordHost{
				Name: result[2],
			},
		)
		client.GetObject(obj, ref, &records)
		for _, record := range records {
			client.deletedEndpoints = append(
				client.deletedEndpoints,
				endpoint.NewEndpoint(
					record.Name,
					"",
					endpoint.RecordTypeA,
				),
			)
		}
	case "record:txt":
		var records []ibclient.RecordTXT
		obj := ibclient.NewRecordTXT(
			ibclient.RecordTXT{
				Name: result[2],
			},
		)
		client.GetObject(obj, ref, &records)
		for _, record := range records {
			client.deletedEndpoints = append(
				client.deletedEndpoints,
				endpoint.NewEndpoint(
					record.Name,
					"",
					endpoint.RecordTypeTXT,
				),
			)
		}
	}
	return "", nil
}

func (client *mockIBConnector) UpdateObject(obj ibclient.IBObject, ref string) (refRes string, err error) {
	switch obj.ObjectType() {
	case "record:a":
		client.updatedEndpoints = append(
			client.updatedEndpoints,
			endpoint.NewEndpoint(
				obj.(*ibclient.RecordA).Name,
				obj.(*ibclient.RecordA).Ipv4Addr,
				endpoint.RecordTypeA,
			),
		)
	case "record:cname":
		client.updatedEndpoints = append(
			client.updatedEndpoints,
			endpoint.NewEndpoint(
				obj.(*ibclient.RecordCNAME).Name,
				obj.(*ibclient.RecordCNAME).Canonical,
				endpoint.RecordTypeCNAME,
			),
		)
	case "record:host":
		for _, i := range obj.(*ibclient.RecordHost).Ipv4Addrs {
			client.updatedEndpoints = append(
				client.updatedEndpoints,
				endpoint.NewEndpoint(
					obj.(*ibclient.RecordHost).Name,
					i.Ipv4Addr,
					endpoint.RecordTypeA,
				),
			)
		}
	case "record:txt":
		client.updatedEndpoints = append(
			client.updatedEndpoints,
			endpoint.NewEndpoint(
				obj.(*ibclient.RecordTXT).Name,
				obj.(*ibclient.RecordTXT).Text,
				endpoint.RecordTypeTXT,
			),
		)
	}
	return "", nil
}

func createMockInfobloxZone(fqdn string) ibclient.ZoneAuth {
	return ibclient.ZoneAuth{
		Fqdn: fqdn,
	}
}

func createMockInfobloxObject(name, recordType, value string) ibclient.IBObject {
	ref := fmt.Sprintf("record:%s/%s:%s/default", strings.ToLower(recordType), base64.StdEncoding.EncodeToString([]byte(name)), name)
	switch recordType {
	case endpoint.RecordTypeA:
		return ibclient.NewRecordA(
			ibclient.RecordA{
				Ref:      ref,
				Name:     name,
				Ipv4Addr: value,
			},
		)
	case endpoint.RecordTypeCNAME:
		return ibclient.NewRecordCNAME(
			ibclient.RecordCNAME{
				Ref:       ref,
				Name:      name,
				Canonical: value,
			},
		)
	case endpoint.RecordTypeTXT:
		return ibclient.NewRecordTXT(
			ibclient.RecordTXT{
				Ref:  ref,
				Name: name,
				Text: value,
			},
		)
	}
	return nil
}

func newInfobloxProvider(domainFilter DomainFilter, dryRun bool, client ibclient.IBConnector) *InfobloxProvider {
	return &InfobloxProvider{
		client:       client,
		domainFilter: domainFilter,
		dryRun:       dryRun,
	}
}

func TestInfobloxRecords(t *testing.T) {
	client := mockIBConnector{
		mockInfobloxZones: &[]ibclient.ZoneAuth{
			createMockInfobloxZone("example.com"),
		},
		mockInfobloxObjects: &[]ibclient.IBObject{
			createMockInfobloxObject("example.com", endpoint.RecordTypeA, "123.123.123.122"),
			createMockInfobloxObject("example.com", endpoint.RecordTypeTXT, "heritage=external-dns,external-dns/owner=default"),
			createMockInfobloxObject("nginx.example.com", endpoint.RecordTypeA, "123.123.123.123"),
			createMockInfobloxObject("nginx.example.com", endpoint.RecordTypeTXT, "heritage=external-dns,external-dns/owner=default"),
			createMockInfobloxObject("whitespace.example.com", endpoint.RecordTypeA, "123.123.123.124"),
			createMockInfobloxObject("whitespace.example.com", endpoint.RecordTypeTXT, "heritage=external-dns,external-dns/owner=white space"),
			createMockInfobloxObject("hack.example.com", endpoint.RecordTypeCNAME, "cerberus.infoblox.com"),
		},
	}

	provider := newInfobloxProvider(NewDomainFilter([]string{"example.com"}), true, &client)
	actual, err := provider.Records()

	if err != nil {
		t.Fatal(err)
	}
	expected := []*endpoint.Endpoint{
		endpoint.NewEndpoint("example.com", "123.123.123.122", endpoint.RecordTypeA),
		endpoint.NewEndpoint("example.com", "\"heritage=external-dns,external-dns/owner=default\"", endpoint.RecordTypeTXT),
		endpoint.NewEndpoint("nginx.example.com", "123.123.123.123", endpoint.RecordTypeA),
		endpoint.NewEndpoint("nginx.example.com", "\"heritage=external-dns,external-dns/owner=default\"", endpoint.RecordTypeTXT),
		endpoint.NewEndpoint("whitespace.example.com", "123.123.123.124", endpoint.RecordTypeA),
		endpoint.NewEndpoint("whitespace.example.com", "\"heritage=external-dns,external-dns/owner=white space\"", endpoint.RecordTypeTXT),
		endpoint.NewEndpoint("hack.example.com", "cerberus.infoblox.com", endpoint.RecordTypeCNAME),
	}
	validateEndpoints(t, actual, expected)
}

func TestInfobloxApplyChanges(t *testing.T) {
	client := mockIBConnector{}

	testInfobloxApplyChangesInternal(t, false, &client)

	validateEndpoints(t, client.createdEndpoints, []*endpoint.Endpoint{
		endpoint.NewEndpoint("example.com", "1.2.3.4", endpoint.RecordTypeA),
		endpoint.NewEndpoint("example.com", "tag", endpoint.RecordTypeTXT),
		endpoint.NewEndpoint("foo.example.com", "1.2.3.4", endpoint.RecordTypeA),
		endpoint.NewEndpoint("foo.example.com", "tag", endpoint.RecordTypeTXT),
		endpoint.NewEndpoint("bar.example.com", "other.com", endpoint.RecordTypeCNAME),
		endpoint.NewEndpoint("bar.example.com", "tag", endpoint.RecordTypeTXT),
		endpoint.NewEndpoint("other.com", "5.6.7.8", endpoint.RecordTypeA),
		endpoint.NewEndpoint("other.com", "tag", endpoint.RecordTypeTXT),
		endpoint.NewEndpoint("new.example.com", "111.222.111.222", endpoint.RecordTypeA),
		endpoint.NewEndpoint("newcname.example.com", "other.com", endpoint.RecordTypeCNAME),
	})

	validateEndpoints(t, client.deletedEndpoints, []*endpoint.Endpoint{
		endpoint.NewEndpoint("old.example.com", "", endpoint.RecordTypeA),
		endpoint.NewEndpoint("oldcname.example.com", "", endpoint.RecordTypeCNAME),
		endpoint.NewEndpoint("deleted.example.com", "", endpoint.RecordTypeA),
		endpoint.NewEndpoint("deletedcname.example.com", "", endpoint.RecordTypeCNAME),
	})

	validateEndpoints(t, client.updatedEndpoints, []*endpoint.Endpoint{})
}

func TestInfobloxApplyChangesDryRun(t *testing.T) {
	client := mockIBConnector{
		mockInfobloxObjects: &[]ibclient.IBObject{},
	}

	testInfobloxApplyChangesInternal(t, true, &client)

	validateEndpoints(t, client.createdEndpoints, []*endpoint.Endpoint{})

	validateEndpoints(t, client.deletedEndpoints, []*endpoint.Endpoint{})

	validateEndpoints(t, client.updatedEndpoints, []*endpoint.Endpoint{})
}

func testInfobloxApplyChangesInternal(t *testing.T, dryRun bool, client ibclient.IBConnector) {
	client.(*mockIBConnector).mockInfobloxZones = &[]ibclient.ZoneAuth{
		createMockInfobloxZone("example.com"),
		createMockInfobloxZone("other.com"),
	}
	client.(*mockIBConnector).mockInfobloxObjects = &[]ibclient.IBObject{
		createMockInfobloxObject("deleted.example.com", endpoint.RecordTypeA, "121.212.121.212"),
		createMockInfobloxObject("deletedcname.example.com", endpoint.RecordTypeCNAME, "other.com"),
		createMockInfobloxObject("old.example.com", endpoint.RecordTypeA, "121.212.121.212"),
		createMockInfobloxObject("oldcname.example.com", endpoint.RecordTypeCNAME, "other.com"),
	}

	provider := newInfobloxProvider(
		NewDomainFilter([]string{""}),
		dryRun,
		client,
	)

	createRecords := []*endpoint.Endpoint{
		endpoint.NewEndpoint("example.com", "1.2.3.4", endpoint.RecordTypeA),
		endpoint.NewEndpoint("example.com", "tag", endpoint.RecordTypeTXT),
		endpoint.NewEndpoint("foo.example.com", "1.2.3.4", endpoint.RecordTypeA),
		endpoint.NewEndpoint("foo.example.com", "tag", endpoint.RecordTypeTXT),
		endpoint.NewEndpoint("bar.example.com", "other.com", endpoint.RecordTypeCNAME),
		endpoint.NewEndpoint("bar.example.com", "tag", endpoint.RecordTypeTXT),
		endpoint.NewEndpoint("other.com", "5.6.7.8", endpoint.RecordTypeA),
		endpoint.NewEndpoint("other.com", "tag", endpoint.RecordTypeTXT),
		endpoint.NewEndpoint("nope.com", "4.4.4.4", endpoint.RecordTypeA),
		endpoint.NewEndpoint("nope.com", "tag", endpoint.RecordTypeTXT),
	}

	updateOldRecords := []*endpoint.Endpoint{
		endpoint.NewEndpoint("old.example.com", "121.212.121.212", endpoint.RecordTypeA),
		endpoint.NewEndpoint("oldcname.example.com", "other.com", endpoint.RecordTypeCNAME),
		endpoint.NewEndpoint("old.nope.com", "121.212.121.212", endpoint.RecordTypeA),
	}

	updateNewRecords := []*endpoint.Endpoint{
		endpoint.NewEndpoint("new.example.com", "111.222.111.222", endpoint.RecordTypeA),
		endpoint.NewEndpoint("newcname.example.com", "other.com", endpoint.RecordTypeCNAME),
		endpoint.NewEndpoint("new.nope.com", "222.111.222.111", endpoint.RecordTypeA),
	}

	deleteRecords := []*endpoint.Endpoint{
		endpoint.NewEndpoint("deleted.example.com", "121.212.121.212", endpoint.RecordTypeA),
		endpoint.NewEndpoint("deletedcname.example.com", "other.com", endpoint.RecordTypeCNAME),
		endpoint.NewEndpoint("deleted.nope.com", "222.111.222.111", endpoint.RecordTypeA),
	}

	changes := &plan.Changes{
		Create:    createRecords,
		UpdateNew: updateNewRecords,
		UpdateOld: updateOldRecords,
		Delete:    deleteRecords,
	}

	if err := provider.ApplyChanges(changes); err != nil {
		t.Fatal(err)
	}
}

func TestInfobloxZones(t *testing.T) {
	client := mockIBConnector{
		mockInfobloxZones: &[]ibclient.ZoneAuth{
			createMockInfobloxZone("example.com"),
			createMockInfobloxZone("lvl1-1.example.com"),
			createMockInfobloxZone("lvl2-1.lvl1-1.example.com"),
		},
		mockInfobloxObjects: &[]ibclient.IBObject{},
	}

	provider := newInfobloxProvider(NewDomainFilter([]string{"example.com"}), true, &client)
	zones, _ := provider.zones()

	assert.Equal(t, provider.findZone(zones, "example.com").Fqdn, "example.com")
	assert.Equal(t, provider.findZone(zones, "nginx.example.com").Fqdn, "example.com")
	assert.Equal(t, provider.findZone(zones, "lvl1-1.example.com").Fqdn, "lvl1-1.example.com")
	assert.Equal(t, provider.findZone(zones, "lvl1-2.example.com").Fqdn, "example.com")
	assert.Equal(t, provider.findZone(zones, "lvl2-1.lvl1-1.example.com").Fqdn, "lvl2-1.lvl1-1.example.com")
	assert.Equal(t, provider.findZone(zones, "lvl2-2.lvl1-1.example.com").Fqdn, "lvl1-1.example.com")
	assert.Equal(t, provider.findZone(zones, "lvl2-2.lvl1-2.example.com").Fqdn, "example.com")
}
