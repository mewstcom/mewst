# typed: strict
# frozen_string_literal: true

class FollowForm < ApplicationForm
  attribute :source_idname, :string
  attribute :target_idname, :string

  validates :source_profile, presence: true
  validates :target_profile, presence: true
  validate :target_profile_difference

  sig { returns(T.nilable(Profile)) }
  def source_profile
    @source_profile ||= T.let(Profile.only_kept.find_by(idname: source_idname), T.nilable(Profile))
  end

  sig { returns(T.nilable(Profile)) }
  def target_profile
    @target_profile ||= T.let(Profile.only_kept.find_by(idname: target_idname), T.nilable(Profile))
  end

  sig { returns(T.untyped) }
  private def target_profile_difference
    if source_profile == target_profile
      errors.add(:base, :target_profile_difference)
    end
  end
end
