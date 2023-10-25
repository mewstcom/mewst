# typed: strict
# frozen_string_literal: true

class Latest::FollowForm < Latest::ApplicationForm
  sig { returns(T.nilable(Actor)) }
  attr_accessor :viewer

  attribute :target_atname, :string

  validates :viewer, presence: true
  validates :target_profile, presence: true
  validate :target_profile_is_not_me

  sig { returns(T.nilable(Profile)) }
  def target_profile
    Profile.kept.find_by(atname: target_atname)
  end

  sig { void }
  private def target_profile_is_not_me
    return unless viewer.not_nil!.me?(target_profile: target_profile.not_nil!)

    errors.add(:target_atname, :cannot_follow_myself)
  end
end
