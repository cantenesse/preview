
default[:preview][:platform] = 'amd64'
default[:preview][:version] = '1.2.0'
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
default[:preview][:libreoffice][:rpm_source] = 'http://download.documentfoundation.org/libreoffice/stable/4.2.5/rpm/x86_64/LibreOffice_4.2.5_Linux_x86-64_rpm.tar.gz'

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

default[:preview][:config][:renderAgents] = {}

default[:preview][:config][:renderAgents][:documentRenderAgent] = {}
default[:preview][:config][:renderAgents][:documentRenderAgent][:enabled] = true
default[:preview][:config][:renderAgents][:documentRenderAgent][:count] = 8
default[:preview][:config][:renderAgents][:documentRenderAgent][:fileTypes] = {}
default[:preview][:config][:renderAgents][:documentRenderAgent][:fileTypes][:docx] = 60
default[:preview][:config][:renderAgents][:documentRenderAgent][:fileTypes][:doc] = 60
default[:preview][:config][:renderAgents][:documentRenderAgent][:fileTypes][:pptx] = 60
default[:preview][:config][:renderAgents][:documentRenderAgent][:fileTypes][:ppt] = 60
default[:preview][:config][:renderAgents][:documentRenderAgent][:rendererParams] = {}
default[:preview][:config][:renderAgents][:documentRenderAgent][:rendererParams][:tempFileBasePath] = "#{node[:preview][:basePath]}tmp/documentRenderAgent/"

default[:preview][:config][:renderAgents][:imageMagickRenderAgent] = {}
default[:preview][:config][:renderAgents][:imageMagickRenderAgent][:enabled] = true
default[:preview][:config][:renderAgents][:imageMagickRenderAgent][:count] = 8
default[:preview][:config][:renderAgents][:imageMagickRenderAgent][:fileTypes] = {}
default[:preview][:config][:renderAgents][:imageMagickRenderAgent][:fileTypes][:jpg] = 60
default[:preview][:config][:renderAgents][:imageMagickRenderAgent][:fileTypes][:png] = 60
default[:preview][:config][:renderAgents][:imageMagickRenderAgent][:fileTypes][:gif] = 60
default[:preview][:config][:renderAgents][:imageMagickRenderAgent][:fileTypes][:pdf] = 60
default[:preview][:config][:renderAgents][:imageMagickRenderAgent][:fileTypes][:jpeg] = 60
default[:preview][:config][:renderAgents][:imageMagickRenderAgent][:rendererParams]
default[:preview][:config][:renderAgents][:imageMagickRenderAgent][:rendererParams][:maxPages] = "20"

default[:preview][:config][:renderAgents][:videoRenderAgent] = {}
default[:preview][:config][:renderAgents][:videoRenderAgent][:enabled] = false
default[:preview][:config][:renderAgents][:videoRenderAgent][:count] = 8
default[:preview][:config][:renderAgents][:videoRenderAgent][:engine] = "zencoder"
default[:preview][:config][:renderAgents][:videoRenderAgent][:fileTypes] = {}
default[:preview][:config][:renderAgents][:videoRenderAgent][:fileTypes][:mov] = 0
default[:preview][:config][:renderAgents][:videoRenderAgent][:rendererParams] = {}
default[:preview][:config][:renderAgents][:videoRenderAgent][:rendererParams][:zencoderNotificationUrl] = ""

default[:preview][:config][:simpleApi] = {}
default[:preview][:config][:simpleApi][:enabled] = true
default[:preview][:config][:simpleApi][:baseUrl] = "/api"
default[:preview][:config][:simpleApi][:edgeBaseUrl] = "http://localhost:#{node[:preview][:port]}"

default[:preview][:config][:assetApi] = {}
default[:preview][:config][:assetApi][:enabled] = true

default[:preview][:config][:uploader] = {}
default[:preview][:config][:uploader][:engine] = "local"

default[:preview][:config][:s3] = {}
default[:preview][:config][:s3][:secret] = ""
default[:preview][:config][:s3][:host] = ""
default[:preview][:config][:s3][:buckets] = [""]
default[:preview][:config][:s3][:verifySsl] = false
default[:preview][:config][:s3][:urlCompatMode] = true

default[:preview][:config][:downloader] = {}
default[:preview][:config][:downloader][:basePath] = "#{node[:preview][:basePath]}tmp/downloads/"
default[:preview][:config][:downloader][:tramEnabled] = false
