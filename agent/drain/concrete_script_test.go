package drain_test

import (
	"errors"

	. "github.com/cloudfoundry/bosh-agent/internal/github.com/onsi/ginkgo"
	. "github.com/cloudfoundry/bosh-agent/internal/github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/drain"
	boshsys "github.com/cloudfoundry/bosh-agent/internal/github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-agent/internal/github.com/cloudfoundry/bosh-utils/system/fakes"
)

type fakeParams struct {
	jobChange       string
	hashChange      string
	updatedPackages []string

	jobState    string
	jobStateErr error

	jobNextState    string
	jobNextStateErr error
}

func (p fakeParams) JobChange() (change string)       { return p.jobChange }
func (p fakeParams) HashChange() (change string)      { return p.hashChange }
func (p fakeParams) UpdatedPackages() (pkgs []string) { return p.updatedPackages }

func (p fakeParams) JobState() (string, error)     { return p.jobState, p.jobStateErr }
func (p fakeParams) JobNextState() (string, error) { return p.jobNextState, p.jobNextStateErr }

var _ = Describe("ConcreteScript", func() {
	var (
		runner *fakesys.FakeCmdRunner
		fs     *fakesys.FakeFileSystem
		script ConcreteScript
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		runner = fakesys.NewFakeCmdRunner()
		script = NewConcreteScript(fs, runner, "/fake/script")
	})

	Describe("Run", func() {
		var (
			params fakeParams
		)

		BeforeEach(func() {
			params = fakeParams{
				jobChange:       "job_shutdown",
				hashChange:      "hash_unchanged",
				updatedPackages: []string{"foo", "bar"},
			}
		})

		It("runs drain script", func() {
			commandResult := fakesys.FakeCmdResult{Stdout: "1"}
			runner.AddCmdResult("/fake/script job_shutdown hash_unchanged foo bar", commandResult)

			_, err := script.Run(params)
			Expect(err).ToNot(HaveOccurred())

			expectedCmd := boshsys.Command{
				Name: "/fake/script",
				Args: []string{"job_shutdown", "hash_unchanged", "foo", "bar"},
				Env: map[string]string{
					"PATH": "/usr/sbin:/usr/bin:/sbin:/bin",
				},
			}

			Expect(len(runner.RunComplexCommands)).To(Equal(1))
			Expect(runner.RunComplexCommands[0]).To(Equal(expectedCmd))
		})

		It("returns parsed stdout", func() {
			commandResult := fakesys.FakeCmdResult{Stdout: "1"}
			runner.AddCmdResult("/fake/script job_shutdown hash_unchanged foo bar", commandResult)

			value, err := script.Run(params)
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal(1))
		})

		It("returns parsed stdout after trimming", func() {
			commandResult := fakesys.FakeCmdResult{Stdout: "-56\n"}
			runner.AddCmdResult("/fake/script job_shutdown hash_unchanged foo bar", commandResult)

			value, err := script.Run(params)
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal(-56))
		})

		It("returns error with non integer stdout", func() {
			commandResult := fakesys.FakeCmdResult{Stdout: "hello!"}
			runner.AddCmdResult("/fake/script job_shutdown hash_unchanged foo bar", commandResult)

			_, err := script.Run(params)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when running command errors", func() {
			commandResult := fakesys.FakeCmdResult{Error: errors.New("woops")}
			runner.AddCmdResult("/fake/script job_shutdown hash_unchanged foo bar", commandResult)

			_, err := script.Run(params)
			Expect(err).To(HaveOccurred())
		})

		Describe("job state", func() {
			It("sets the BOSH_JOB_STATE env variable if job state is present", func() {
				params.jobState = "fake-job-state"

				_, err := script.Run(params)
				Expect(err).To(HaveOccurred())

				Expect(len(runner.RunComplexCommands)).To(Equal(1))

				env := runner.RunComplexCommands[0].Env
				Expect(env["BOSH_JOB_STATE"]).To(Equal("fake-job-state"))
			})

			It("does not set the BOSH_JOB_STATE env variable if job state is empty", func() {
				params.jobState = ""

				_, err := script.Run(params)
				Expect(err).To(HaveOccurred())

				Expect(len(runner.RunComplexCommands)).To(Equal(1))
				Expect(runner.RunComplexCommands[0].Env).ToNot(HaveKey("BOSH_JOB_STATE"))
			})

			It("returns error when cannot get the job state and does not run drain script", func() {
				params.jobStateErr = errors.New("fake-job-state-err")

				_, err := script.Run(params)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-job-state-err"))

				Expect(len(runner.RunComplexCommands)).To(Equal(0))
			})
		})

		Describe("job next state", func() {
			It("sets the BOSH_JOB_NEXT_STATE env variable if job next state is present", func() {
				params.jobNextState = "fake-job-next-state"

				_, err := script.Run(params)
				Expect(err).To(HaveOccurred())

				Expect(len(runner.RunComplexCommands)).To(Equal(1))

				env := runner.RunComplexCommands[0].Env
				Expect(env["BOSH_JOB_NEXT_STATE"]).To(Equal("fake-job-next-state"))
			})

			It("does not set the BOSH_JOB_NEXT_STATE env variable if job next state is empty", func() {
				params.jobNextState = ""

				_, err := script.Run(params)
				Expect(err).To(HaveOccurred())

				Expect(len(runner.RunComplexCommands)).To(Equal(1))
				Expect(runner.RunComplexCommands[0].Env).ToNot(HaveKey("BOSH_JOB_NEXT_STATE"))
			})

			It("returns error when cannot get the job next state and does not run drain script", func() {
				params.jobNextStateErr = errors.New("fake-job-next-state-err")

				_, err := script.Run(params)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-job-next-state-err"))

				Expect(len(runner.RunComplexCommands)).To(Equal(0))
			})
		})
	})

	Describe("Exists", func() {
		It("returns bool", func() {
			commandResult := fakesys.FakeCmdResult{Stdout: "1"}
			runner.AddCmdResult("/fake/script job_shutdown hash_unchanged foo bar", commandResult)

			Expect(script.Exists()).To(BeFalse())

			fs.WriteFile("/fake/script", []byte{})
			Expect(script.Exists()).To(BeTrue())
		})
	})
})
