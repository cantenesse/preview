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

include_recipe 'haproxy::default'

## NKG: At some point, this should be updated to either reference attribuites, serf/consul or nodes in an env that have preview_prod::node in their runlist.
## http://docs.opscode.com/essentials_search.html
# preview_nodes = partial_search('node', "role:preview_node AND chef_environment:#{node.chef_environment}", :keys => {'name' => ['name'], 'ip'   => ['ipaddress'],}) || []

haproxy_lb 'preview' do
  bind '0.0.0.0:80'
  mode 'http'
  {'node1' => '127.0.0.1' }.each do |name, ip|
    "#{name} #{ip}:#{node[:preview][:port]}"
  end
  params({
    'balance' => 'roundrobin'
  })
end
