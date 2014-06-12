require 'chefspec'
require 'chefspec/berkshelf'
ChefSpec::Coverage.start!

platforms = {
  'centos' => ['5.9']
}

describe 'preview::app' do

  platforms.each do |platform_name, platform_versions|

    platform_versions.each do |platform_version|

      context "no install type on #{platform_name} #{platform_version}" do

        let(:chef_run) do
          ChefSpec::Runner.new(platform: platform_name, version: platform_version) do |node|
            node.set['preview']['install_type'] = 'none'
          end.converge('preview::app')
        end

        it 'includes dependent receipes' do
          expect(chef_run).to include_recipe('apt::default')
          expect(chef_run).to include_recipe('yum::default')
          expect(chef_run).to include_recipe('monit::default')
          expect(chef_run).to include_recipe('logrotate::default')
          expect(chef_run).to include_recipe('preview::_common')
          expect(chef_run).to include_recipe('preview::_libreoffice')
          expect(chef_run).to include_recipe('preview::placeholders')
        end

        it 'places configuration' do
          expect(chef_run).to create_template('/etc/preview.conf')
        end

        it 'installs required packages' do
          expect(chef_run).to install_package('unzip')
          expect(chef_run).to install_package('curl')
          expect(chef_run).to install_package('ImageMagick')
          expect(chef_run).to install_package('poppler-utils')
        end

        it 'prepares the preview service' do
          expect(chef_run).to create_cookbook_file('/etc/init.d/preview')
        end

      end

      context "package install type on #{platform_name} #{platform_version}" do

        let(:chef_run) do
          ChefSpec::Runner.new(platform: platform_name, version: platform_version) do |node|
            node.set['preview']['install_type'] = 'package'
          end.converge('preview::app')
        end

        it 'installs the preview package' do
          expect(chef_run).to install_package('preview')
        end

      end

    end

  end

end
