name             'preview_prod'
maintainer       'Nick Gerakines'
maintainer_email 'nick@gerakines.net'
license          'MIT'
description      'Installs/Configures preview_prod'
long_description IO.read(File.join(File.dirname(__FILE__), 'README.md'))
version          '1.0.1'

depends 'tram', '~> 1.0.0'
depends 'preview', '~> 1.0.1'
depends 'cassandra', '~> 2.4.0'
depends 'haproxy', '~> 1.6.4'

supports 'centos'

recipe 'preview_prod::node', 'Configures and prepares a preview application node.'
recipe 'preview_prod::storage', 'Configures and prepares a storage node.'
recipe 'preview_prod::proxy', 'Configures and prepares haproxy.'
recipe 'preview_prod::cache', 'Configures and prepares tram.'

attribute 'preview_prod/node_id',
  :display_name => 'The id of the preview node.',
  :required => 'required',
  :type => 'string',
  :recipes => ['preview_prod::node']

attribute 'preview_prod/cassandra_hosts',
  :display_name => 'The cassandra hosts used by the preview node.',
  :required => 'required',
  :type => 'array',
  :recipes => ['preview_prod::node']

attribute 'preview_prod/edge_host',
  :display_name => 'The base url used to request assets from the cluster.',
  :required => 'required',
  :type => 'string',
  :recipes => ['preview_prod::node']

attribute 'preview_prod/s3Key',
  :display_name => 'The S3 key used to store generated assets.',
  :required => 'required',
  :type => 'string',
  :recipes => ['preview_prod::node', 'preview_prod::cache']

attribute 'preview_prod/s3Secret',
  :display_name => 'The S3 secret key used to store generated assets.',
  :required => 'required',
  :type => 'string',
  :recipes => ['preview_prod::node', 'preview_prod::cache']

attribute 'preview_prod/s3Host',
  :display_name => 'The S3 host used to store generated assets.',
  :required => 'required',
  :type => 'string',
  :recipes => ['preview_prod::node', 'preview_prod::cache']

attribute 'preview_prod/s3Buckets',
  :display_name => 'The S3 buckets used to store generated assets.',
  :required => 'required',
  :type => 'array',
  :recipes => ['preview_prod::node', 'preview_prod::cache']

attribute 'preview_prod/cacheS3Buckets',
  :display_name => 'The S3 buckets used to cache source assets.',
  :required => 'required',
  :type => 'array',
  :recipes => ['preview_prod::cache']

attribute 'preview_prod/tramHosts',
  :display_name => 'The tram hosts used to cache source assets.',
  :required => 'required',
  :type => 'array',
  :recipes => ['preview_prod::node']

attribute 'preview_prod/proxyApps',
  :display_name => 'A map of apps that the proxy distributes requests to.',
  :required => 'required',
  :type => 'hash',
  :recipes => ['preview_prod::proxy']
