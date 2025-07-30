# typed: strict
# frozen_string_literal: true

class Buttons::CheckSuggestedProfileButtonComponent < ApplicationComponent
  sig { params(target_profile: ProfileRecord).void }
  def initialize(target_profile:)
    @target_profile = target_profile
  end

  sig { returns(ProfileRecord) }
  attr_reader :target_profile
  private :target_profile
end
