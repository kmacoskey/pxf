package cmd_test

import (
	"fmt"
	"pxf-cli/cmd"
	"pxf-cli/pxf"

	"github.com/greenplum-db/gp-common-go-libs/cluster"
	"github.com/greenplum-db/gp-common-go-libs/operating"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pkg/errors"
)

var (
	configMaster = cluster.SegConfig{ContentID: -1, Hostname: "mdw", DataDir: "/data/gpseg-1"}
	configSegOne = cluster.SegConfig{ContentID: 0, Hostname: "sdw1", DataDir: "/data/gpseg0"}
	configSegTwo = cluster.SegConfig{ContentID: 1, Hostname: "sdw2", DataDir: "/data/gpseg1"}
	clusterData  = &cmd.ClusterData{
		Cluster:  cluster.NewCluster([]cluster.SegConfig{configMaster, configSegOne, configSegTwo}),
		NumHosts: 0,
		Output:   nil,
	}
)

var _ = Describe("GenerateHostList", func() {
	It("Returns the correct hostlist and master hostname", func() {
		operating.System.Hostname = func() (string, error) {
			return "mdw", nil
		}
		err := clusterData.CountHostsAndValidateMaster()
		Expect(err).To(BeNil())
		Expect(clusterData.NumHosts).To(Equal(2))
	})
	It("Errors if not running from master host", func() {
		operating.System.Hostname = func() (string, error) {
			return "fake-host", nil
		}
		err := clusterData.CountHostsAndValidateMaster()
		Expect(err.Error()).To(Equal("ERROR: pxf cluster commands should only be run from Greenplum master"))
		Expect(clusterData.NumHosts).To(Equal(-1))
	})
})

