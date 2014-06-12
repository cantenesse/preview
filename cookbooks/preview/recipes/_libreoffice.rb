#
# Cookbook Name:: preview
# Recipe:: _libreoffice
#
# Copyright (C) 2014 Nick Gerakines <nick@gerakines.net>
# 
# This project and its contents are open source under the MIT license.
#

include_recipe 'yum::default'

if platform?("redhat", "centos", "fedora")

	if node[:preview][:libreoffice][:create_yum]

		remote_file "#{Chef::Config[:file_cache_path]}/LibreOffice_4.2.4_Linux_x86-64_rpm.tar.gz" do
			source node[:preview][:libreoffice][:rpm_source]
		end

		directory '/opt/yum/libreoffice/' do
			owner 'root'
			group 'root'
			recursive true
			mode 00644
			action :create
		end

		package 'createrepo'

		bash 'unpack libreoffice' do
			cwd '/opt/yum/libreoffice/'
			code <<-EOH
			tar zxvf #{Chef::Config[:file_cache_path]}/LibreOffice_4.2.4_Linux_x86-64_rpm.tar.gz
			createrepo .
			EOH
		end

		yum_repository 'libreoffice-local' do
			description 'libreoffice-local'
			baseurl 'file:///opt/yum/libreoffice/'
			gpgcheck false
			enabled true
			action :create
		end

		execute 'yum clean all'
	end

	execute 'yum -y install libreoffice4.2*'
	execute 'yum -y install libobasis4.2*'

end

link '/usr/bin/soffice' do
	to '/opt/libreoffice4.2/program/soffice.bin'
	only_if { File.exist?('/opt/libreoffice4.2/program/soffice.bin') }
end
