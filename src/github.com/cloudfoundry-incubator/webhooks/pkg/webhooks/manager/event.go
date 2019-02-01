package main

import (
	"context"
	"encoding/json"
	"errors"
	c "github.com/cloudfoundry-incubator/webhooks/pkg/webhooks/manager/constants"
	"github.com/cloudfoundry-incubator/webhooks/pkg/webhooks/manager/resources"

	"k8s.io/client-go/rest"

	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// EventType denotes the types of metering events
type EventType string

const (
	//UpdateEvent signals the update of an instance
	UpdateEvent EventType = "update"
	//CreateEvent signals the create of an instance
	CreateEvent EventType = "create"
	//DeleteEvent signals the delete of an instance
	DeleteEvent EventType = "delete"
	//InvalidEvent is not yet supported
	InvalidEvent EventType = "default"
)

//LastOperationType
const (
	loUpdate string = "update"
	loCreate string = "create"
)

//State
const (
	Succeeded string = "succeeded"
	Delete    string = "delete"
)

// CrdKind
const (
	Director string = "Director"
	Docker   string = "Docker"
	Sfevent  string = "Sfevent"
)

// Event stores the event details
type Event struct {
	AdmissionReview *v1beta1.AdmissionReview
	crd             resources.GenericResource
	oldCrd          resources.GenericResource
}

// NewEvent is a constructor for Event
func NewEvent(ar *v1beta1.AdmissionReview) (*Event, error) {
	arjson, err := json.Marshal(ar)
	req := ar.Request
	glog.Infof(`
    Creating event for
	%v
	Namespace=%v
	Request Name=%v
	UID=%v
	patchOperation=%v
	UserInfo=%v`,
		req.Kind,
		req.Namespace,
		req.Name,
		req.UID,
		req.Operation,
		req.UserInfo)
	crd, err := resources.GetGenericResource(ar.Request.Object.Raw)
	glog.Infof("Resource name : %v", crd.Name)
	if err != nil {
		glog.Errorf("Admission review JSON: %v", string(arjson))
		glog.Errorf("Could not get the GenericResource object %v", err)
		return nil, err
	}
	crd.Status.LastOperationObj = resources.GetLastOperation(crd)
	crd.Status.AppliedOptionsObj = resources.GetAppliedOptions(crd)

	var oldCrd resources.GenericResource
	if len(ar.Request.OldObject.Raw) != 0 {
		oldCrd, err = resources.GetGenericResource(ar.Request.OldObject.Raw)
		if err != nil {
			glog.Errorf("Admission review JSON: %v", string(arjson))
			glog.Errorf("Could not get the old GenericResource object %v", err)
			return nil, err
		}
		oldCrd.Status.LastOperationObj = resources.GetLastOperation(oldCrd)
		oldCrd.Status.AppliedOptionsObj = resources.GetAppliedOptions(oldCrd)
	} else {
		oldCrd = resources.GenericResource{}
	}

	return &Event{
		AdmissionReview: ar,
		crd:             crd,
		oldCrd:          oldCrd,
	}, nil
}

func (e *Event) isStateChanged() bool {
	glog.Infof("Checking state change new state: %s\n", e.crd.Status.State)
	glog.Infof("Checking state change old state: %s\n", e.oldCrd.Status.State)
	return e.crd.Status.State != e.oldCrd.Status.State
}

func (e *Event) isDeleteTriggered() bool {
	return e.crd.Status.State == Delete
}

func (e *Event) isPlanChanged() bool {
	appliedOptionsNew := e.crd.Status.AppliedOptionsObj
	appliedOptionsOld := e.oldCrd.Status.AppliedOptionsObj
	return appliedOptionsNew.PlanID != appliedOptionsOld.PlanID
}

func (e *Event) isCreate() bool {
	return e.crd.Status.LastOperationObj.Type == loCreate
}

func (e *Event) isUpdate() bool {
	return e.crd.Status.LastOperationObj.Type == loUpdate
}

func (e *Event) isSucceeded() bool {
	return e.crd.Status.State == Succeeded
}

func (e *Event) isDirector() bool {
	return e.crd.Kind == Director
}

func (e *Event) isDocker() bool {
	return e.crd.Kind == Docker
}

func (e *Event) isMeteringEvent() bool {
	// An event is metering event if
	// Create succeeded
	// or Update Succeeded
	// or Delete Triggered
	if e.isDirector() && e.isStateChanged() {
		if e.isSucceeded() {
			return (e.isUpdate() && e.isPlanChanged()) || e.isCreate()
		}
		return e.isDeleteTriggered()
	}
	return e.isDocker() && e.isStateChanged() && (e.isSucceeded() || e.isDeleteTriggered())
}

// ObjectToMapInterface converts an Object to map[string]interface{}
func ObjectToMapInterface(obj interface{}) (map[string]interface{}, error) {
	values := make(map[string]interface{})
	options, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(options, &values)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func getClient(cfg *rest.Config) (client.Client, error) {
	glog.Infof("Get client for Apiserver")
	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		glog.Errorf("unable to set up overall controller manager %v", err)
		return nil, err
	}
	options := client.Options{
		Scheme: mgr.GetScheme(),
		Mapper: mgr.GetRESTMapper(),
	}
	apiserver, err := client.New(cfg, options)
	if err != nil {
		glog.Errorf("Unable to create kubernetes client %v", err)
		return nil, err
	}
	return apiserver, err
}

func meteringToUnstructured(m *Metering) (*unstructured.Unstructured, error) {
	values, err := ObjectToMapInterface(m)
	if err != nil {
		glog.Errorf("unable convert to map interface %v", err)
		return nil, err
	}
	meteringDoc := &unstructured.Unstructured{}
	meteringDoc.SetUnstructuredContent(values)
	meteringDoc.SetKind(Sfevent)
	meteringDoc.SetAPIVersion(c.InstanceAPIVersion)
	meteringDoc.SetNamespace(c.DefaultNamespace)
	meteringDoc.SetName(m.getName())
	labels := make(map[string]string)
	labels[c.MeterStateKey] = c.ToBeMetered
	meteringDoc.SetLabels(labels)
	return meteringDoc, nil
}

func (e *Event) getMeteringEvent(opt resources.GenericOptions, signal int) *Metering {
	return newMetering(opt, e.crd, signal)
}

func (e *Event) getEventType() (EventType, error) {
	lo := e.crd.Status.LastOperationObj
	eventType := InvalidEvent
	if e.crd.Status.State == Delete {
		eventType = DeleteEvent
	} else if e.isDirector() {
		switch lo.Type {
		case loUpdate:
			eventType = UpdateEvent
		case loCreate:
			eventType = CreateEvent
		}
	} else if e.isDocker() && e.crd.Status.State == Succeeded {
		eventType = CreateEvent
	}
	if eventType == InvalidEvent {
		return eventType, errors.New("No supported event found")
	}
	return eventType, nil
}

func (e *Event) getMeteringEvents() ([]*Metering, error) {
	options, _ := e.crd.Spec.GetOptions()
	oldAppliedOptions := e.oldCrd.Status.AppliedOptionsObj
	var meteringDocs []*Metering

	et, err := e.getEventType()
	if err != nil {
		return nil, err
	}
	switch et {
	case UpdateEvent:
		meteringDocs = append(meteringDocs, e.getMeteringEvent(options, c.MeterStart))
		meteringDocs = append(meteringDocs, e.getMeteringEvent(oldAppliedOptions, c.MeterStop))
	case CreateEvent:
		meteringDocs = append(meteringDocs, e.getMeteringEvent(options, c.MeterStart))
	case DeleteEvent:
		meteringDocs = append(meteringDocs, e.getMeteringEvent(oldAppliedOptions, c.MeterStop))
	}
	return meteringDocs, nil
}

func (e *Event) createMertering(cfg *rest.Config) error {
	apiserver, err := getClient(cfg)
	if err != nil {
		return err
	}
	events, err := e.getMeteringEvents()
	if err != nil {
		return err
	}
	for _, evt := range events {
		unstructuredDoc, err := meteringToUnstructured(evt)
		if err != nil {
			glog.Errorf("Error converting event : %v", err)
			return err
		}
		err = apiserver.Create(context.TODO(), unstructuredDoc)
		if err != nil {
			glog.Errorf("Error creating: %v", err)
			return err
		}
		glog.Infof("Successfully created metering resource")
	}
	return nil
}
