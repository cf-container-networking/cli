package isolated

import (
	"fmt"

	"code.cloudfoundry.org/cli/integration/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("tasks command", func() {
	Context("when --help flag is set", func() {
		It("Displays command usage to output", func() {
			session := helpers.CF("tasks", "--help")
			Eventually(session.Out).Should(Say("NAME:"))
			Eventually(session.Out).Should(Say("   tasks - List tasks of an app"))
			Eventually(session.Out).Should(Say("USAGE:"))
			Eventually(session.Out).Should(Say("   cf tasks APP_NAME"))
			Eventually(session.Out).Should(Say("SEE ALSO:"))
			Eventually(session.Out).Should(Say("   apps, logs, run-task, terminate-task"))
			Eventually(session).Should(Exit(0))
		})
	})
	Context("when the environment is not setup correctly", func() {
		Context("when no API endpoint is set", func() {
			BeforeEach(func() {
				helpers.UnsetAPI()
			})

			It("fails with no API endpoint set message", func() {
				session := helpers.CF("run-task", "app-name", "some command")
				Eventually(session.Out).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("No API endpoint set. Use 'cf login' or 'cf api' to target an endpoint."))
				Eventually(session).Should(Exit(1))
			})
		})

		Context("when not logged in", func() {
			BeforeEach(func() {
				helpers.LogoutCF()
			})

			It("fails with not logged in message", func() {
				session := helpers.CF("run-task", "app-name", "some command")
				Eventually(session.Out).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("Not logged in. Use 'cf login' to log in."))
				Eventually(session).Should(Exit(1))
			})
		})

		Context("when there is no org set", func() {
			BeforeEach(func() {
				helpers.LogoutCF()
				helpers.LoginCF()
			})

			It("fails with no targeted org error message", func() {
				session := helpers.CF("run-task", "app-name", "some command")
				Eventually(session.Out).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("No org targeted, use 'cf target -o ORG' to target an org."))
				Eventually(session).Should(Exit(1))
			})
		})

		Context("when there is no space set", func() {
			BeforeEach(func() {
				helpers.LogoutCF()
				helpers.LoginCF()
				helpers.TargetOrg(ReadOnlyOrg)
			})

			It("fails with no space targeted error message", func() {
				session := helpers.CF("run-task", "app-name", "some command")
				Eventually(session.Out).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("No space targeted, use 'cf target -s SPACE' to target a space"))
				Eventually(session).Should(Exit(1))
			})
		})
	})

	Context("when the environment is setup correctly", func() {
		var (
			orgName   string
			spaceName string
			appName   string
		)

		BeforeEach(func() {
			orgName = helpers.NewOrgName()
			spaceName = helpers.NewSpaceName()
			appName = helpers.PrefixedRandomName("APP")

			setupCF(orgName, spaceName)
		})

		Context("when the application does not exist", func() {
			It("fails and outputs an app not found message", func() {
				session := helpers.CF("run-task", appName, "echo hi")
				Eventually(session.Out).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say(fmt.Sprintf("App %s not found", appName)))
				Eventually(session).Should(Exit(1))
			})
		})

		Context("when the application exists", func() {
			BeforeEach(func() {
				helpers.WithHelloWorldApp(func(appDir string) {
					Eventually(helpers.CF("push", appName, "-p", appDir, "-b", "staticfile_buildpack")).Should(Exit(0))
				})
			})

			Context("when the application does not have associated tasks", func() {
				It("displays an empty table", func() {
					session := helpers.CF("tasks", appName)
					Eventually(session.Out).Should(Say(`
id   name   state   start time   command
`,
					))
					Consistently(session.Out).ShouldNot(Say("1"))
					Eventually(session).Should(Exit(0))
				})
			})

			Context("when the application has associated tasks", func() {
				BeforeEach(func() {
					Eventually(helpers.CF("run-task", appName, "echo hello world")).Should(Exit(0))
					Eventually(helpers.CF("run-task", appName, "echo foo bar")).Should(Exit(0))
				})

				It("displays all the tasks in descending order", func() {
					session := helpers.CF("tasks", appName)
					userName, _ := helpers.GetCredentials()
					Eventually(session.Out).Should(Say(fmt.Sprintf("Getting tasks for app %s in org %s / space %s as %s...", appName, orgName, spaceName, userName)))
					Eventually(session.Out).Should(Say("OK\n"))
					Eventually(session.Out).Should(Say(`id\s+name\s+state\s+start time\s+command
2\s+[a-zA-Z-0-9 ,:]+echo foo bar
1\s+[a-zA-Z-0-9 ,:]+echo hello world`))
					Eventually(session).Should(Exit(0))
				})

				Context("when the logged in user does not have authorization to see task commands", func() {
					var user string

					BeforeEach(func() {
						user = helpers.NewUsername()
						password := helpers.NewPassword()
						Eventually(helpers.CF("create-user", user, password)).Should(Exit(0))
						Eventually(helpers.CF("set-space-role", user, orgName, spaceName, "SpaceAuditor")).Should(Exit(0))
						Eventually(helpers.CF("auth", user, password)).Should(Exit(0))
						Eventually(helpers.CF("target", "-o", orgName, "-s", spaceName)).Should(Exit(0))
					})

					It("does not display task commands", func() {
						session := helpers.CF("tasks", appName)
						Eventually(session.Out).Should(Say("2\\s+[a-zA-Z-0-9 ,:]+\\[hidden\\]"))
						Eventually(session.Out).Should(Say("1\\s+[a-zA-Z-0-9 ,:]+\\[hidden\\]"))
						Eventually(session).Should(Exit(0))
					})
				})
			})
		})
	})
})
