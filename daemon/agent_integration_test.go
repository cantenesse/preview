package daemon

import (
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/daemon/agent"
	"github.com/ngerakines/preview/daemon/storage"
	"github.com/ngerakines/testutils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rcrowley/go-metrics"
	"path/filepath"
	"strconv"
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

	AfterEach(func() {
		env.teardown()
	})

	// Put the phrase "Integration Test" somewhere in the It statement so that the test won't run on Travis
	It("Integration Test - Renders a docx", func() {
		env.setup(60, 10)
		runTest("ChefConf2014schedule.docx", "docx", []string{common.DocumentConversionTemplateId}, common.GeneratedAssetStatusComplete, 5, env)
	})

	It("Integration Test - Renders a multipage docx", func() {
		env.setup(60, 10)
		runTest("Multipage.docx", "docx", []string{common.DocumentConversionTemplateId}, common.GeneratedAssetStatusComplete, 13, env)
	})

	It("Integration Test - Renders a pdf", func() {
		env.setup(60, 10)
		runTest("ChefConf2014schedule.pdf", "pdf", common.LegacyDefaultTemplates, common.GeneratedAssetStatusComplete, 4, env)
	})

	It("Integration Test - Renders a multipage pdf", func() {
		env.setup(60, 10)
		runTest("Multipage.pdf", "pdf", common.LegacyDefaultTemplates, common.GeneratedAssetStatusComplete, 12, env)
	})

	It("Integration Test - Renders a jpg", func() {
		env.setup(60, 10)
		runTest("wallpaper-641916.jpg", "jpg", common.LegacyDefaultTemplates, common.GeneratedAssetStatusComplete, 4, env)
	})

	It("Integration Test - Renders a gif", func() {
		env.setup(60, 10)
		runTest("Animated.gif", "gif", common.LegacyDefaultTemplates, common.GeneratedAssetStatusComplete, 4, env)
	})

	It("Integration Test - Renders a png", func() {
		env.setup(60, 10)
		runTest("COW.png", "png", common.LegacyDefaultTemplates, common.GeneratedAssetStatusComplete, 4, env)
	})

	It("Integration Test - Renders a truncated multipage docx", func() {
		env.setup(60, 2)
		runTest("Multipage.docx", "docx", []string{common.DocumentConversionTemplateId}, common.GeneratedAssetStatusComplete, 9, env)
	})

	It("Integration Test - Renders a truncated multipage pdf", func() {
		env.setup(60, 2)
		runTest("Multipage.pdf", "pdf", common.LegacyDefaultTemplates, common.GeneratedAssetStatusComplete, 8, env)
	})

	It("Integration Test - Times out rendering a jpg", func() {
		env.setup(0, 10)
		runTest("wallpaper-641916.jpg", "jpg", common.LegacyDefaultTemplates, common.GeneratedAssetStatusFailed+","+common.ErrorRenderingTimedOut.Error(), 4, env)
	})
})

func (env *testEnvironment) setup(timeout, maxPages int) {
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
		"doc":  timeout,
		"docx": timeout,
		"ppt":  timeout,
		"pptx": timeout,
	}
	agentFileTypes["imageMagickRenderAgent"] = map[string]int{
		"jpg":  timeout,
		"jpeg": timeout,
		"png":  timeout,
		"pdf":  timeout,
		"gif":  timeout,
	}

	env.rm = agent.NewRenderAgentManager(env.registry, env.sasm, env.gasm, env.tm, env.tfm, env.uploader, true, nil, agentFileTypes)

	for i := 0; i < 4; i++ {
		env.rm.AddRenderAgent("documentRenderAgent", map[string]string{"tempFileBasePath": filepath.Join(env.dm.Path, "doc-cache")}, env.downloader, env.uploader, 5)
	}

	for i := 0; i < 8; i++ {
		env.rm.AddRenderAgent("imageMagickRenderAgent", map[string]string{"maxPages": strconv.Itoa(maxPages)}, env.downloader, env.uploader, 5)
	}
}

func (env *testEnvironment) teardown() {
	env.rm.Stop()
	env.dm.Close()
}

func runTest(file, fileType string, templateIds []string, status string, expectedCount int, env *testEnvironment) {
	sourceAssetId, err := common.NewUuid()
	Expect(err).To(BeNil())

	attributes := make(map[string][]string)
	attributes[common.SourceAssetAttributeSize] = []string{"12345"}
	attributes[common.SourceAssetAttributeType] = []string{fileType}

	env.rm.CreateWorkFromTemplates(sourceAssetId, fileUrl("test-data", file), attributes, templateIds)

	// 1 minute timeout, 1 second update interval
	Eventually(func() bool {
		return isComplete(sourceAssetId, status, env.gasm, expectedCount)
	}, 1*time.Minute, 1*time.Second).Should(BeTrue())
}

func isComplete(id, status string, generatedAssetStorageManager common.GeneratedAssetStorageManager, expectedCount int) bool {
	generatedAssets, err := generatedAssetStorageManager.FindBySourceAssetId(id)
	Expect(err).To(BeNil())
	count := 0
	for _, generatedAsset := range generatedAssets {
		if generatedAsset.Status == status {
			count += 1
		} else {
			Expect(generatedAsset.Status).ToNot(ContainSubstring(common.GeneratedAssetStatusFailed))
			return false
		}
	}
	Expect(count).To(Equal(expectedCount))
	return count == expectedCount
}

func fileUrl(dir, file string) string {
	return "file://" + filepath.Join(common.Cwd(), "../", dir, file)
}
