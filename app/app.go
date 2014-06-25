package app

import (
	"github.com/bmizerany/pat"
	"github.com/codegangsta/negroni"
	"github.com/etix/stoppableListener"
	"github.com/jherman3/zencoder"
	"github.com/ngerakines/preview/api"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/config"
	"github.com/ngerakines/preview/render"
	"github.com/rcrowley/go-metrics"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type AppContext struct {
	registry                     metrics.Registry
	appConfig                    *config.AppConfig
	agentManager                 *render.RenderAgentManager
	sourceAssetStorageManager    common.SourceAssetStorageManager
	generatedAssetStorageManager common.GeneratedAssetStorageManager
	templateManager              common.TemplateManager
	downloader                   common.Downloader
	uploader                     common.Uploader
	temporaryFileManager         common.TemporaryFileManager
	placeholderManager           common.PlaceholderManager
	signatureManager             api.SignatureManager
	simpleBlueprint              api.Blueprint
	assetBlueprint               api.Blueprint
	adminBlueprint               api.Blueprint
	staticBlueprint              api.Blueprint
	webhookBlueprint             api.Blueprint
	apiBlueprint                 api.Blueprint
	listener                     *stoppableListener.StoppableListener
	negroni                      *negroni.Negroni
	cassandraManager             *common.CassandraManager
	mysqlManager                 *common.MysqlManager
	zencoder                     *zencoder.Zencoder
}

func NewApp(appConfig *config.AppConfig) (*AppContext, error) {
	log.Println("Creating application with config", appConfig)
	app := new(AppContext)
	app.registry = metrics.NewRegistry()

	metrics.RegisterRuntimeMemStats(app.registry)
	go metrics.CaptureRuntimeMemStats(app.registry, 60e9)

	app.appConfig = appConfig

	err := app.initTrams()
	if err != nil {
		return nil, err
	}
	err = app.initStorage()
	if err != nil {
		return nil, err
	}
	if appConfig.VideoRenderAgent.Enabled {
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

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	return app, nil
}

func (app *AppContext) Start() {
	httpListener, err := net.Listen("tcp", app.appConfig.Http.Listen)
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
			log.Fatalf("Server stopped after 5 seconds with %d client(s) still connected.", alive)
		} else {
			log.Println("Server stopped gracefully.")
			os.Exit(0)
		}
	} else if err != nil {
		log.Fatal(err)
	}
}

func (app *AppContext) initTrams() error {
	app.placeholderManager = common.NewPlaceholderManager(app.appConfig)
	app.temporaryFileManager = common.NewTemporaryFileManager()
	if app.appConfig.Downloader.TramEnabled {
		tramHosts := app.appConfig.Downloader.TramHosts
		app.downloader = common.NewDownloader(app.appConfig.Downloader.BasePath, app.appConfig.Common.LocalAssetStoragePath, app.temporaryFileManager, true, tramHosts, app.buildS3Client())
	} else {
		app.downloader = common.NewDownloader(app.appConfig.Downloader.BasePath, app.appConfig.Common.LocalAssetStoragePath, app.temporaryFileManager, false, []string{}, app.buildS3Client())
	}

	switch app.appConfig.Uploader.Engine {
	case "s3":
		{
			s3Client := app.buildS3Client()
			buckets := app.appConfig.S3.Buckets
			app.uploader = common.NewUploader(buckets, s3Client)
		}
	case "local":
		{
			app.uploader = common.NewLocalUploader(app.appConfig.Common.LocalAssetStoragePath)
		}
	}

	return nil
}

