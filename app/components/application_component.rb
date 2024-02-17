# typed: strict
# frozen_string_literal: true

class ApplicationComponent < ViewComponent::Base
  extend T::Sig

  # ref: https://github.com/ViewComponent/view_component/issues/1099
  include Turbo::FramesHelper

  delegate :current_actor!, :mst_absolute_time, :mst_image_url, :mst_time_ago_in_words, :signed_in?, to: :helpers
end
