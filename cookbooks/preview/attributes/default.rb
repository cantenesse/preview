
default[:preview][:platform] = 'amd64'
default[:preview][:version] = '1.1.0'
default[:preview][:install_type] = 'archive'
default[:preview][:package] = 'preview'
default[:preview][:archive_source] = "https://github.com/ngerakines/preview/releases/download/v#{node[:preview][:version]}/preview-#{node[:preview][:version]}-linux_#{node[:preview][:platform]}.zip"

default[:preview][:port] = 8080
default[:preview][:basePath] = '/home/preview/data/'

default[:preview][:enable_monit] = true
default[:preview][:enable_logrotate] = true

default[:preview][:placeholders][:install_type] = 'none'
default[:preview][:placeholders][:package] = 'preview-placeholders'
default[:preview][:placeholders][:base] = "#{node[:preview][:basePath]}placeholders"
default[:preview][:placeholders][:source_location] = nil

default[:preview][:libreoffice][:create_yum] = true
default[:preview][:libreoffice][:rpm_source] = 'http://download.documentfoundation.org/libreoffice/stable/4.2.4/rpm/x86_64/LibreOffice_4.2.4_Linux_x86-64_rpm.tar.gz'

default[:preview][:config] = {}

default[:preview][:config][:common] = {}
default[:preview][:config][:common][:placeholderBasePath] = "#{node[:preview][:basePath]}placeholders"
default[:preview][:config][:common][:placeholderGroups] = {}
default[:preview][:config][:common][:placeholderGroups][:image] = ['jpg', 'jpeg', 'png', 'gif']
default[:preview][:config][:common][:placeholderGroups][:document] = ['pdf', 'doc', 'docx']
default[:preview][:config][:common][:placeholderGroups][:presentation] = ['ppt', 'pptx']
default[:preview][:config][:common][:localAssetStoragePath] = "#{node[:preview][:basePath]}assets"
default[:preview][:config][:common][:nodeId] = "E876F147E331"
default[:preview][:config][:common][:workDispatcherEnabled] = true

default[:preview][:config][:http] = {}
default[:preview][:config][:http][:listen] = ":#{node[:preview][:port]}"

default[:preview][:config][:storage] = {}
default[:preview][:config][:storage][:engine] = "memory"

default[:preview][:config][:documentRenderAgent] = {}
default[:preview][:config][:documentRenderAgent][:enabled] = true
default[:preview][:config][:documentRenderAgent][:count] = 8
default[:preview][:config][:documentRenderAgent][:basePath] = "#{node[:preview][:basePath]}tmp/documentRenderAgent/"
default[:preview][:config][:documentRenderAgent][:supportedFileTypes] = ["docx", "pptx", "doc", "ppt"]

default[:preview][:config][:imageMagickRenderAgent] = {}
default[:preview][:config][:imageMagickRenderAgent][:enabled] = true
default[:preview][:config][:imageMagickRenderAgent][:count] = 8
default[:preview][:config][:imageMagickRenderAgent][:supportedFileTypes] = ["jpg", "jpeg", "png", "gif", "pdf"]

default[:preview][:config][:videoRenderAgent] = {}
default[:preview][:config][:videoRenderAgent][:enabled] = false
default[:preview][:config][:videoRenderAgent][:count] = 8
default[:preview][:config][:videoRenderAgent][:engine] = "zencoder"
default[:preview][:config][:videoRenderAgent][:zencoderKey] = ""
default[:preview][:config][:videoRenderAgent][:zencoderNotificationUrl] = ""
default[:preview][:config][:videoRenderAgent][:zencoderS3Bucket] = ""
default[:preview][:config][:videoRenderAgent][:supportedFileTypes] = ["mp4", "mov"]

default[:preview][:config][:simpleApi] = {}
default[:preview][:config][:simpleApi][:enabled] = true
default[:preview][:config][:simpleApi][:baseUrl] = "/api"
default[:preview][:config][:simpleApi][:edgeBaseUrl] = "http://localhost:#{node[:preview][:port]}"

default[:preview][:config][:assetApi] = {}
default[:preview][:config][:assetApi][:enabled] = true

default[:preview][:config][:uploader] = {}
default[:preview][:config][:uploader][:engine] = "local"

default[:preview][:config][:downloader] = {}
default[:preview][:config][:downloader][:basePath] = "#{node[:preview][:basePath]}tmp/downloads/"
default[:preview][:config][:downloader][:tramEnabled] = false
