# typed: strict
# frozen_string_literal: true

class Buttons::FollowButtonComponent < ApplicationComponent
  sig { params(target_profile: Profile).void }
  def initialize(target_profile:)
    @target_profile = target_profile
  end
end
