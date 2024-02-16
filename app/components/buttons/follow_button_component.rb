# typed: strict
# frozen_string_literal: true

class Buttons::FollowButtonComponent < ApplicationComponent
  sig { params(target_profile: Profile, follow_checker: FollowChecker).void }
  def initialize(target_profile:, follow_checker:)
    @target_profile = target_profile
    @follow_checker = follow_checker
  end

  sig { returns(Profile) }
  attr_reader :target_profile
  private :target_profile

  sig { returns(FollowChecker) }
  attr_reader :follow_checker
  private :follow_checker
end