func (app *AppContext) initStorage() error {
	// NKG: This is where local (in-memory) or cassandra backed storage is
	// configured and the SourceAssetStorageManager,
	// GeneratedAssetStorageManager and TemplateManager objects are created
	// and placed into the app context.

	app.templateManager = common.NewTemplateManager()
	for _, template := range app.appConfig.Templates {
		temp := new(common.Template)
		temp.Id = template.Id
		temp.RenderAgent = template.RenderAgent
		temp.Group = template.Group
		for k, v := range template.Attributes {
			temp.AddAttribute(k, v)
		}
		app.templateManager.Store(temp)
	}

	switch app.appConfig.Storage.Engine {
	case "memory":
		{
			app.sourceAssetStorageManager = common.NewSourceAssetStorageManager()
			app.generatedAssetStorageManager = common.NewGeneratedAssetStorageManager(app.templateManager)
			return nil
		}
	case "mysql":
		{
			mysqlHost := app.appConfig.Storage.MysqlHost
			mysqlUser := app.appConfig.Storage.MysqlUser
			mysqlPassword := app.appConfig.Storage.MysqlPassword
			mysqlDatabase := app.appConfig.Storage.MysqlDatabase
			app.mysqlManager = common.NewMysqlManager(mysqlHost, mysqlUser, mysqlPassword, mysqlDatabase)
			app.sourceAssetStorageManager, _ = common.NewMysqlSourceAssetStorageManager(app.mysqlManager, app.appConfig.Common.NodeId)
			app.generatedAssetStorageManager, _ = common.NewMysqlGeneratedAssetStorageManager(app.mysqlManager, app.templateManager, app.appConfig.Common.NodeId)
			return nil
		}
	case "cassandra":
		{
			log.Println("Using cassandra!")
			cassandraNodes := app.appConfig.Storage.CassandraNodes
			keyspace := app.appConfig.Storage.CassandraKeyspace
			cm, err := common.NewCassandraManager(cassandraNodes, keyspace)
			if err != nil {
				return err
			}
			app.cassandraManager = cm
			app.sourceAssetStorageManager, err = common.NewCassandraSourceAssetStorageManager(cm, app.appConfig.Common.NodeId, keyspace)
			if err != nil {
				return err
			}
			app.generatedAssetStorageManager, err = common.NewCassandraGeneratedAssetStorageManager(cm, app.templateManager, app.appConfig.Common.NodeId, keyspace)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return common.ErrorNotImplemented
}

func (app *AppContext) initRenderers() error {
	// NKG: This is where the RendererManager is constructed and renderers
	// are configured and enabled through it.
	app.agentManager = render.NewRenderAgentManager(app.registry, app.sourceAssetStorageManager, app.generatedAssetStorageManager, app.templateManager, app.temporaryFileManager, app.uploader, app.appConfig.Common.WorkDispatcherEnabled, app.zencoder, app.appConfig.VideoRenderAgent.ZencoderS3Bucket, app.appConfig.VideoRenderAgent.ZencoderNotificationUrl, app.appConfig.DocumentRenderAgent.SupportedFileTypes, app.appConfig.ImageMagickRenderAgent.SupportedFileTypes, app.appConfig.VideoRenderAgent.SupportedFileTypes)
	app.agentManager.SetRenderAgentInfo(common.RenderAgentImageMagick, app.appConfig.ImageMagickRenderAgent.Enabled, app.appConfig.ImageMagickRenderAgent.Count)
	app.agentManager.SetRenderAgentInfo(common.RenderAgentDocument, app.appConfig.DocumentRenderAgent.Enabled, app.appConfig.DocumentRenderAgent.Count)
	app.agentManager.SetRenderAgentInfo(common.RenderAgentVideo, app.appConfig.VideoRenderAgent.Enabled, app.appConfig.VideoRenderAgent.Count)
	if app.appConfig.ImageMagickRenderAgent.Enabled {
		for i := 0; i < app.appConfig.ImageMagickRenderAgent.Count; i++ {
			app.agentManager.AddImageMagickRenderAgent(app.downloader, app.uploader, 5)
		}
	}
	if app.appConfig.DocumentRenderAgent.Enabled {
		for i := 0; i < app.appConfig.DocumentRenderAgent.Count; i++ {
			app.agentManager.AddDocumentRenderAgent(app.downloader, app.uploader, app.appConfig.DocumentRenderAgent.BasePath, 5)
		}
	}
	if app.appConfig.VideoRenderAgent.Enabled {
		for i := 0; i < app.appConfig.VideoRenderAgent.Count; i++ {
			app.agentManager.AddVideoRenderAgent(nil, nil, 5)
		}
	}
	return nil
}

func (app *AppContext) initApis() error {
	// NKG: This is where different APIs are configured and enabled.

	allSupportedFileTypes := make(map[string]int64)

	app.signatureManager = api.NewSignatureManager()

	var err error

	p := pat.New()

	if app.appConfig.SimpleApi.Enabled {
		app.simpleBlueprint, err = api.NewSimpleBlueprint(app.registry, app.appConfig.SimpleApi.BaseUrl, app.appConfig.SimpleApi.EdgeBaseUrl, app.agentManager, app.sourceAssetStorageManager, app.generatedAssetStorageManager, app.templateManager, app.placeholderManager, app.signatureManager, allSupportedFileTypes)
		if err != nil {
			return err
		}
		app.simpleBlueprint.AddRoutes(p)
	}
	s3Client := app.buildS3Client()

	// TODO: proper config
	app.apiBlueprint = api.NewApiBlueprint(app.appConfig.SimpleApi.BaseUrl, app.agentManager, app.generatedAssetStorageManager, app.sourceAssetStorageManager, app.registry, s3Client, app.appConfig.Common.LocalAssetStoragePath)
	app.apiBlueprint.AddRoutes(p)

	app.assetBlueprint = api.NewAssetBlueprint(app.registry, app.appConfig.Common.LocalAssetStoragePath, app.sourceAssetStorageManager, app.generatedAssetStorageManager, app.templateManager, app.placeholderManager, s3Client, app.signatureManager)
	app.assetBlueprint.AddRoutes(p)

	app.adminBlueprint = api.NewAdminBlueprint(app.registry, app.appConfig, app.placeholderManager, app.temporaryFileManager, app.agentManager)
	app.adminBlueprint.AddRoutes(p)

	app.staticBlueprint = api.NewStaticBlueprint(app.placeholderManager)
	app.staticBlueprint.AddRoutes(p)

	app.webhookBlueprint = api.NewWebhookBlueprint(app.generatedAssetStorageManager, app.agentManager)
	app.webhookBlueprint.AddRoutes(p)

	app.negroni = negroni.Classic()
	app.negroni.UseHandler(p)

	return nil
}

func (app *AppContext) initZencoder() error {
	apikey := app.appConfig.VideoRenderAgent.ZencoderKey
	app.zencoder = zencoder.NewZencoder(apikey)
	_, err := app.zencoder.GetAccount()
	if err != nil {
		log.Println("Invalid Zencoder key")
		return common.ErrorNotImplemented
	}
	return nil
}

func (app *AppContext) Stop() {
	panic("ok")
	app.agentManager.Stop()
	if app.cassandraManager != nil {
		app.cassandraManager.Stop()
	}
	app.listener.Stop <- true
}

func (app *AppContext) buildS3Client() common.S3Client {
	if app.appConfig.Uploader.Engine != "s3" {
		return nil
	}
	awsKey := app.appConfig.S3.Key
	awsSecret := app.appConfig.S3.Secret
	awsHost := app.appConfig.S3.Host
	verifySsl := app.appConfig.S3.VerifySsl
	urlCompatMode := app.appConfig.S3.UrlCompatMode
	log.Println("Creating s3 client with host", awsHost, "key", awsKey, "and secret", awsSecret)
	return common.NewAmazonS3Client(common.NewBasicS3Config(awsKey, awsSecret, awsHost, verifySsl, urlCompatMode))
}
