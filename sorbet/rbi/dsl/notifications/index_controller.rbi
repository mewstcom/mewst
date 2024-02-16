# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for dynamic methods in `Notifications::IndexController`.
# Please instead update this file by running `bin/tapioca dsl Notifications::IndexController`.

class Notifications::IndexController
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
    include ::ApplicationController::HelperMethods

    sig { returns(T.nilable(::Actor)) }
    def current_actor; end

    sig { returns(::Actor) }
    def current_actor!; end

    sig { returns(T::Boolean) }
    def signed_in?; end
  end

  class HelperProxy < ::ActionView::Base
    include HelperMethods
  end
end
