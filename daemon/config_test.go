package daemon

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Default config", func() {
	appConfig, err := loadDaemonConfig("")
	It("has valid syntax", func() {
		Expect(err).To(BeNil())
		Expect(appConfig).ToNot(BeNil())
	})
})
