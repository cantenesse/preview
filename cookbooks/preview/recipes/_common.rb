#
# Cookbook Name:: preview
# Recipe:: _common
#
# Copyright (C) 2014 Nick Gerakines <nick@gerakines.net>
# 
# This project and its contents are open source under the MIT license.
#

user 'preview' do
  username 'preview'
  home '/home/preview'
  action :remove
  action :create
  supports ({ :manage_home => true })
end

group 'preview' do
  group_name 'preview'
  members 'preview'
  action :remove
  action :create
end
