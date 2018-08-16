package v3_test

import (
	"errors"

	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/actor/cfnetworkingaction"
	"code.cloudfoundry.org/cli/actor/v3action"
	"code.cloudfoundry.org/cli/command/commandfakes"
	. "code.cloudfoundry.org/cli/command/v3"
	"code.cloudfoundry.org/cli/command/v3/v3fakes"
	"code.cloudfoundry.org/cli/util/configv3"
	"code.cloudfoundry.org/cli/util/ui"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("network-policies Command", func() {
	var (
		cmd             NetworkPoliciesCommand
		testUI          *ui.UI
		fakeConfig      *commandfakes.FakeConfig
		fakeSharedActor *commandfakes.FakeSharedActor
		fakeActor       *v3fakes.FakeNetworkPoliciesActor
		binaryName      string
		executeErr      error
		srcApp          string
	)

	BeforeEach(func() {
		testUI = ui.NewTestUI(nil, NewBuffer(), NewBuffer())
		fakeConfig = new(commandfakes.FakeConfig)
		fakeSharedActor = new(commandfakes.FakeSharedActor)
		fakeActor = new(v3fakes.FakeNetworkPoliciesActor)

		srcApp = ""

		cmd = NetworkPoliciesCommand{
			UI:          testUI,
			Source:      srcApp,
			Config:      fakeConfig,
			SharedActor: fakeSharedActor,
			Actor:       fakeActor,
		}

		binaryName = "faceman"
		fakeConfig.BinaryNameReturns(binaryName)
	})

	JustBeforeEach(func() {
		executeErr = cmd.Execute(nil)
	})

	Context("when passing source types", func() {
		BeforeEach(func() {
			fakeConfig.TargetedOrganizationReturns(configv3.Organization{
				Name: "some-org",
				GUID: "org-guid",
			})

			fakeConfig.TargetedSpaceReturns(configv3.Space{
				Name: "some-space",
				GUID: "space-guid",
			})
		})

		Context("when type is specified", func() {
			Context("when type and source are apps", func() {
				BeforeEach(func() {
					cmd.Type = "app"
					cmd.Source = "app-name"
				})

				It("allows for type app", func() {
					Expect(executeErr).NotTo(HaveOccurred())
					spaceGuid, source := fakeActor.NetworkPoliciesBySpaceAndAppNameArgsForCall(0)
					Expect(spaceGuid).To(Equal("space-guid"))
					Expect(source).To(Equal("app-name"))
				})
			})

			Context("when type and source are spaces", func() {
				BeforeEach(func() {
					fakeActor.GetSpaceByNameAndOrganizationReturns(
						v3action.Space{
							GUID: "space-guid",
						}, nil, nil,
					)
					cmd.Type = "space"
					cmd.Source = "space-name"
				})

				It("allows for type space", func() {
					Expect(executeErr).NotTo(HaveOccurred())
					spaceGuid := fakeActor.NetworkPoliciesBySpaceArgsForCall(0)
					Expect(spaceGuid).To(Equal("space-guid"))
				})

				Context("when getting the space returns an error", func() {
					BeforeEach(func() {
						fakeActor.GetSpaceByNameAndOrganizationReturns(
							v3action.Space{},
							nil,
							errors.New("whiskey"))
					})

					It("should return the error from the actor", func() {
						Expect(executeErr).To(MatchError("whiskey"))
					})
				})

				Context("when getting the space returns a warning", func() {
					BeforeEach(func() {
						fakeActor.GetSpaceByNameAndOrganizationReturns(
							v3action.Space{},
							cfnetworkingaction.Warnings{"i am a warning", "tango"},
							nil)

						fakeActor.NetworkPoliciesBySpaceReturns(
							nil,
							cfnetworkingaction.Warnings{"warning from network policies"},
							nil)
					})

					It("should display the warnings from the actor", func() {
						Expect(testUI.Err).To(Say("i am a warning"))
						Expect(testUI.Err).To(Say("tango"))
						Expect(testUI.Err).To(Say("warning from network policies"))
					})
				})
			})

			Context("when type is not app or space", func() {
				BeforeEach(func() {
					cmd.Type = "unsupported"
					cmd.Source = "app-name"
				})

				It("returns an unknown type error", func() {
					Expect(executeErr).To(MatchError("unknown source type: unsupported"))
				})
			})

			Context("when there is no source and type is defined", func() {
				BeforeEach(func() {
					cmd.Type = "space"
					cmd.Source = ""
				})

				It("returns a missing source error", func() {
					Expect(executeErr).To(MatchError("missing source argument"))
				})
			})
		})

		Context("when no type is specified", func() {
			Context("when source is specified", func() {
				BeforeEach(func() {
					cmd.Type = ""
					cmd.Source = "app-name"
				})

				It("queries for app source by default", func() {
					Expect(executeErr).NotTo(HaveOccurred())
					spaceGuid, source := fakeActor.NetworkPoliciesBySpaceAndAppNameArgsForCall(0)
					Expect(spaceGuid).To(Equal("space-guid"))
					Expect(source).To(Equal("app-name"))
				})
			})

			Context("when no source is specified", func() {
				BeforeEach(func() {
					cmd.Type = ""
					cmd.Source = ""
				})

				It("queries against targeted org and space policies", func() {
					Expect(executeErr).NotTo(HaveOccurred())
					spaceGuid := fakeActor.NetworkPoliciesBySpaceArgsForCall(0)
					Expect(spaceGuid).To(Equal("space-guid"))
				})
			})
		})

	})
	Context("when checking target fails", func() {
		BeforeEach(func() {
			fakeSharedActor.CheckTargetReturns(actionerror.NotLoggedInError{BinaryName: binaryName})
		})

		It("returns an error", func() {
			Expect(executeErr).To(MatchError(actionerror.NotLoggedInError{BinaryName: binaryName}))

			Expect(fakeSharedActor.CheckTargetCallCount()).To(Equal(1))
			checkTargetedOrg, checkTargetedSpace := fakeSharedActor.CheckTargetArgsForCall(0)
			Expect(checkTargetedOrg).To(BeTrue())
			Expect(checkTargetedSpace).To(BeTrue())
		})
	})

	Context("when the user is logged in", func() {
		BeforeEach(func() {
			fakeConfig.CurrentUserReturns(configv3.User{Name: "some-user"}, nil)
			fakeConfig.TargetedSpaceReturns(configv3.Space{Name: "some-space", GUID: "some-space-guid"})
			fakeConfig.TargetedOrganizationReturns(configv3.Organization{Name: "some-org"})
		})

		It("outputs flavor text", func() {
			Expect(testUI.Out).To(Say(`Listing network policies in org some-org / space some-space as some-user\.\.\.`))
		})

		Context("when fetching the user fails", func() {
			BeforeEach(func() {
				fakeConfig.CurrentUserReturns(configv3.User{}, errors.New("some-error"))
			})

			It("returns an error", func() {
				Expect(executeErr).To(MatchError("some-error"))
			})
		})

		Context("when listing policies is successful", func() {
			BeforeEach(func() {
				fakeActor.NetworkPoliciesBySpaceReturns([]cfnetworkingaction.Policy{
					{
						SourceName:      "app1",
						SourceType:      "app",
						DestinationName: "app2",
						DestinationType: "app",
						Protocol:        "tcp",
						StartPort:       8080,
						EndPort:         8080,
					}, {
						SourceName:      "app2",
						SourceType:      "app",
						DestinationName: "app1",
						DestinationType: "app",
						Protocol:        "udp",
						StartPort:       1234,
						EndPort:         2345,
					}, {
						SourceName:         "app3",
						SourceType:         "app",
						DestinationStartIP: "1.2.3.4",
						DestinationEndIP:   "1.2.3.5",
						DestinationType:    "ip",
						Protocol:           "tcp",
						StartPort:          1111,
						EndPort:            2222,
					}, {
						SourceName:         "app4",
						SourceType:         "app",
						DestinationStartIP: "1.2.3.4",
						DestinationEndIP:   "1.2.3.4",
						DestinationType:    "ip",
						Protocol:           "tcp",
						StartPort:          1111,
						EndPort:            2222,
					},
				}, cfnetworkingaction.Warnings{"some-warning-1", "some-warning-2"}, nil)
			})

			It("lists the policies when no error occurs", func() {
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(fakeActor.NetworkPoliciesBySpaceCallCount()).To(Equal(1))
				passedSpaceGuid := fakeActor.NetworkPoliciesBySpaceArgsForCall(0)
				Expect(passedSpaceGuid).To(Equal("some-space-guid"))

				Expect(testUI.Out).To(Say(`Listing network policies in org some-org / space some-space as some-user\.\.\.`))
				Expect(testUI.Out).To(Say("\n\n"))
				Expect(testUI.Out).To(Say("source\\s+source type\\s+destination\\s+destination type\\s+protocol\\s+ports"))
				Expect(testUI.Out).To(Say("app1\\s+app\\s+app2\\s+app\\s+tcp\\s+8080[^-]"))
				Expect(testUI.Out).To(Say("app2\\s+app\\s+app1\\s+app\\s+udp\\s+1234-2345"))
				Expect(testUI.Out).To(Say("app3\\s+app\\s+1.2.3.4-1.2.3.5\\s+ip\\s+tcp\\s+1111-2222"))
				Expect(testUI.Out).To(Say("app4\\s+app\\s+1.2.3.4\\s+ip\\s+tcp\\s+1111-2222"))

				Expect(testUI.Err).To(Say("some-warning-1"))
				Expect(testUI.Err).To(Say("some-warning-2"))
			})

			Context("when a source app name is passed", func() {
				BeforeEach(func() {
					cmd.Source = "some-app"
					fakeActor.NetworkPoliciesBySpaceAndAppNameReturns([]cfnetworkingaction.Policy{
						{
							SourceName:      "app1",
							DestinationName: "app2",
							Protocol:        "tcp",
							StartPort:       8080,
							EndPort:         8080,
						}, {
							SourceName:      "app2",
							DestinationName: "app1",
							Protocol:        "udp",
							StartPort:       1234,
							EndPort:         2345,
						},
					}, cfnetworkingaction.Warnings{"some-warning-1", "some-warning-2"}, nil)
				})

				It("lists the policies when no error occurs", func() {
					Expect(executeErr).ToNot(HaveOccurred())
					Expect(fakeActor.NetworkPoliciesBySpaceAndAppNameCallCount()).To(Equal(1))
					passedSpaceGuid, passedSrcAppName := fakeActor.NetworkPoliciesBySpaceAndAppNameArgsForCall(0)
					Expect(passedSpaceGuid).To(Equal("some-space-guid"))
					Expect(passedSrcAppName).To(Equal("some-app"))

					Expect(testUI.Out).To(Say(`Listing network policies of app %s in org some-org / space some-space as some-user\.\.\.`, cmd.Source))
					Expect(testUI.Out).To(Say("\n\n"))
					Expect(testUI.Out).To(Say("source\\s+source type\\s+destination\\s+destination type\\s+protocol\\s+ports"))
					Expect(testUI.Out).To(Say("app1\\s+app2\\s+tcp\\s+8080[^-]"))
					Expect(testUI.Out).To(Say("app2\\s+app1\\s+udp\\s+1234-2345"))

					Expect(testUI.Err).To(Say("some-warning-1"))
					Expect(testUI.Err).To(Say("some-warning-2"))
				})
			})
			Context("when ports are not specified", func() {
				BeforeEach(func() {
					cmd.Source = "some-app"
					fakeActor.NetworkPoliciesBySpaceAndAppNameReturns([]cfnetworkingaction.Policy{
						{
							SourceName:         "app1",
							SourceType:         "app",
							DestinationStartIP: "4.4.4.4",
							DestinationEndIP:   "5.5.5.5",
							DestinationType:    "ip",
							Protocol:           "tcp",
						},
					}, cfnetworkingaction.Warnings{"some-warning-1", "some-warning-2"}, nil)
				})
				It("lists the policies when no error occurs", func() {
					Expect(executeErr).ToNot(HaveOccurred())
					Expect(fakeActor.NetworkPoliciesBySpaceAndAppNameCallCount()).To(Equal(1))
					passedSpaceGuid, passedSrcAppName := fakeActor.NetworkPoliciesBySpaceAndAppNameArgsForCall(0)
					Expect(passedSpaceGuid).To(Equal("some-space-guid"))
					Expect(passedSrcAppName).To(Equal("some-app"))

					Expect(testUI.Out).To(Say(`Listing network policies of app %s in org some-org / space some-space as some-user\.\.\.`, cmd.Source))
					Expect(testUI.Out).To(Say("\n\n"))
					Expect(testUI.Out).To(Say("source\\s+source type\\s+destination\\s+destination type\\s+protocol\\s+ports"))
					Expect(testUI.Out).To(Say("app1\\s+app\\s+4.4.4.4-5.5.5.5\\s+ip\\s+tcp\\s+$"))

					Expect(testUI.Err).To(Say("some-warning-1"))
					Expect(testUI.Err).To(Say("some-warning-2"))
				})
			})
		})

		Context("when listing the policies is not successful", func() {
			BeforeEach(func() {
				fakeActor.NetworkPoliciesBySpaceReturns([]cfnetworkingaction.Policy{}, cfnetworkingaction.Warnings{"some-warning-1", "some-warning-2"}, actionerror.ApplicationNotFoundError{Name: srcApp})
			})

			It("displays warnings and returns the error", func() {
				Expect(executeErr).To(MatchError(actionerror.ApplicationNotFoundError{Name: srcApp}))

				Expect(testUI.Out).To(Say(`Listing network policies in org some-org / space some-space as some-user\.\.\.`))
				Expect(testUI.Err).To(Say("some-warning-1"))
				Expect(testUI.Err).To(Say("some-warning-2"))
			})
		})
	})
})
