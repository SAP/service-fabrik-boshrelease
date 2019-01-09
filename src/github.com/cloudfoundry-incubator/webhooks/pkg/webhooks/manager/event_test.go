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
		ar v1beta1.AdmissionReview
	)
	dat, err := ioutil.ReadFile("test_resources/admission_request.json")
	if err != nil {
		panic(err)
	}

	BeforeEach(func() {
		err = json.Unmarshal(dat, &ar)
		if err != nil {
			panic(err)
		}
	})

	Describe("NewEvent", func() {
		It("Should create a new Event object", func() {
			evt, err := NewEvent(&ar)
			Expect(evt).ToNot(Equal(nil))
			Expect(evt.crd.Status.lastOperation).To(Equal(GenericLastOperation{
				Type:  "create",
				State: "succeeded",
			}))
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
		It("Should throw error if old object cannot be parsed", func() {
			ar.Request.OldObject.Raw = []byte("")
			evt, err := NewEvent(&ar)
			Expect(evt).To(BeNil())
			Expect(err).ToNot(BeNil())
		})
	})
	Describe("isMeteringEvent", func() {
		Context("When Type is Update", func() {
			It("Should should return true if update with plan change succeeds", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "update"
				evt.crd.Status.lastOperation.State = "succeeded"
				evt.oldCrd.Status.lastOperation.Type = "update"
				evt.oldCrd.Status.lastOperation.State = "in_progress"
				evt.crd.Status.appliedOptions.PlanID = "newPlanUUID"
				evt.oldCrd.Status.appliedOptions.PlanID = "oldPlanUUID"
				Expect(evt.isMeteringEvent()).To(Equal(true))
			})
			It("Should should return flase if update with no plan change succeeds", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "update"
				evt.crd.Status.lastOperation.State = "succeeded"
				evt.oldCrd.Status.lastOperation.Type = "update"
				evt.oldCrd.Status.lastOperation.State = "in_progress"
				evt.crd.Status.appliedOptions.PlanID = "PlanUUID"
				evt.oldCrd.Status.appliedOptions.PlanID = "PlanUUID"
				Expect(evt.isMeteringEvent()).To(Equal(false))
			})
			It("Should should return flase if state does not change", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "update"
				evt.crd.Status.lastOperation.State = "succeeded"
				evt.oldCrd.Status.lastOperation.Type = "update"
				evt.oldCrd.Status.lastOperation.State = "succeeded"
				evt.crd.Status.appliedOptions.PlanID = "newPlanUUID"
				evt.oldCrd.Status.appliedOptions.PlanID = "oldPlanUUID"
				Expect(evt.isMeteringEvent()).To(Equal(false))
			})
			It("Should should return false if update fails", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "update"
				evt.crd.Status.lastOperation.State = "failed"
				evt.oldCrd.Status.lastOperation.Type = "update"
				evt.oldCrd.Status.lastOperation.State = "in_progress"
				evt.crd.Status.appliedOptions.PlanID = "newPlanUUID"
				evt.oldCrd.Status.appliedOptions.PlanID = "oldPlanUUID"
				Expect(evt.isMeteringEvent()).To(Equal(false))
			})
		})
		Context("When Type is Create", func() {
			It("Should should return true if create succeeds", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "create"
				evt.crd.Status.lastOperation.State = "succeeded"
				evt.oldCrd.Status.lastOperation.Type = "create"
				evt.oldCrd.Status.lastOperation.State = "in_progress"
				evt.crd.Status.appliedOptions.PlanID = "PlanUUID"
				evt.oldCrd.Status.appliedOptions.PlanID = "PlanUUID"
				Expect(evt.isMeteringEvent()).To(Equal(true))
			})
			It("Should should return false if create state change does not change", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "create"
				evt.crd.Status.lastOperation.State = "succeeded"
				evt.oldCrd.Status.lastOperation.Type = "create"
				evt.oldCrd.Status.lastOperation.State = "succeeded"
				evt.crd.Status.appliedOptions.PlanID = "newPlanUUID"
				evt.oldCrd.Status.appliedOptions.PlanID = "oldPlanUUID"
				Expect(evt.isMeteringEvent()).To(Equal(false))
			})
			It("Should should return false if create fails", func() {
				evt, _ := NewEvent(&ar)
				evt.crd.Status.lastOperation.Type = "create"
				evt.crd.Status.lastOperation.State = "failed"
				evt.oldCrd.Status.lastOperation.Type = "create"
				evt.oldCrd.Status.lastOperation.State = "in_progress"
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
				Expect(docs[0].Spec.Options.PlanID).To(Equal("new plan in options"))
				Expect(docs[0].Spec.Options.Signal).To(Equal("start"))
				Expect(docs[1].Spec.Options.PlanID).To(Equal("oldPlan"))
				Expect(docs[1].Spec.Options.Signal).To(Equal("stop"))
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
