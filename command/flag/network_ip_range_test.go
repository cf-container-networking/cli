package flag_test

import (
	"code.cloudfoundry.org/cli/command/flag"
	flags "github.com/jessevdk/go-flags"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetworkIPRange", func() {
	Describe("UnmarshalFlag", func() {
		var (
			ipRange flag.NetworkIPRange
		)

		BeforeEach(func() {
			ipRange = flag.NetworkIPRange{}
		})

		DescribeTable("it sets the ip range correctly",
			func(input string, expectedStart string, expectedEnd string) {
				err := ipRange.UnmarshalFlag(input)
				Expect(err).ToNot(HaveOccurred())
				Expect(ipRange).To(Equal(flag.NetworkIPRange{
					Start: expectedStart,
					End:   expectedEnd,
				}))
			},
			Entry("when provided '1.2.3.4' it sets the start and end to '1.2.3.4'", "1.2.3.4", "1.2.3.4", "1.2.3.4"),
			Entry("when provided '1.2.3.4-1.2.3.5' it sets the start to '1.2.3.4' and end to '1.2.3.5'", "1.2.3.4-1.2.3.5", "1.2.3.4", "1.2.3.5"),
		)
		DescribeTable("errors on bad input",
			func(input string, expectedError error) {
				err := ipRange.UnmarshalFlag(input)
				Expect(err).To(MatchError(expectedError))
			},
			Entry("when provided '1.2.3.4-1.2.3.5-1.2.3.6' it errors out",
				"1.2.3.4-1.2.3.5-1.2.3.6", &flags.Error{
					Type:    flags.ErrUnknown,
					Message: `invalid ip range format: 1.2.3.4-1.2.3.5-1.2.3.6`,
				}),
			Entry("when provided 'x.y.z.1-1.2.3.4' it errors out",
				"x.y.z.1", &flags.Error{
					Type:    flags.ErrUnknown,
					Message: `invalid ip address: x.y.z.1`,
				}),
			Entry("when provided '1.2.3.4-x.y.z.1' it errors out",
				"x.y.z.1", &flags.Error{
					Type:    flags.ErrUnknown,
					Message: `invalid ip address: x.y.z.1`,
				}),
		)
	})
})
