# typed: strict
# frozen_string_literal: true

class FollowForm < ApplicationForm
  sig { returns(T.nilable(Profile)) }
  attr_accessor :profile

  attribute :target_atname, :string

  validates :profile, presence: true
  validates :target_profile, presence: true
  validate :target_profile_is_not_me

  sig { returns(T.nilable(Profile)) }
  def target_profile
    @target_profile ||= T.let(Profile.kept.find_by(atname: target_atname), T.nilable(Profile))
  end

  sig { void }
  private def target_profile_is_not_me
    return unless profile.not_nil!.me?(target_profile: target_profile.not_nil!)

    errors.add(:target_atname, :cannot_follow_myself)
  end
end
