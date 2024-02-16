# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for dynamic methods in `ViewComponentsSystemTestController`.
# Please instead update this file by running `bin/tapioca dsl ViewComponentsSystemTestController`.

class ViewComponentsSystemTestController
  include GeneratedUrlHelpersModule
  include GeneratedPathHelpersModule

  sig { returns(HelperProxy) }
  def helpers; end

  module HelperMethods
    include ::ActionController::Base::HelperMethods
    include ::ApplicationHelper
    include ::ComponentDataFetcherHelper
    include ::FlashToastHelper
    include ::LanguageHelper
    include ::ProfilesHelper
    include ::TextHelper
    include ::TimeHelper
    include ::PreviewHelper
    include ::Doorkeeper::DashboardHelper
  end

  class HelperProxy < ::ActionView::Base
    include HelperMethods
  end
end
