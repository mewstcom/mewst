# typed: strict
# frozen_string_literal: true

class Buttons::FollowButtonWithAvatarComponent < ApplicationComponent
  sig { params(target_profile: Profile, follow_checker: FollowChecker, avatar_width: Integer, button_class_name: String).void }
  def initialize(target_profile:, follow_checker:, avatar_width:, button_class_name: "")
    @target_profile = target_profile
    @follow_checker = follow_checker
    @avatar_width = avatar_width
    @button_class_name = button_class_name
  end

  sig { returns(Profile) }
  attr_reader :target_profile
  private :target_profile

  sig { returns(FollowChecker) }
  attr_reader :follow_checker
  private :follow_checker

  sig { returns(Integer) }
  attr_reader :avatar_width
  private :avatar_width

  sig { returns(String) }
  attr_reader :button_class_name
  private :button_class_name
end
