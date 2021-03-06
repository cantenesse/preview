#
# Cookbook Name:: preview_prod
# Recipe:: cache
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

node.override[:tram][:config][:storage][:engine] = 's3'
node.override[:tram][:config][:storage][:s3Key] = node[:preview_prod][:s3Key]
node.override[:tram][:config][:storage][:s3Secret] = node[:preview_prod][:s3Secret]
node.override[:tram][:config][:storage][:s3Host] = node[:preview_prod][:s3Host]
node.override[:tram][:config][:storage][:s3Buckets] = node[:preview_prod][:cacheS3Buckets]
node.override[:tram][:package_source] = "https://github.com/ngerakines/tram/releases/download/1.0.0/tram-1.0.0.linux_amd64.zip"

include_recipe 'tram::default'
