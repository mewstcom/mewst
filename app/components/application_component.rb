# typed: strict
# frozen_string_literal: true

class ApplicationComponent < ViewComponent::Base
  extend T::Sig

  delegate :current_profile, :current_profile!, :mst_image_url, :mst_time_ago_in_words, :signed_in?, to: :helpers
end
