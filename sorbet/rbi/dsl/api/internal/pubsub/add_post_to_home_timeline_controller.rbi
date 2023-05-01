# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for dynamic methods in `Api::Internal::Pubsub::AddPostToHomeTimelineController`.
# Please instead update this file by running `bin/tapioca dsl Api::Internal::Pubsub::AddPostToHomeTimelineController`.

class Api::Internal::Pubsub::AddPostToHomeTimelineController
  sig { returns(HelperProxy) }
  def helpers; end

  module HelperMethods
    include ::ActionController::Base::HelperMethods
    include ::ApplicationHelper
    include ::ComponentDataFetcherHelper
    include ::LanguageHelper
    include ::TextHelper
    include ::TimeHelper
    include ::PreviewHelper
  end

  class HelperProxy < ::ActionView::Base
    include HelperMethods
  end
end