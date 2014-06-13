name             'preview_test'
maintainer       'Nick Gerakines'
maintainer_email 'nick@gerakines.net'
license          'MIT'
description      'Installs/Configures preview_test'
long_description IO.read(File.join(File.dirname(__FILE__), 'README.md'))
version          '1.0.0'

depends 'yum'
depends 'yum-epel'
depends 'java'
depends 'cassandra', '~> 2.4.0'
depends 'mysql', '~> 5.2.12'
depends 'monit', '~> 1.5.3'
depends 'preview_build', '~> 1.0.0'

supports 'centos'
