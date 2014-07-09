package daemon

import (
	"testing"

	. "github.com/franela/goblin"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	g := Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) {
		g.Fail(m)
	})
	g.Describe("default config", func() {
		g.It("has valid syntax", func() {
			conf, err := loadDaemonConfig("")
			Expect(err).To(BeNil())
			Expect(conf).ToNot(BeNil())
		})
	})
}
