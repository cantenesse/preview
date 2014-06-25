# Preview

> The work you do while you procrastinate is probably the work you should be doing for the rest of your life.

This is the `preview` project, it provides a service to create, cache and serve preview images for different types of files.

## Rendering

The primary rendering agent uses image magic to create resized images using the `convert` command. It supports the following file types:

* jpg
* jpeg
* png

# Configuration

The preview application uses JSON for configuration. It will attempt to load configuration from the following paths, in this order:

1. ./preview.config
2. ~/.preview.config
3. /etc/preview.config

Additionally, using the `-c` or `--config` command line arguments, a configuration file can be passed when starting the application.

    $ preview -c servera.config

The configuration object has the following top level sections:

* common
* http
* storage
* imageMagickRenderAgent
* documentRenderAgent
* videoRenderAgent
* simpleApi
* assetApi
* uploader
* s3
* downloader

The "common" group has the following keys:

* "nodeId" - The unique identifier of the preview instance.
* "placeholderBasePath" - The directory that contains placeholder image information.
* "placeholderGroups" - A map of grouped types of file types to groups used to determine the availability of file types when displaying placeholder images.
* "localAssetStoragePath" - The location of locally stored assets.

The "http" group has the following keys:

* "listen" -  The binding pattern for the HTTP interface.

The "storage" group has the following keys:

* "engine" - The storage engine to use to persist source assets and group assets.
* "cassandraNodes" - An array of strings representing cassandra nodes to interact with. Only available when the engine is "cassandra".
* "cassandraKeyspace" - The cassandra keyspace that queries are executed against. Only available when the engine is "cassandra".

The "documentRenderAgent" group has the following keys:

* "enabled" - Used to determine if the document rendering agent should be started with the application.
* "count" - The number of agents to run concurrently.
* "basePath" - The path of the temporary directory to be used by the agent.
* "supportedFileTypes" - An array of strings corresponding to file types to be supported by this render agent.

The "videoRenderAgent" group has the following keys:

* "enabled" - Used to determine if the document rendering agent should be started with the application.
* "count" - The number of agents to run concurrently.
* "basePath" - The path of the temporary directory to be used by the agent.
* "zencoderKey" - The Zencoder API key of an account to be used for the transcoding process.
* "zencoderNotificationUrl" - The URL of the server to listen for updates from Zencoder.
* "zencoderS3Bucket" - The S3 bucket for Zencoder to use as output.
* "supportedFileTypes" - An array of strings corresponding to file types to be supported by this render agent.

The "imageMagickRenderAgent" group has the following keys:

* "enabled" - Used to determine if the image magick rendering agent should be started with the application.
* "count" - The number of agents to run concurrently.
* "supportedFileTypes" - An array of strings corresponding to file types to be supported by this render agent.

The "simpleApi" group has the following keys:

* "enabled" - If enabled, the simple API will be available.
* "baseUrl" - The url prefix to use, defaulting to "/api".
* "edgeBaseUrl" - The base URL used when crafting links to renders and placeholders.

The "assetApi" group has the following keys:

* "enabled" - If enabled, the simple API will be available with the "/asset" base URL on the listen port.

The "uploader" group has the following keys:

* "engine" - The engine to use when uploading rendered images.

The "s3" group is only available when the uploader engine is "s3". It has the following keys:

* "key" - The AWS key to use when uploading rendered images to S3. 
* "secret" - The AWS secret to use when uploading rendered images to S3. 
* "host" - The host and base URL used to submit requests to when uploading rendered images to S3. Required to be of format "https://#{bucket}.s3.amazonaws.com" unless "urlCompatMode" is true. 
* "buckets" - A list of buckets used to distribute rendered images to when uploading rendered images to S3.
* "verifySsl"
* "urlCompatMode" - Allows "host" to be of format: "s3://#{bucket}".

The "downloader" group has the following keys:

* "basePath" - The directory that downloaded files are stored to.
* "tramEnabled"
* "tramHosts"

## Default Configuration

By default, the application will use the following configuration json:

