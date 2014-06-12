#
# Cookbook Name:: preview_prod
# Recipe:: proxy
#
# Copyright (C) 2014 Nick Gerakines
# 
# Permission is hereby granted, free of charge, to any person obtaining
# a copy of this software and associated documentation files (the
# "Software"), to deal in the Software without restriction, including
# without limitation the rights to use, copy, modify, merge, publish,
# distribute, sublicense, and/or sell copies of the Software, and to
# permit persons to whom the Software is furnished to do so, subject to
# the following conditions:
# 
# The above copyright notice and this permission notice shall be
# included in all copies or substantial portions of the Software.
# 
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
# EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
# MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
# NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
# LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
# OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
# WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
#

haproxy_lb 'preview_backend' do
	type 'backend'
	mode 'http'
	servers node[:preview_prod][:proxyApps].to_a.map {|kv| "#{kv[0]} #{kv[1]}:#{node[:preview][:port]} weight 1 check" }
	params([
		"option httpchk GET /admin/metrics",
		"option httplog"
	])
end

haproxy_lb 'preview_frontend' do
	type 'frontend'
	bind "0.0.0.0:#{node[:preview_prod][:proxy_port]}"
	mode 'http'
	params({
		'default_backend' => 'preview_backend',
		'option' => 'contstats'
		})
end

node.override['haproxy']['enable_default_http'] = false

include_recipe 'haproxy::default'
