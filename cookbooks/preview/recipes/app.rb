#
# Cookbook Name:: preview
# Recipe:: app
#
# Copyright (C) 2014 Nick Gerakines <nick@gerakines.net>
# 
# This project and its contents are open source under the MIT license.
#

node.set["monit"]["reload_on_change"] = false

include_recipe 'apt::default'
include_recipe 'yum::default'

if node[:preview][:enable_monit] then
  include_recipe 'monit::default'
end

if node[:preview][:enable_logrotate] then
  include_recipe 'logrotate::default'
end

include_recipe 'preview::_common'
include_recipe 'preview::_libreoffice'
include_recipe 'preview::placeholders'

require 'json'

preview_packages = %w{unzip curl ImageMagick poppler-utils}

preview_packages.each do |pkg|
  package pkg do
    action :install
  end
end

template '/etc/preview.conf' do
  source 'preview.conf.erb'
  mode 0640
  group 'preview'
  owner 'preview'
  variables(:json => JSON.pretty_generate(node[:preview][:config].to_hash))
end

case node[:preview][:install_type]
when 'package'
  package node[:preview][:package]

when 'archive'
  remote_file "#{Chef::Config[:file_cache_path]}/preview.zip" do
    source node[:preview][:archive_source]
  end

  bash 'extract_app' do
    cwd '/home/preview/'
    code <<-EOH
    unzip #{Chef::Config[:file_cache_path]}/preview.zip
    EOH
    not_if { ::File.exists?('/home/preview/preview') }
  end

  execute 'chown -R preview:preview /home/preview/'

  file '/home/preview/preview' do
    mode 00777
  end
end

cookbook_file '/etc/init.d/preview' do
  source 'preview'
  mode 00777
  owner 'root'
  group 'root'
end

service 'preview' do
  provider Chef::Provider::Service::Init
  action [:start]
end

if node[:preview][:enable_monit] then
  monit_monitrc 'preview' do
    variables({ category: 'preview' })
  end
end

if node[:preview][:enable_logrotate] then
  logrotate_app 'preview' do
    cookbook  'logrotate'
    path      ['/var/log/preview.log']
    options   ['missingok', 'delaycompress', 'copytruncate']
    frequency 'daily'
    size      1048576
    maxsize   2097152
    rotate    2
    create    '644 root root'
  end
end
