package daemon

import (
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/daemon/agent"
	"github.com/ngerakines/preview/daemon/storage"
	"github.com/ngerakines/testutils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rcrowley/go-metrics"
	"log"
	"path/filepath"
	"time"
)

type testEnvironment struct {
	sasm       common.SourceAssetStorageManager
	gasm       common.GeneratedAssetStorageManager
	rm         *agent.RenderAgentManager
	dm         *testutils.DirectoryManager
	tfm        common.TemporaryFileManager
	downloader common.Downloader
	uploader   common.Uploader
	registry   metrics.Registry
	tm         common.TemplateManager
}

var _ = Describe("Agent integration test", func() {
	env := new(testEnvironment)

	BeforeEach(func() {
		env.setup()
	})

	AfterEach(func() {
		env.teardown()
	})

	// Put the phrase "Integration Test" somewhere in the It statement so that the test won't run on Travis
	It("Integration Test - Renders a docx", func() {
		runTest("ChefConf2014schedule.docx", "docx", []string{common.DocumentConversionTemplateId}, 5, env)
	})

	It("Integration Test - Renders a multipage docx", func() {
		runTest("Multipage.docx", "docx", []string{common.DocumentConversionTemplateId}, 13, env)
	})

	It("Integration Test - Renders a pdf", func() {
		runTest("ChefConf2014schedule.pdf", "pdf", common.LegacyDefaultTemplates, 4, env)
	})

	It("Integration Test - Renders a multipage pdf", func() {
		runTest("Multipage.pdf", "pdf", common.LegacyDefaultTemplates, 12, env)
	})

	It("Integration Test - Renders a jpg", func() {
		runTest("wallpaper-641916.jpg", "jpg", common.LegacyDefaultTemplates, 4, env)
	})

	It("Integration Test - Renders a gif", func() {
		runTest("Animated.gif", "gif", common.LegacyDefaultTemplates, 4, env)
	})

	It("Integration Test - Renders a png", func() {
		runTest("COW.png", "png", common.LegacyDefaultTemplates, 4, env)
	})
})

func (env *testEnvironment) setup() {
	env.dm = testutils.NewDirectoryManager()
	env.tm = common.NewTemplateManager()
	env.sasm = storage.NewSourceAssetStorageManager()
	env.gasm = storage.NewGeneratedAssetStorageManager(env.tm)

	env.tfm = common.NewTemporaryFileManager()
	env.downloader = newDownloader(env.dm.Path, env.dm.Path, env.tfm, false, []string{}, nil)
	env.uploader = newLocalUploader(env.dm.Path)
	env.registry = metrics.NewRegistry()

	agentFileTypes := make(map[string]map[string]int)
	agentFileTypes["documentRenderAgent"] = map[string]int{
		"doc":  60,
		"docx": 60,
		"ppt":  60,
		"pptx": 60,
	}
	agentFileTypes["imageMagickRenderAgent"] = map[string]int{
		"jpg":  60,
		"jpeg": 60,
		"png":  60,
		"pdf":  60,
		"gif":  60,
	}

	env.rm = agent.NewRenderAgentManager(env.registry, env.sasm, env.gasm, env.tm, env.tfm, env.uploader, true, nil, agentFileTypes)

	for i := 0; i < 4; i++ {
		env.rm.AddRenderAgent("documentRenderAgent", map[string]string{"tempFileBasePath": filepath.Join(env.dm.Path, "doc-cache")}, env.downloader, env.uploader, 5)
	}

	for i := 0; i < 8; i++ {
		env.rm.AddRenderAgent("imageMagickRenderAgent", map[string]string{"maxPages": "10"}, env.downloader, env.uploader, 5)
	}
}

func (env *testEnvironment) teardown() {
	env.rm.Stop()
	env.dm.Close()
}

func runTest(file, fileType string, templateIds []string, expectedCount int, env *testEnvironment) {
	sourceAssetId, err := common.NewUuid()
	Expect(err).To(BeNil())

	sourceAsset, err := common.NewSourceAsset(sourceAssetId, common.SourceAssetTypeOrigin)
	Expect(err).To(BeNil())

	sourceAsset.AddAttribute(common.SourceAssetAttributeSize, []string{"12345"})
	sourceAsset.AddAttribute(common.SourceAssetAttributeSource, []string{fileUrl("test-data", file)})
	sourceAsset.AddAttribute(common.SourceAssetAttributeType, []string{fileType})

	err = env.sasm.Store(sourceAsset)
	Expect(err).To(BeNil())

	templates, err := env.tm.FindByIds(templateIds)
	Expect(err).To(BeNil())

	for _, template := range templates {
		ga, err := common.NewGeneratedAssetFromSourceAsset(sourceAsset, template.Id, env.uploader.Url(sourceAsset, template, 0))
		Expect(err).To(BeNil())

		err = env.gasm.Store(ga)
		Expect(err).To(BeNil())

		log.Println(ga)
	}

	env.rm.DispatchMoreWork()

	// 1 minute timeout, 1 second update interval
	Eventually(func() bool {
		return isComplete(sourceAssetId, env.gasm, expectedCount)
	}, 1*time.Minute, 1*time.Second).Should(BeTrue())
}

func isComplete(id string, generatedAssetStorageManager common.GeneratedAssetStorageManager, expectedCount int) bool {
	generatedAssets, err := generatedAssetStorageManager.FindBySourceAssetId(id)
	Expect(err).To(BeNil())
	count := 0
	for _, generatedAsset := range generatedAssets {
		if generatedAsset.Status == common.GeneratedAssetStatusComplete {
			count += 1
		}
		Expect(generatedAsset.Status).ToNot(ContainSubstring(common.GeneratedAssetStatusFailed))
	}
	if count == expectedCount {
		return true
	}
	if count > 0 {
		log.Println("Count is", count, "but wanted", expectedCount)
		return false
	}
	return false
}

func fileUrl(dir, file string) string {
	return "file://" + filepath.Join(common.Cwd(), "../", dir, file)
}
