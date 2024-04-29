# typed: strict
# frozen_string_literal: true

class ApplicationComponent < ViewComponent::Base
  extend T::Sig

  delegate :mst_absolute_time, :mst_image_url, :mst_time_ago_in_words, :signed_in?, :viewer!, to: :helpers
end