var _ = Describe("GenerateOutput", func() {
	BeforeEach(func() {
		clusterData.Output = &cluster.RemoteOutput{
			NumErrors: 0,
			Stderrs:   map[int]string{-1: "", 0: "", 1: ""},
			Stdouts:   map[int]string{-1: "everything fine", 0: "everything fine", 1: "everything fine"},
			Errors:    map[int]error{-1: nil, 0: nil, 1: nil},
		}
	})
	Describe("Running supported commands", func() {
		Context("When all hosts are successful", func() {
			It("Reports all hosts initialized successfully", func() {
				_ = cmd.GenerateOutput(&pxf.Init, clusterData)
				Expect(testStdout).To(gbytes.Say("PXF initialized successfully on 3 out of 3 hosts"))
			})

			It("Reports all hosts started successfully", func() {
				_ = cmd.GenerateOutput(&pxf.Start, clusterData)
				Expect(testStdout).To(gbytes.Say("PXF started successfully on 3 out of 3 hosts"))
			})

			It("Reports all hosts stopped successfully", func() {
				_ = cmd.GenerateOutput(&pxf.Stop, clusterData)
				Expect(testStdout).To(gbytes.Say("PXF stopped successfully on 3 out of 3 hosts"))
			})

			It("Reports all hosts synced successfully", func() {
				_ = cmd.GenerateOutput(&pxf.Sync, clusterData)
				Expect(testStdout).To(gbytes.Say("PXF configs synced successfully on 3 out of 3 hosts"))
			})
		})

		Context("When some hosts fail", func() {
			var expectedError string
			BeforeEach(func() {
				clusterData.Output = &cluster.RemoteOutput{
					NumErrors: 1,
					Stderrs:   map[int]string{-1: "", 0: "", 1: "an error happened on sdw2"},
					Stdouts:   map[int]string{-1: "everything fine", 0: "everything fine", 1: "something wrong"},
					Errors:    map[int]error{-1: nil, 0: nil, 1: errors.New("some error")},
				}
				expectedError = "sdw2 ==> an error happened on sdw2"
			})
			It("Reports the number of hosts that failed to initialize", func() {
				_ = cmd.GenerateOutput(&pxf.Init, clusterData)
				Expect(testStdout).Should(gbytes.Say("PXF failed to initialize on 1 out of 3 hosts"))
				Expect(testStderr).Should(gbytes.Say(expectedError))
			})

			It("Reports the number of hosts that failed to start", func() {
				_ = cmd.GenerateOutput(&pxf.Start, clusterData)
				Expect(testStdout).Should(gbytes.Say("PXF failed to start on 1 out of 3 hosts"))
				Expect(testStderr).Should(gbytes.Say(expectedError))
			})

			It("Reports the number of hosts that failed to stop", func() {
				_ = cmd.GenerateOutput(&pxf.Stop, clusterData)
				Expect(testStdout).Should(gbytes.Say("PXF failed to stop on 1 out of 3 hosts"))
				Expect(testStderr).Should(gbytes.Say(expectedError))
			})

			It("Reports the number of hosts that failed to sync", func() {
				_ = cmd.GenerateOutput(&pxf.Sync, clusterData)
				Expect(testStdout).Should(gbytes.Say("PXF configs failed to sync on 1 out of 3 hosts"))
				Expect(testStderr).Should(gbytes.Say(expectedError))
			})
		})

		Context("Before the command returns", func() {
			BeforeEach(func() {
				clusterData.NumHosts = 2
			})
			It("Reports the number of hosts that will be initialized", func() {
				_ = cmd.GenerateStatusReport(&pxf.Init, clusterData)
				Expect(testStdout).Should(gbytes.Say("Initializing PXF on master and 2 segment hosts..."))
			})

			It("Reports the number of hosts will be started", func() {
				_ = cmd.GenerateStatusReport(&pxf.Start, clusterData)
				Expect(testStdout).Should(gbytes.Say("Starting PXF on 2 segment hosts..."))
			})

			It("Reports the number of hosts that will be stopped", func() {
				_ = cmd.GenerateStatusReport(&pxf.Stop, clusterData)
				Expect(testStdout).Should(gbytes.Say("Stopping PXF on 2 segment hosts..."))
			})

			It("Reports the number of hosts that will be synced", func() {
				_ = cmd.GenerateStatusReport(&pxf.Sync, clusterData)
				Expect(testStdout).Should(gbytes.Say("Syncing PXF configuration files to 2 hosts..."))
			})
		})

		Context("When we see messages in Stderr, but NumErrors is 0", func() {
			It("Reports all hosts were successful", func() {
				clusterData.Output = &cluster.RemoteOutput{
					NumErrors: 0,
					Stderrs:   map[int]string{-1: "typical stderr", 0: "typical stderr", 1: "typical stderr"},
					Stdouts:   map[int]string{-1: "typical stdout", 0: "typical stdout", 1: "typical stdout"},
					Errors:    map[int]error{-1: nil, 0: nil, 1: nil},
				}
				_ = cmd.GenerateOutput(&pxf.Stop, clusterData)
				Expect(testStdout).To(gbytes.Say("PXF stopped successfully on 3 out of 3 hosts"))
			})
		})

		Context("When a command fails, and output is multiline", func() {
			It("Truncates the output to two lines", func() {
				stderr := `stderr line one
stderr line two
stderr line three`
				clusterData.Output = &cluster.RemoteOutput{
					NumErrors: 1,
					Stderrs:   map[int]string{-1: "", 0: "", 1: stderr},
					Stdouts:   map[int]string{-1: "everything fine", 0: "everything fine", 1: "everything not fine"},
					Errors:    map[int]error{-1: nil, 0: nil, 1: errors.New("some error")},
				}
				expectedError := `sdw2 ==> stderr line one
stderr line two...`
				_ = cmd.GenerateOutput(&pxf.Stop, clusterData)
				Expect(testStdout).Should(gbytes.Say(fmt.Sprintf("PXF failed to stop on 1 out of 3 hosts")))
				Expect(testStderr).Should(gbytes.Say(expectedError))
			})
		})

		Context("When NumErrors is non-zero, but Stderr is empty", func() {
			It("Reports Stdout in error message", func() {
				clusterData.Output = &cluster.RemoteOutput{
					NumErrors: 1,
					Stderrs:   map[int]string{-1: "", 0: "", 1: ""},
					Stdouts:   map[int]string{-1: "everything fine", 0: "everything fine", 1: "something wrong on sdw2\nstderr line2\nstderr line3"},
					Errors:    map[int]error{-1: nil, 0: nil, 1: errors.New("some error")},
				}
				expectedError := "sdw2 ==> something wrong on sdw2\nstderr line2..."
				_ = cmd.GenerateOutput(&pxf.Stop, clusterData)
				Expect(testStdout).Should(gbytes.Say("PXF failed to stop on 1 out of 3 hosts"))
				Expect(testStderr).Should(gbytes.Say(expectedError))
			})
		})
	})
})
