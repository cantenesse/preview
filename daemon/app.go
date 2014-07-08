package daemon

import (
	"github.com/bmizerany/pat"
	"github.com/codegangsta/negroni"
	"github.com/etix/stoppableListener"
	"github.com/jherman3/zencoder"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/daemon/agent"
	"github.com/ngerakines/preview/daemon/api"
	"github.com/ngerakines/preview/daemon/storage"
	"github.com/rcrowley/go-metrics"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type daemonContext struct {
	registry                     metrics.Registry
	config                       *daemonConfig
	agentManager                 *agent.RenderAgentManager
	sourceAssetStorageManager    common.SourceAssetStorageManager
	generatedAssetStorageManager common.GeneratedAssetStorageManager
	templateManager              common.TemplateManager
	downloader                   common.Downloader
	uploader                     common.Uploader
	temporaryFileManager         common.TemporaryFileManager
	placeholderManager           common.PlaceholderManager
	signatureManager             api.SignatureManager
	simpleBlueprint              common.Blueprint
	assetBlueprint               common.Blueprint
	adminBlueprint               common.Blueprint
	staticBlueprint              common.Blueprint
	webhookBlueprint             common.Blueprint
	apiBlueprint                 common.Blueprint
	listener                     *stoppableListener.StoppableListener
	negroni                      *negroni.Negroni
	cassandraManager             *storage.CassandraManager
	mysqlManager                 *storage.MysqlManager
	zencoder                     *zencoder.Zencoder
}

func NewDaemonContext(config *daemonConfig) (*daemonContext, error) {
	log.Println("Creating application with config", config)
	app := new(daemonContext)
	app.registry = metrics.NewRegistry()

	metrics.RegisterRuntimeMemStats(app.registry)
	go metrics.CaptureRuntimeMemStats(app.registry, 60e9)

	app.config = config

	err := app.initAgentSupport()
	if err != nil {
		return nil, err
	}
	err = app.initStorage()
	if err != nil {
		return nil, err
	}
	if config.Zencoder.Enabled {
		err = app.initZencoder()
		if err != nil {
			return nil, err
		}
	}
	err = app.initRenderers()
	if err != nil {
		return nil, err
	}
	err = app.initApis()
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (app *daemonContext) Start() {
	httpListener, err := net.Listen("tcp", app.config.Http.Listen)
	if err != nil {
		panic(err)
	}
	app.listener = stoppableListener.Handle(httpListener)

	http.Serve(app.listener, app.negroni)

	if app.listener.Stopped {
		var alive int

		/* Wait at most 5 seconds for the clients to disconnect */
		for i := 0; i < 5; i++ {
			/* Get the number of clients still connected */
			alive = app.listener.ConnCount.Get()
			if alive == 0 {
				break
			}
			log.Printf("%d client(s) still connected…\n", alive)
			time.Sleep(1 * time.Second)
		}

		alive = app.listener.ConnCount.Get()
		if alive > 0 {
			log.Printf("Server stopped after 5 seconds with %d client(s) still connected.", alive)
		} else {
			log.Println("Server stopped gracefully.")
			os.Exit(0)
		}
	} else if err != nil {
		log.Println(err)
	}
}

func (app *daemonContext) initAgentSupport() error {

	placeholderGroups := make(map[string]string)

	for group, fileTypes := range app.config.Common.PlaceholderGroups {
		for _, fileType := range fileTypes {
			placeholderGroups[fileType] = group
		}
	}

	app.placeholderManager = newPlaceholderManager(app.config.Common.PlaceholderBasePath, placeholderGroups)

	app.temporaryFileManager = common.NewTemporaryFileManager()

	if app.config.Downloader.TramEnabled {
		tramHosts := app.config.Downloader.TramHosts
		app.downloader = newDownloader(app.config.Downloader.BasePath, app.config.Common.LocalAssetStoragePath, app.temporaryFileManager, true, tramHosts, app.buildS3Client())
	} else {
		app.downloader = newDownloader(app.config.Downloader.BasePath, app.config.Common.LocalAssetStoragePath, app.temporaryFileManager, false, []string{}, app.buildS3Client())
	}

	switch app.config.Uploader.Engine {
	case "s3":
		{
			s3Client := app.buildS3Client()
			buckets := app.config.S3.Buckets
			app.uploader = newUploader(buckets, s3Client)
		}
	case "local":
		{
			app.uploader = newLocalUploader(app.config.Common.LocalAssetStoragePath)
		}
	}

	return nil
}

func (app *daemonContext) initStorage() error {
	// NKG: This is where local (in-memory) or cassandra backed storage is
	// configured and the SourceAssetStorageManager,
	// GeneratedAssetStorageManager and TemplateManager objects are created
	// and placed into the app context.

	app.templateManager = common.NewTemplateManager()
	for _, template := range app.config.Templates {
		temp := new(common.Template)
		temp.Id = template.Id
		temp.RenderAgent = template.RenderAgent
		temp.Group = template.Group
		for k, v := range template.Attributes {
			temp.AddAttribute(k, v)
		}
		app.templateManager.Store(temp)
	}

	switch app.config.Storage.Engine {
	case "memory":
		{
			app.sourceAssetStorageManager = storage.NewSourceAssetStorageManager()
			app.generatedAssetStorageManager = storage.NewGeneratedAssetStorageManager(app.templateManager)
			return nil
		}
	case "mysql":
		{
			mysqlHost := app.config.Storage.MysqlHost
			mysqlUser := app.config.Storage.MysqlUser
			mysqlPassword := app.config.Storage.MysqlPassword
			mysqlDatabase := app.config.Storage.MysqlDatabase
			app.mysqlManager = storage.NewMysqlManager(mysqlHost, mysqlUser, mysqlPassword, mysqlDatabase)
			app.sourceAssetStorageManager, _ = storage.NewMysqlSourceAssetStorageManager(app.mysqlManager, app.config.Common.NodeId)
			app.generatedAssetStorageManager, _ = storage.NewMysqlGeneratedAssetStorageManager(app.mysqlManager, app.templateManager, app.config.Common.NodeId)
			return nil
		}
	case "cassandra":
		{
			log.Println("Using cassandra!")
			cassandraNodes := app.config.Storage.CassandraNodes
			keyspace := app.config.Storage.CassandraKeyspace
			cm, err := storage.NewCassandraManager(cassandraNodes, keyspace)
			if err != nil {
				return err
			}
			app.cassandraManager = cm
			app.sourceAssetStorageManager, err = storage.NewCassandraSourceAssetStorageManager(cm, app.config.Common.NodeId, keyspace)
			if err != nil {
				return err
			}
			app.generatedAssetStorageManager, err = storage.NewCassandraGeneratedAssetStorageManager(cm, app.templateManager, app.config.Common.NodeId, keyspace)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return common.ErrorNotImplemented
}

func (app *daemonContext) initRenderers() error {
	// NKG: This is where the RendererManager is constructed and renderers
	// are configured and enabled through it.
	fileTypes := make(map[string]map[string]map[string]int)
	for k, v := range app.config.RenderAgents {
		fileTypes[k] = v.FileTypes
	}
	app.agentManager = agent.NewRenderAgentManager(app.registry, app.sourceAssetStorageManager, app.generatedAssetStorageManager, app.templateManager, app.temporaryFileManager, app.uploader, app.config.Common.WorkDispatcherEnabled, app.zencoder, fileTypes)
	for k, v := range app.config.RenderAgents {
		app.agentManager.SetRenderAgentInfo(k, v.Enabled, v.Count)
		if v.Enabled {
			for i := 0; i < v.Count; i++ {
				app.agentManager.AddRenderAgent(k, v.RendererParams, app.downloader, app.uploader, 5)
			}
		}
	}
	return nil
}

func (app *daemonContext) initApis() error {
	// NKG: This is where different APIs are configured and enabled.

	allSupportedFileTypes := make(map[string]int64)

	app.signatureManager = api.NewSignatureManager()

	var err error

	p := pat.New()

	if app.config.SimpleApi.Enabled {
		app.simpleBlueprint, err = api.NewSimpleBlueprint(app.registry, app.config.SimpleApi.BaseUrl, app.config.SimpleApi.EdgeBaseUrl, app.agentManager, app.sourceAssetStorageManager, app.generatedAssetStorageManager, app.templateManager, app.placeholderManager, app.signatureManager, allSupportedFileTypes)
		if err != nil {
			return err
		}
		app.simpleBlueprint.AddRoutes(p)
	}
	s3Client := app.buildS3Client()

	// TODO: proper config
	app.apiBlueprint = api.NewApiBlueprint(app.config.SimpleApi.BaseUrl, app.agentManager, app.generatedAssetStorageManager, app.sourceAssetStorageManager, app.registry, s3Client)
	app.apiBlueprint.AddRoutes(p)

	app.assetBlueprint = api.NewAssetBlueprint(app.registry, app.config.Common.LocalAssetStoragePath, app.sourceAssetStorageManager, app.generatedAssetStorageManager, app.templateManager, app.placeholderManager, s3Client, app.signatureManager)
	app.assetBlueprint.AddRoutes(p)

	app.adminBlueprint = api.NewAdminBlueprint(app.registry,
		app.config.Source,
		[]string{},
		app.placeholderManager,
		app.temporaryFileManager,
		app.agentManager)
	app.adminBlueprint.AddRoutes(p)

	app.staticBlueprint = api.NewStaticBlueprint(app.placeholderManager)
	app.staticBlueprint.AddRoutes(p)

	app.webhookBlueprint = api.NewWebhookBlueprint(app.generatedAssetStorageManager, app.agentManager)
	app.webhookBlueprint.AddRoutes(p)

	app.negroni = negroni.Classic()
	app.negroni.UseHandler(p)

	return nil
}

func (app *daemonContext) initZencoder() error {
	apikey := app.config.Zencoder.Key
	app.zencoder = zencoder.NewZencoder(apikey)
	_, err := app.zencoder.GetAccount()
	if err != nil {
		log.Println("Invalid Zencoder key")
		return common.ErrorNotImplemented
	}
	return nil
}

func (app *daemonContext) Stop() {
	app.agentManager.Stop()
	if app.cassandraManager != nil {
		app.cassandraManager.Stop()
	}
	app.listener.Stop <- true
}

func (app *daemonContext) buildS3Client() common.S3Client {
	if app.config.Uploader.Engine != "s3" {
		return nil
	}
	awsKey := app.config.S3.Key
	awsSecret := app.config.S3.Secret
	awsHost := app.config.S3.Host
	verifySsl := app.config.S3.VerifySsl
	urlCompatMode := app.config.S3.UrlCompatMode
	log.Println("Creating s3 client with host", awsHost, "key", awsKey, "and secret", awsSecret)
	return common.NewAmazonS3Client(common.NewBasicS3Config(awsKey, awsSecret, awsHost, verifySsl, urlCompatMode))
}
