package global

import (
	"code.cloudfoundry.org/cli/integration/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("rename buildpack command", func() {
	Describe("help", func() {
		Context("when --help flag is set", func() {
			It("Displays command usage to output", func() {
				session := helpers.CF("rename-buildpack", "--help")
				Eventually(session).Should(Say("NAME:"))
				Eventually(session).Should(Say("rename-buildpack - Rename a buildpack"))
				Eventually(session).Should(Say("USAGE:"))
				Eventually(session).Should(Say("cf rename-buildpack BUILDPACK_NAME NEW_BUILDPACK_NAME"))
				Eventually(session).Should(Say("SEE ALSO:"))
				Eventually(session).Should(Say("update-buildpack"))
				Eventually(session).Should(Exit(0))
			})
		})
	})

	Context("when the environment is not setup correctly", func() {
		It("fails with the appropriate errors", func() {
			helpers.CheckEnvironmentTargetedCorrectly(false, false, ReadOnlyOrg, "rename-buildpack", "fake-buildpack", "some-name")
		})
	})

	Context("when the user is logged in", func() {
		var (
			oldBuildpackName string
			newBuildpackName string
			stacks           []string
			username         string
		)

		BeforeEach(func() {
			helpers.LoginCF()
			oldBuildpackName = helpers.NewBuildpack()
			newBuildpackName = helpers.NewBuildpack()
			stacks = helpers.FetchStacks()

			username, _ = helpers.GetCredentials()
		})

		Context("when the user provides a stack", func() {
			var session *Session

			JustBeforeEach(func() {
				session = helpers.CF("rename-buildpack", oldBuildpackName, newBuildpackName, "-s", stacks[0])
			})

			Context("when no buildpack with the name/stack combo is found", func() {
				Context("when no buildpacks with the same name exist", func() {
					It("returns a buildpack not found error", func() {
						Eventually(session).Should(Say("Renaming buildpack %s to %s with stack %s as %s...", oldBuildpackName, newBuildpackName, stacks[0], username))
						Eventually(session).Should(Say("FAILED"))
						Eventually(session.Err).Should(Say("Buildpack %s with stack %s not found", oldBuildpackName, stacks[0]))
						Eventually(session).Should(Exit(1))
					})
				})

				Context("when no buildpacks with the same name and stack exist", func() {
					BeforeEach(func() {
						helpers.BuildpackWithStack(func(buildpackPath string) {
							session = helpers.CF("create-buildpack", oldBuildpackName, buildpackPath, "10")
							Eventually(session).Should(Exit(0))
						}, "")
					})

					It("returns a buildpack not found error", func() {
						Eventually(session).Should(Say("Renaming buildpack %s to %s with stack %s as %s...", oldBuildpackName, newBuildpackName, stacks[0], username))
						Eventually(session).Should(Say("FAILED"))
						Eventually(session.Err).Should(Say("Buildpack %s with stack %s not found", oldBuildpackName, stacks[0]))
						Eventually(session).Should(Exit(1))
					})
				})
			})

			Context("when there are multiple existing buildpacks with the specified old name", func() {
				Context("when one of the existing buildpacks has an empty stack", func() {
					BeforeEach(func() {
						helpers.BuildpackWithStack(func(buildpackPath string) {
							session = helpers.CF("create-buildpack", oldBuildpackName, buildpackPath, "10")
							Eventually(session).Should(Exit(0))
						}, stacks[0])

						helpers.BuildpackWithStack(func(buildpackPath string) {
							session = helpers.CF("create-buildpack", oldBuildpackName, buildpackPath, "10")
							Eventually(session).Should(Exit(0))
						}, "")
					})

					Context("when renaming to unique name", func() {
						It("successfully renames the buildpack", func() {
							Eventually(session).Should(Say("Renaming buildpack %s to %s with stack %s as %s...", oldBuildpackName, newBuildpackName, stacks[0], username))
							Eventually(session).Should(Say("OK"))
							Eventually(session).Should(Exit(0))
						})
					})

					Context("when renaming to the same name as another buildpack", func() {
						Context("when the existing existing buildpack with the new name has the same stack", func() {
							BeforeEach(func() {
								helpers.BuildpackWithStack(func(buildpackPath string) {
									session = helpers.CF("create-buildpack", newBuildpackName, buildpackPath, "10")
									Eventually(session).Should(Exit(0))
								}, stacks[0])
							})
							It("returns an error", func() {
								Eventually(session).Should(Say("Renaming buildpack %s to %s with stack %s as %s...", oldBuildpackName, newBuildpackName, stacks[0], username))
								Eventually(session).Should(Say("FAILED"))
								Eventually(session.Err).Should(Say("The buildpack name %s is already in use with stack %s", newBuildpackName, stacks[0]))
								Eventually(session).Should(Exit(1))
							})
						})

						Context("when the existing buildpack with the new name has a different stack", func() {
							BeforeEach(func() {
								helpers.SkipIfOneStack()
								helpers.BuildpackWithStack(func(buildpackPath string) {
									session = helpers.CF("create-buildpack", newBuildpackName, buildpackPath, "10")
									Eventually(session).Should(Exit(0))
								}, stacks[1])
							})

							It("successfully renames the buildpack", func() {
								Eventually(session).Should(Say("Renaming buildpack %s to %s with stack %s as %s...", oldBuildpackName, newBuildpackName, stacks[0], username))
								Eventually(session).Should(Say("OK"))
								Eventually(session).Should(Exit(0))
							})
						})

						Context("when the existing existing buildpack with the new name has an empty stack", func() {
							BeforeEach(func() {
								helpers.SkipIfOneStack()
								helpers.BuildpackWithStack(func(buildpackPath string) {
									session = helpers.CF("create-buildpack", newBuildpackName, buildpackPath, "10")
									Eventually(session).Should(Exit(0))
								}, "")

								It("successfully renames the buildpack", func() {
									Eventually(session).Should(Say("Renaming buildpack %s to %s with stack %s as %s...", oldBuildpackName, newBuildpackName, stacks[0], username))
									Eventually(session).Should(Say("OK"))
									Eventually(session).Should(Exit(0))
								})
							})
						})
					})
				})

				Context("when neither of the existing buildpacks has an empty stack", func() {
					BeforeEach(func() {
						helpers.SkipIfOneStack()

						helpers.BuildpackWithStack(func(buildpackPath string) {
							session = helpers.CF("create-buildpack", oldBuildpackName, buildpackPath, "10")
							Eventually(session).Should(Exit(0))
						}, stacks[0])
						helpers.BuildpackWithStack(func(buildpackPath string) {
							session = helpers.CF("create-buildpack", oldBuildpackName, buildpackPath, "10")
							Eventually(session).Should(Exit(0))
						}, stacks[1])
					})

					Context("when renaming to unique name", func() {
						It("successfully renames the buildpack", func() {
							Eventually(session).Should(Say("Renaming buildpack %s to %s with stack %s as %s...", oldBuildpackName, newBuildpackName, stacks[0], username))
							Eventually(session).Should(Say("OK"))
							Eventually(session).Should(Exit(0))
						})
					})
				})
			})

			Context("when just one buildpack is found with the name/stack combo", func() {
				BeforeEach(func() {
					helpers.BuildpackWithStack(func(buildpackPath string) {
						session = helpers.CF("create-buildpack", oldBuildpackName, buildpackPath, "10")
						Eventually(session).Should(Exit(0))
					}, stacks[0])
				})

				Context("when renaming to unique name", func() {
					It("successfully renames the buildpack", func() {
						Eventually(session).Should(Say("Renaming buildpack %s to %s with stack %s as %s...", oldBuildpackName, newBuildpackName, stacks[0], username))
						Eventually(session).Should(Say("OK"))
						Eventually(session).Should(Exit(0))
					})
				})

				Context("when renaming to the same name as another buildpack", func() {
					Context("when the existing buildpack with the new name has the same stack", func() {
						BeforeEach(func() {
							helpers.BuildpackWithStack(func(buildpackPath string) {
								session = helpers.CF("create-buildpack", newBuildpackName, buildpackPath, "10")
								Eventually(session).Should(Exit(0))
							}, stacks[0])
						})

						It("returns a buildpack name/stack taken error", func() {
							Eventually(session).Should(Say("Renaming buildpack %s to %s with stack %s as %s...", oldBuildpackName, newBuildpackName, stacks[0], username))
							Eventually(session).Should(Say("FAILED"))
							Eventually(session.Err).Should(Say("The buildpack name %s is already in use with stack %s", newBuildpackName, stacks[0]))
							Eventually(session).Should(Exit(1))
						})
					})

					Context("when the existing buildpack with the new name has a different stack", func() {
						BeforeEach(func() {
							helpers.SkipIfOneStack()

							helpers.BuildpackWithStack(func(buildpackPath string) {
								session = helpers.CF("create-buildpack", newBuildpackName, buildpackPath, "10")
								Eventually(session).Should(Exit(0))
							}, stacks[1])
						})
						It("successfully renames the buildpack", func() {
							Eventually(session).Should(Say("Renaming buildpack %s to %s with stack %s as %s...", oldBuildpackName, newBuildpackName, stacks[0], username))
							Eventually(session).Should(Say("OK"))
							Eventually(session).Should(Exit(0))
						})
					})

					Context("when the existing buildpack with the new name has an empty stack", func() {
						BeforeEach(func() {
							helpers.BuildpackWithStack(func(buildpackPath string) {
								session = helpers.CF("create-buildpack", newBuildpackName, buildpackPath, "10")
								Eventually(session).Should(Exit(0))
							}, "")
							It("successfully renames the buildpack", func() {
								Eventually(session).Should(Say("Renaming buildpack %s to %s with stack %s as %s...", oldBuildpackName, newBuildpackName, stacks[0], username))
								Eventually(session).Should(Say("OK"))
								Eventually(session).Should(Exit(0))
							})
						})
					})
				})
			})
		})

		//If the user does not provide a stack, and there are multiple ambiguous buildpacks, we assume that they intended to rename the one with an empty stack.
		Context("when the user does not provide a stack", func() {
			var session *Session

			JustBeforeEach(func() {
				session = helpers.CF("rename-buildpack", oldBuildpackName, newBuildpackName)
			})

			Context("when no buildpacks with the old name exist", func() {
				It("returns a buildpack not found error", func() {
					Eventually(session).Should(Say("Renaming buildpack %s to %s as %s...", oldBuildpackName, newBuildpackName, username))
					Eventually(session).Should(Say("FAILED"))
					Eventually(session.Err).Should(Say("Buildpack %s not found", oldBuildpackName))
					Eventually(session).Should(Exit(1))
				})
			})

			Context("when there are multiple existing buildpacks with the old name", func() {
				Context("when none of the buildpacks has an empty stack", func() {
					BeforeEach(func() {
						helpers.SkipIfOneStack()
						helpers.BuildpackWithStack(func(buildpackPath string) {
							session = helpers.CF("create-buildpack", oldBuildpackName, buildpackPath, "10")
							Eventually(session).Should(Exit(0))
						}, stacks[0])
						helpers.BuildpackWithStack(func(buildpackPath string) {
							session = helpers.CF("create-buildpack", oldBuildpackName, buildpackPath, "10")
							Eventually(session).Should(Exit(0))
						}, stacks[1])
					})

					It("returns a buildpack not found error", func() {
						Eventually(session).Should(Say("Renaming buildpack %s to %s as %s...", oldBuildpackName, newBuildpackName, username))
						Eventually(session).Should(Say("FAILED"))
						Eventually(session.Err).Should(Say("Multiple buildpacks named %s found\\. Specify a stack name by using a '-s' flag\\.", oldBuildpackName))
						Eventually(session).Should(Exit(1))
					})
				})

				Context("when one of the existing buildpacks with the old name has an empty stack", func() {
					BeforeEach(func() {
						helpers.BuildpackWithStack(func(buildpackPath string) {
							session = helpers.CF("create-buildpack", oldBuildpackName, buildpackPath, "10")
							Eventually(session).Should(Exit(0))
						}, stacks[0])
						helpers.BuildpackWithStack(func(buildpackPath string) {
							session = helpers.CF("create-buildpack", oldBuildpackName, buildpackPath, "10")
							Eventually(session).Should(Exit(0))
						}, "")
					})

					Context("when renaming to unique name", func() {
						It("successfully renames the buildpack", func() {
							Eventually(session).Should(Say("Renaming buildpack %s to %s as %s...", oldBuildpackName, newBuildpackName, username))
							Eventually(session).Should(Say("OK"))
							Eventually(session).Should(Exit(0))
						})
					})

					Context("when renaming to the same name as another buildpack", func() {
						Context("when the existing buildpack with the new name has a non-empty stack", func() {
							BeforeEach(func() {
								helpers.SkipIfOneStack()
								helpers.BuildpackWithStack(func(buildpackPath string) {
									session = helpers.CF("create-buildpack", newBuildpackName, buildpackPath, "10")
									Eventually(session).Should(Exit(0))
								}, stacks[1])
							})

							It("successfully renames the buildpack", func() {
								Eventually(session).Should(Say("Renaming buildpack %s to %s as %s...", oldBuildpackName, newBuildpackName, username))
								Eventually(session).Should(Say("OK"))
								Eventually(session).Should(Exit(0))
							})
						})

						Context("when the existing buildpack with the new name has an empty stack", func() {
							BeforeEach(func() {
								helpers.BuildpackWithStack(func(buildpackPath string) {
									session = helpers.CF("create-buildpack", newBuildpackName, buildpackPath, "10")
									Eventually(session).Should(Exit(0))
								}, "")

								It("returns a buildpack name/stack taken error", func() {
									Eventually(session).Should(Say("Renaming buildpack %s to %s as %s...", oldBuildpackName, newBuildpackName, username))
									Eventually(session).Should(Say("FAILED"))
									Eventually(session.Err).Should(Say("Buildpack %s is already in use without a stack", newBuildpackName))
									Eventually(session).Should(Exit(1))
								})
							})
						})
					})
				})
			})
		})
	})
})