```json
{
   "common": {
      "placeholderBasePath":"/var/preview/placeholders",
      "placeholderGroups": {
         "image":["jpg", "jpeg", "png", "gif", "pdf"],
         "document":["doc", "docx"],
         "video":["mp4"]
      },
      "localAssetStoragePath":"/var/preview/assets",
      "nodeId":"E876F147E331",
      "workDispatcherEnabled":true
   },
   "http":{
      "listen":":8080"
   },
   "storage":{
      "engine":"memory"
   },
   "renderAgents": {
      "documentRenderAgent":{
         "enabled":true,
         "count":16,
         "supportedFileTypes":["doc", "docx", "ppt", "pptx"],
         "rendererParams":{
             "basePath":"/var/preview/tmp/document"
         }
      },
      "videoRenderAgent":{
         "enabled":false,
         "count":16,
         "supportedFileTypes":["mp4"],
         "engine":"zencoder",
         "rendererParams":{
         }         
      },
      "imageMagickRenderAgent":{
         "enabled":true,
         "count":16,
         "supportedFileTypes":["jpg", "jpeg", "png", "gif", "pdf"],
         "rendererParams":{
         }
      }
   },
   "zencoder":{
      "enabled":false,
      "key":"YOUR_KEY_HERE"
   },
   "simpleApi":{
      "enabled":true,
      "baseUrl":"/api",
      "edgeBaseUrl":"http://localhost:8080"
   },
   "assetApi":{
      "enabled":true
   },
   "uploader":{
      "engine":"local"
   },
   "downloader":{
      "basePath":"/var/preview/tmp/download",
      "tramEnabled": false
   },
   "templates": [
        {
            "id":"04a2c710-8872-4c88-9c75-a67175d3a8e7",
            "renderAgent":"imageMagickRenderAgent",
            "group":"4C96",
            "attributes":{
                "width":["1040"],
                "height":["780"],
                "output":["jpg"],
                "placeholderSize":["jumbo"]
            }
        },
	{
            "id":"2eee7c27-75e2-4682-9920-9a4e14caa433",
            "renderAgent":"imageMagickRenderAgent",
            "group":"4C96",
            "attributes":{
                "width":["520"],
                "height":["390"],
                "output":["jpg"],
                "placeholderSize":["large"]
            }
        },
	{
            "id":"a89a6a0d-51d9-4d99-b278-0c5dfc538984",
            "renderAgent":"imageMagickRenderAgent",
            "group":"4C96",
            "attributes":{
                "width":["500"],
                "height":["376"],
                "output":["jpg"],
                "placeholderSize":["medium"]
            }
        },
	{
            "id":"eaa7be0e-354f-482c-ac75-75cbdafecb6e",
            "renderAgent":"imageMagickRenderAgent",
            "group":"4C96",
            "attributes":{
                "width":["250"],
                "height":["188"],
                "output":["jpg"],
                "placeholderSize":["small"]
            }
        },
	{
            "id":"9B17C6CE-7B09-4FD5-92AD-D85DD218D6D7",
            "renderAgent":"documentRenderAgent",
            "group":"A907",
            "attributes":{
                "output":["pdf"]
            }
        },
	{
            "id":"4128966B-9F69-4E56-AD5C-1FDB3C24F910",
            "renderAgent":"videoRenderAgent",
            "group":"7A96",
            "attributes":{
                "output":["m3u8"],
		"forceS3Location":["s3://YOUR_BUCKET"],
                "zencoderNotificationUrl":["http://example.com/zencoderhandler"]
            }
        }
    ]
}
```

The `BASEPATH` set is the ".cache" directory in the current working directory when the executable is run.

# Usage

Through configuration different API resources and render agents can be toggle and configured.

## Simple API

By default, the simple API resources are enabled.

## Asset API

This API set serves generated assets based on the location of the generated asset.

* If the location is "local", it will attempt to load serve the file from disk.
* If the location is HTTP, it will attempt to redirect the file.
* If the location is S3, it will attempt to cache the file locally and serve it from the cache.

## Static API

By default, the static API resources are enabled.

This API set allows placeholder images to be served from the "/static/" base URL.

## Storage

By default, the "memory" storage system is enabled. All source asset and generated asset records are lost when the process is stopped when the "memory" storage engine is used.

Alternatively, the "cassandra" engine can be enabled to persist records to Cassandra. When enabled, one or more cassandra nodes must be configured and the keyspace configured.

```cql
CREATE KEYSPACE preview WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 3 };
USE preview;
CREATE TABLE IF NOT EXISTS generated_assets (id varchar, source varchar, status varchar, template_id varchar, message blob, PRIMARY KEY (id));
CREATE TABLE IF NOT EXISTS active_generated_assets (id varchar PRIMARY KEY);
CREATE TABLE IF NOT EXISTS waiting_generated_assets (id varchar, source varchar, template varchar, state varchar, PRIMARY KEY(template, source, id));
CREATE INDEX IF NOT EXISTS ON generated_assets (source);
CREATE INDEX IF NOT EXISTS ON generated_assets (status);
CREATE INDEX IF NOT EXISTS ON generated_assets (template_id);
CREATE TABLE IF NOT EXISTS source_assets (id varchar, type varchar, message blob, PRIMARY KEY (id, type));
CREATE INDEX IF NOT EXISTS ON source_assets (type);

```

## ImageMagick Render Agent

By default, the imagemagick render agent is enabled.

To support creating images for PDF files, the `gs` application in the ghostscript package is required.

## Document Render Agent

By default, the document render agent is enabled.

This render agent will attempt to convert documents to PDF files to then have images generated from the PDF files.

This render agent requires the following executables be available on the path:

* soffice
* pdfinfo

## Video Render Agent

By default, the video render agent is disabled.

This render agent will upload videos to Zencoder to be transcoded into HLS streams, which will then be uploaded to S3.

## Uploader

By default, the "local" uploader is enabled. This uploader engine will simply copy rendered images from the temporary file/directory to the configured base path.

Alternatively, the "s3" engine can be enabled. With the key, secret, buckets and host set, rendered images will be uploaded to an S3 providing host.

## Downloader

The downloader cannot be disabled. The only configuration is the base directory in which files are downloaded from. It is important to understand how the downloader will attempt to count the number of references to a downloaded file. Once a file has been "released", temporary file manager will attempt to delete the file, freeing disk space.

## Running The Service

To run the service, execute the preview command.

    $ preview

# Contributing

1. Run `go fmt */*.go` before committing code.
2. Run the tests before pushing code to the code repository.
3. Run integration tests regularly.
4. Consider using [golint](https://github.com/golang/lint) before pushing code to the code repository.

## Unit Tests

Unit tests use the `testing` golang package.

When authoring unit tests, consider the amount of time taken to execute the test. If the test is particularly long (more than a second) then consider skipping the test when only short tests are run.

    $ go test ./...
    $ go test ./... -test.short

## Integration Tests

Integration tests are written the same way as unit tests but are skipped unless the -test.integration flag is present. Integration tests are assumed to require a development environment where Cassandra, Tram and s3ninja are running.

    $ go test ./... -test.integration -v

## Misc

Golint is a style correctness tool.

    $ go get github.com/golang/lint/golint
    $ golint .../*.go

Govet is a static code analyzer.

    $ go vet */*.go

To determine the size of the project:

    $ find . -type f -name '*.go' -exec wc -l {} \; | awk '{total += $1} END {print total}'
