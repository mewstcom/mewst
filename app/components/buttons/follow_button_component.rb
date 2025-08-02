# typed: strict
# frozen_string_literal: true

class Buttons::FollowButtonComponent < ApplicationComponent
  sig { params(target_profile: ProfileRecord, follow_checker: FollowChecker, class_name: String).void }
  def initialize(target_profile:, follow_checker:, class_name: "")
    @target_profile = target_profile
    @follow_checker = follow_checker
    @class_name = class_name
  end

  sig { returns(ProfileRecord) }
  attr_reader :target_profile
  private :target_profile

  sig { returns(FollowChecker) }
  attr_reader :follow_checker
  private :follow_checker

  sig { returns(String) }
  attr_reader :class_name
  private :class_name
end
