package main

import (
	"encoding/json"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/admission/v1beta1"
)

var _ = Describe("Event", func() {
	var (
		ar             v1beta1.AdmissionReview
		arDockerCreate v1beta1.AdmissionReview
	)
	dat, err := ioutil.ReadFile("test_resources/admission_request.json")
	dockerCreateAr, err := ioutil.ReadFile("test_resources/admission_request_docker_create.json")
	if err != nil {
		panic(err)
	}

	BeforeEach(func() {
		err = json.Unmarshal(dat, &ar)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(dockerCreateAr, &arDockerCreate)
		if err != nil {
			panic(err)
		}
	})

	Describe("NewEvent", func() {
		It("Should create a new Event object", func() {
			evt, err := NewEvent(&ar)
			Expect(evt).ToNot(Equal(nil), "Should return an event object")
			Expect(evt.crd.Status.lastOperation).To(Equal(GenericLastOperation{
				Type:  "create",
				State: "succeeded",
			}), "Should return an event object with valid LastOperation")
			Expect(err).To(BeNil())
		})
		It("Should throw error if object cannot be parsed", func() {
			temp := ar.Request.Object.Raw
			ar.Request.Object.Raw = []byte("")
			evt, err := NewEvent(&ar)
			Expect(evt).To(BeNil())
			Expect(err).ToNot(BeNil())
			ar.Request.Object.Raw = temp
		})
		It("Should set oldCrd empty if old object cannot be parsed", func() {
			ar.Request.OldObject.Raw = []byte("")
			evt, err := NewEvent(&ar)
			Expect(evt.oldCrd).To(Equal(GenericResource{}))
			Expect(err).To(BeNil())
		})
	})
	Describe("isMeteringEvent", func() {
		Context("When Type is Update and kind is Director", func() {
			It("Should should return true if update with plan change succeeds", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "update"
				evt.crd.Status.lastOperation.State = "succeeded"
				evt.crd.Status.State = "succeeded"
				evt.oldCrd.Status.lastOperation.Type = "update"
				evt.oldCrd.Status.lastOperation.State = "in_progress"
				evt.oldCrd.Status.State = "in_progress"
				evt.crd.Status.appliedOptions.PlanID = "newPlanUUID"
				evt.oldCrd.Status.appliedOptions.PlanID = "oldPlanUUID"
				Expect(evt.isMeteringEvent()).To(Equal(true))
			})
			It("Should should return flase if update with no plan change succeeds", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "update"
				evt.crd.Status.lastOperation.State = "succeeded"
				evt.crd.Status.State = "succeeded"
				evt.oldCrd.Status.lastOperation.Type = "update"
				evt.oldCrd.Status.lastOperation.State = "in_progress"
				evt.oldCrd.Status.State = "in_progress"
				evt.crd.Status.appliedOptions.PlanID = "PlanUUID"
				evt.oldCrd.Status.appliedOptions.PlanID = "PlanUUID"
				Expect(evt.isMeteringEvent()).To(Equal(false))
			})
			It("Should should return flase if state does not change", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "update"
				evt.crd.Status.lastOperation.State = "succeeded"
				evt.crd.Status.State = "succeeded"
				evt.oldCrd.Status.lastOperation.Type = "update"
				evt.oldCrd.Status.lastOperation.State = "succeeded"
				evt.oldCrd.Status.State = "succeeded"
				evt.crd.Status.appliedOptions.PlanID = "newPlanUUID"
				evt.oldCrd.Status.appliedOptions.PlanID = "oldPlanUUID"
				Expect(evt.isMeteringEvent()).To(Equal(false))
			})
			It("Should should return false if update fails", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "update"
				evt.crd.Status.lastOperation.State = "failed"
				evt.crd.Status.State = "failed"
				evt.oldCrd.Status.lastOperation.Type = "update"
				evt.oldCrd.Status.lastOperation.State = "in_progress"
				evt.oldCrd.Status.State = "in_progress"
				evt.crd.Status.appliedOptions.PlanID = "newPlanUUID"
				evt.oldCrd.Status.appliedOptions.PlanID = "oldPlanUUID"
				Expect(evt.isMeteringEvent()).To(Equal(false))
			})
		})
		Context("When Type is Create and kind is Director", func() {
			It("Should should return true if create succeeds", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "create"
				evt.crd.Status.lastOperation.State = "succeeded"
				evt.crd.Status.State = "succeeded"
				evt.oldCrd.Status.lastOperation.Type = "create"
				evt.oldCrd.Status.lastOperation.State = "in_progress"
				evt.oldCrd.Status.State = "in_progress"
				evt.crd.Status.appliedOptions.PlanID = "PlanUUID"
				evt.oldCrd.Status.appliedOptions.PlanID = "PlanUUID"
				Expect(evt.isMeteringEvent()).To(Equal(true))
			})
			It("Should should return false if create state change does not change", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "create"
				evt.crd.Status.lastOperation.State = "succeeded"
				evt.crd.Status.State = "succeeded"
				evt.oldCrd.Status.lastOperation.Type = "create"
				evt.oldCrd.Status.lastOperation.State = "succeeded"
				evt.oldCrd.Status.State = "succeeded"
				evt.crd.Status.appliedOptions.PlanID = "newPlanUUID"
				evt.oldCrd.Status.appliedOptions.PlanID = "oldPlanUUID"
				Expect(evt.isMeteringEvent()).To(Equal(false))
			})
			It("Should should return false if create fails", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "create"
				evt.crd.Status.State = "failed"
				evt.oldCrd.Status.lastOperation.Type = "create"
				evt.oldCrd.Status.State = "in_progress"
				Expect(evt.isMeteringEvent()).To(Equal(false))
			})
		})
		Context("When Type is Create and kind is Docker", func() {
			It("Should should return true if create succeeds", func() {
				evt, _ := NewEvent(&arDockerCreate)
				evt.crd.Status.State = "succeeded"
				evt.oldCrd.Status.State = "in_progress"
				Expect(evt.isMeteringEvent()).To(Equal(true))
			})
			It("Should should return false if create state change does not change", func() {
				evt, _ := NewEvent(&arDockerCreate)
				evt.crd.Status.State = "succeeded"
				evt.oldCrd.Status.State = "succeeded"
				Expect(evt.isMeteringEvent()).To(Equal(false))
			})
			It("Should should return false if create fails", func() {
				evt, _ := NewEvent(&arDockerCreate)
				evt.crd.Status.State = "failed"
				evt.oldCrd.Status.State = "in_progress"
				Expect(evt.isMeteringEvent()).To(Equal(false))
			})
		})
		Context("When Type is Delete and kind is Director", func() {
			It("Should should return true if delete is triggered", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.State = "delete"
				evt.oldCrd.Status.State = "succeeded"
				Expect(evt.isMeteringEvent()).To(Equal(true))
			})
			It("Should should return false when delete state change does not change", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.State = "delete"
				evt.crd.Status.lastOperation.Type = "delete"
				evt.oldCrd.Status.State = "delete"
				evt.oldCrd.Status.lastOperation.Type = "delete"
				Expect(evt.isMeteringEvent()).To(Equal(false))
			})
			It("Should should return false if create fails", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "delete"
				evt.crd.Status.State = "failed"
				evt.oldCrd.Status.State = "delete"
				evt.oldCrd.Status.lastOperation.Type = "delete"
				Expect(evt.isMeteringEvent()).To(Equal(false))
			})
		})
		Context("When Type is Delete and kind is Docker", func() {
			It("Should should return true if delete is triggered", func() {
				evt, _ := NewEvent(&arDockerCreate)
				evt.crd.Status.State = "delete"
				evt.oldCrd.Status.State = "succeeded"
				Expect(evt.isMeteringEvent()).To(Equal(true))
			})
			It("Should should return false when delete state change does not change", func() {
				evt, _ := NewEvent(&arDockerCreate)
				evt.crd.Status.State = "delete"
				evt.oldCrd.Status.State = "delete"
				Expect(evt.isMeteringEvent()).To(Equal(false))
			})
			It("Should should return false if create fails", func() {
				evt, _ := NewEvent(&arDockerCreate)
				evt.crd.Status.lastOperation.Type = "delete"
				evt.crd.Status.State = "failed"
				evt.oldCrd.Status.State = "delete"
				evt.oldCrd.Status.lastOperation.Type = "delete"
				Expect(evt.isMeteringEvent()).To(Equal(false))
			})
		})
	})

	Describe("ObjectToMapInterface", func() {
		It("Should convert object to map", func() {
			expected := make(map[string]interface{})
			expected["options"] = "dummyOptions"
			Expect(ObjectToMapInterface(GenericSpec{
				Options: "dummyOptions",
			})).To(Equal(expected))
		})
	})

	Describe("meteringToUnstructured", func() {
		It("Creates unstructured metering instance", func() {
			m := Metering{
				Spec: MeteringSpec{
					Options: MeteringOptions{},
				},
			}
			val, err := meteringToUnstructured(&m)
			Expect(err).To(BeNil())
			Expect(val).ToNot(BeNil())
			Expect(val.GetKind()).To(Equal("Sfevent"))
			Expect(val.GetAPIVersion()).To(Equal("instance.servicefabrik.io/v1alpha1"))
			Expect(val.GetLabels()[MeterStateKey]).To(Equal(ToBeMetered))

		})
	})

	Describe("getMeteringEvents", func() {
		Context("when type is update", func() {
			It("Generates two metering docs", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "update"

				evt.crd.Spec.options.PlanID = "new plan in options"
				evt.crd.Status.appliedOptions.PlanID = "newPlan"
				evt.oldCrd.Status.appliedOptions.PlanID = "oldPlan"

				docs, err := evt.getMeteringEvents()
				Expect(err).To(BeNil())
				Expect(len(docs)).To(Equal(2))
				var docStart MeteringOptions
				var docStop MeteringOptions
				docStart = docs[0].Spec.Options
				docStop = docs[1].Spec.Options
				Expect(docStart.ServiceInfo.Plan).To(Equal("new plan in options"))
				Expect(docStart.InstancesMeasures[0].Value).To(Equal(MeterStart))
				Expect(docStop.ServiceInfo.Plan).To(Equal("oldPlan"))
				Expect(docStop.InstancesMeasures[0].Value).To(Equal(MeterStop))
			})
		})
		Context("when type is create", func() {
			It("Generates one metering doc", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "create"
				docs, err := evt.getMeteringEvents()
				Expect(err).To(BeNil())
				Expect(len(docs)).To(Equal(1))
			})
		})
	})
})
