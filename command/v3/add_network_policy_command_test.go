package v3_test

import (
	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/actor/cfnetworkingaction"
	"code.cloudfoundry.org/cli/command/commandfakes"
	"code.cloudfoundry.org/cli/command/flag"
	"code.cloudfoundry.org/cli/command/translatableerror"
	. "code.cloudfoundry.org/cli/command/v3"
	"code.cloudfoundry.org/cli/command/v3/v3fakes"
	"code.cloudfoundry.org/cli/util/configv3"
	"code.cloudfoundry.org/cli/util/ui"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("add-network-policy Command", func() {
	var (
		cmd             AddNetworkPolicyCommand
		testUI          *ui.UI
		fakeConfig      *commandfakes.FakeConfig
		fakeSharedActor *commandfakes.FakeSharedActor
		fakeActor       *v3fakes.FakeAddNetworkPolicyActor
		binaryName      string
		executeErr      error
		srcApp          string
		destApp         string
		protocol        string
	)

	BeforeEach(func() {
		testUI = ui.NewTestUI(nil, NewBuffer(), NewBuffer())
		fakeConfig = new(commandfakes.FakeConfig)
		fakeSharedActor = new(commandfakes.FakeSharedActor)
		fakeActor = new(v3fakes.FakeAddNetworkPolicyActor)

		srcApp = "some-app"
		destApp = "some-other-app"
		protocol = "tcp"

		cmd = AddNetworkPolicyCommand{
			UI:             testUI,
			Config:         fakeConfig,
			SharedActor:    fakeSharedActor,
			Actor:          fakeActor,
			RequiredArgs:   flag.AddNetworkPolicyArgs{SourceName: srcApp},
			DestinationApp: destApp,
		}
		cmd.DestinationIPs = flag.NetworkIPRange{}

		binaryName = "faceman"
		fakeConfig.BinaryNameReturns(binaryName)
	})

	JustBeforeEach(func() {
		executeErr = cmd.Execute(nil)
	})

	Context("destination args", func() {
		Context("when both --destination-app and --destination-ips are specified", func() {
			BeforeEach(func() {
				cmd.DestinationApp = "myapp"
				cmd.DestinationIPs.UnmarshalFlag("1.2.3.4-1.2.4.5")
			})
			It("errors out with a message", func() {
				Expect(executeErr).To(MatchError("cannot specify both --destination-ips and --destination-app at the same time"))
			})
		})
		Context("when neither --destination-ips nor --destination-app is specified", func() {
			BeforeEach(func() {
				cmd.DestinationApp = ""
				cmd.DestinationIPs = flag.NetworkIPRange{}
			})

			It("errors out with a message", func() {
				Expect(executeErr).To(MatchError("must specify either --destination-ips or --destination-app"))
			})
		})
		Context("when --destination-app but no --destination-ips", func() {
			BeforeEach(func() {
				cmd.DestinationApp = "myapp"
			})
			It("creates a c2c policy", func() {
				Expect(executeErr).NotTo(HaveOccurred())
				Expect(fakeActor.AddNetworkPolicyCallCount()).To(Equal(1))
			})
		})
		Context("when --destination-ips but no --destination-app", func() {
			BeforeEach(func() {
				cmd.DestinationApp = ""
				cmd.DestinationIPs.UnmarshalFlag("1.2.3.4-1.2.4.5")
			})
			It("creates an egress policy", func() {
				Expect(executeErr).NotTo(HaveOccurred())
				Expect(fakeActor.AddNetworkPolicyCallCount()).To(Equal(1))
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

	Context("when the user is logged in, an org is targeted, and a space is targeted", func() {
		BeforeEach(func() {
			fakeConfig.CurrentUserReturns(configv3.User{Name: "some-user"}, nil)
			fakeConfig.TargetedSpaceReturns(configv3.Space{Name: "some-space", GUID: "some-space-guid"})
			fakeConfig.TargetedOrganizationReturns(configv3.Organization{Name: "some-org"})
		})

		Context("when protocol is specified but port is not", func() {
			BeforeEach(func() {
				cmd.Protocol = flag.NetworkProtocol{Protocol: protocol}
			})

			It("returns an error", func() {
				Expect(executeErr).To(MatchError(translatableerror.NetworkPolicyProtocolOrPortNotProvidedError{}))
				Expect(testUI.Out).NotTo(Say(`Adding network policy`))
			})
		})

		Context("when port is specified but protocol is not", func() {
			BeforeEach(func() {
				cmd.Port = flag.NetworkPort{StartPort: 8080, EndPort: 8081}
			})

			It("returns an error", func() {
				Expect(executeErr).To(MatchError(translatableerror.NetworkPolicyProtocolOrPortNotProvidedError{}))
				Expect(testUI.Out).NotTo(Say(`Adding network policy`))
			})
		})

		Context("when both protocol and port are specificed", func() {
			BeforeEach(func() {
				cmd.Protocol = flag.NetworkProtocol{Protocol: protocol}
				cmd.Port = flag.NetworkPort{StartPort: 8080, EndPort: 8081}
			})

			Context("when the policy creation is successful", func() {
				BeforeEach(func() {
					fakeActor.AddNetworkPolicyReturns(cfnetworkingaction.Warnings{"some-warning-1", "some-warning-2"}, nil)
				})

				It("displays OK when no error occurs", func() {
					Expect(executeErr).ToNot(HaveOccurred())
					Expect(fakeActor.AddNetworkPolicyCallCount()).To(Equal(1))
					passedSpaceGuid, passedSrcAppName, _, passedDestAppName, _, _, passedProtocol, passedStartPort, passedEndPort := fakeActor.AddNetworkPolicyArgsForCall(0)
					Expect(passedSpaceGuid).To(Equal("some-space-guid"))
					Expect(passedSrcAppName).To(Equal("some-app"))
					Expect(passedDestAppName).To(Equal("some-other-app"))
					Expect(passedProtocol).To(Equal("tcp"))
					Expect(passedStartPort).To(Equal(8080))
					Expect(passedEndPort).To(Equal(8081))

					Expect(testUI.Out).To(Say(`Adding network policy to app %s in org some-org / space some-space as some-user\.\.\.`, srcApp))
					Expect(testUI.Err).To(Say("some-warning-1"))
					Expect(testUI.Err).To(Say("some-warning-2"))
					Expect(testUI.Out).To(Say("OK"))
				})
			})

			Context("when the policy creation is not successful", func() {
				BeforeEach(func() {
					fakeActor.AddNetworkPolicyReturns(cfnetworkingaction.Warnings{"some-warning-1", "some-warning-2"}, actionerror.ApplicationNotFoundError{Name: srcApp})
				})

				It("does not display OK when an error occurs", func() {
					Expect(executeErr).To(MatchError(actionerror.ApplicationNotFoundError{Name: srcApp}))

					Expect(testUI.Out).To(Say(`Adding network policy to app %s in org some-org / space some-space as some-user\.\.\.`, srcApp))
					Expect(testUI.Err).To(Say("some-warning-1"))
					Expect(testUI.Err).To(Say("some-warning-2"))
					Expect(testUI.Out).ToNot(Say("OK"))
				})
			})
		})

		Context("when destination ips is specified", func() {
			BeforeEach(func() {
				fakeActor.AddNetworkPolicyReturns(cfnetworkingaction.Warnings{"some-warning-1", "some-warning-2"}, nil)

				cmd.DestinationIPs = flag.NetworkIPRange{
					Start: "1.2.3.4",
					End:   "1.2.3.5",
				}
				cmd.DestinationApp = ""
			})

			It("displays OK when no error occurs", func() {
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(fakeActor.AddNetworkPolicyCallCount()).To(Equal(1))
				passedSpaceGuid, passedSrcAppName, _, passedDestAppName, passedDestIPStart, passedDestIPEnd, passedProtocol, passedStartPort, passedEndPort := fakeActor.AddNetworkPolicyArgsForCall(0)
				Expect(passedSpaceGuid).To(Equal("some-space-guid"))
				Expect(passedSrcAppName).To(Equal("some-app"))
				Expect(passedDestAppName).To(Equal(""))
				Expect(passedProtocol).To(Equal("tcp"))
				Expect(passedStartPort).To(Equal(8080))
				Expect(passedEndPort).To(Equal(8080))
				Expect(passedDestIPStart).To(Equal("1.2.3.4"))
				Expect(passedDestIPEnd).To(Equal("1.2.3.5"))

				Expect(testUI.Out).To(Say(`Adding network policy to app %s in org some-org / space some-space as some-user\.\.\.`, srcApp))
				Expect(testUI.Err).To(Say("some-warning-1"))
				Expect(testUI.Err).To(Say("some-warning-2"))
				Expect(testUI.Out).To(Say("OK"))
			})
		})

		Context("when source type is specified", func() {
			BeforeEach(func() {
				fakeActor.AddNetworkPolicyReturns(cfnetworkingaction.Warnings{"some-warning-1", "some-warning-2"}, nil)

				cmd.DestinationIPs = flag.NetworkIPRange{
					Start: "1.2.3.4",
					End:   "1.2.3.5",
				}
				cmd.DestinationApp = ""
				cmd.SourceType = "space"
			})

			It("displays OK when no error occurs", func() {
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(fakeActor.AddNetworkPolicyCallCount()).To(Equal(1))
				passedSpaceGuid, passedSrcAppName, passedSrcType, passedDestAppName, passedDestIPStart, passedDestIPEnd, passedProtocol, passedStartPort, passedEndPort := fakeActor.AddNetworkPolicyArgsForCall(0)
				Expect(passedSpaceGuid).To(Equal("some-space-guid"))
				Expect(passedSrcAppName).To(Equal("some-app"))
				Expect(passedSrcType).To(Equal("space"))
				Expect(passedDestAppName).To(Equal(""))
				Expect(passedProtocol).To(Equal("tcp"))
				Expect(passedStartPort).To(Equal(8080))
				Expect(passedEndPort).To(Equal(8080))
				Expect(passedDestIPStart).To(Equal("1.2.3.4"))
				Expect(passedDestIPEnd).To(Equal("1.2.3.5"))

				Expect(testUI.Out).To(Say(`Adding network policy to space %s in org some-org / space some-space as some-user\.\.\.`, srcApp))
				Expect(testUI.Err).To(Say("some-warning-1"))
				Expect(testUI.Err).To(Say("some-warning-2"))
				Expect(testUI.Out).To(Say("OK"))
			})
		})

		Context("when both protocol and port are not specified", func() {
			It("defaults protocol to 'tcp' and port to '8080'", func() {
				Expect(executeErr).ToNot(HaveOccurred())

				Expect(fakeActor.AddNetworkPolicyCallCount()).To(Equal(1))
				_, _, _, _, _, _, passedProtocol, passedStartPort, passedEndPort := fakeActor.AddNetworkPolicyArgsForCall(0)
				Expect(passedProtocol).To(Equal("tcp"))
				Expect(passedStartPort).To(Equal(8080))
				Expect(passedEndPort).To(Equal(8080))
			})
		})
	})
})
