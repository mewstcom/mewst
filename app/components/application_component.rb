# typed: strict
# frozen_string_literal: true

class ApplicationComponent < ViewComponent::Base
  extend T::Sig

  delegate :current_profile, :mst_image_url, :signed_in?, to: :helpers
end
