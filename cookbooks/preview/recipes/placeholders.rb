#
# Cookbook Name:: preview
# Recipe:: placeholders
#
# Copyright (C) 2014 Nick Gerakines <nick@gerakines.net>
# 
# This project and its contents are open source under the MIT license.
#

include_recipe 'apt::default'
include_recipe 'yum::default'
include_recipe 'preview::_common'

execute 'set-placeholder-directory-permissions' do
 command "chown -R preview:preview {node[:preview][:placeholders][:base]}"
 action :nothing
end

case node[:preview][:placeholders][:install_type]
when 'package'
  package node[:preview][:placeholders][:package] do
    notifies :run, "execute[set-placeholder-directory-permissions]", :immediately
  end

when 'source'
  package 'unzip'

  directory node[:preview][:placeholders][:base] do
    owner 'preview'
    group 'preview'
    recursive true
    mode 00644
    action :create
  end

  remote_file "#{Chef::Config[:file_cache_path]}/placeholders.zip" do
    source node[:preview][:placeholders][:source_location]
  end

  bash 'extract preview placeholders' do
    code <<-EOH
      unzip -o "#{Chef::Config[:file_cache_path]}/placeholders.zip" -d #{node[:preview][:placeholders][:base]}
      EOH
    notifies :run, "execute[set-placeholder-directory-permissions]", :immediately
  end

end
